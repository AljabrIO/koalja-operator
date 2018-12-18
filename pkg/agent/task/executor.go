//
// Copyright © 2018 Aljabr, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package task

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"

	"github.com/dchest/uniuri"
	"github.com/rs/zerolog"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	toolscache "k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/AljabrIO/koalja-operator/pkg/annotatedvalue"
	koalja "github.com/AljabrIO/koalja-operator/pkg/apis/koalja/v1alpha1"
	"github.com/AljabrIO/koalja-operator/pkg/constants"
	fs "github.com/AljabrIO/koalja-operator/pkg/fs/client"
	ptask "github.com/AljabrIO/koalja-operator/pkg/task"
	"github.com/AljabrIO/koalja-operator/pkg/tracking"
	"github.com/AljabrIO/koalja-operator/pkg/util"
)

// Executor describes the API implemented by a task executor
type Executor interface {
	ptask.OutputFileSystemServiceServer
	// Run the executor until the given context is canceled.
	Run(ctx context.Context) error
	// Execute on the task with the given snapshot as input.
	Execute(context.Context, *InputSnapshot) error
}

// NewExecutor initializes a new Executor.
func NewExecutor(log zerolog.Logger, client client.Client, cache cache.Cache, fileSystem fs.FileSystemClient,
	pipeline *koalja.Pipeline, taskSpec *koalja.TaskSpec, pod *corev1.Pod, taskAgentServicesPort int,
	podGC PodGarbageCollector, tf *templateFunctions, outputPublisher OutputPublisher, statistics *tracking.TaskStatistics) (Executor, error) {
	// Get output addresses
	outputAddressesMap := make(map[string][]string)
	for _, tos := range taskSpec.Outputs {
		annKey := constants.CreateOutputLinkAddressesAnnotationName(tos.Name)
		addresses := pod.GetAnnotations()[annKey]
		if addresses == "" {
			return nil, fmt.Errorf("No output addresses annotation found for output '%s'", tos.Name)
		}
		outputAddressesMap[tos.Name] = strings.Split(addresses, ",")
	}
	// Get my own DNS address
	dnsName, err := constants.GetDNSName()
	if err != nil {
		return nil, maskAny(err)
	}
	// Get ServiceAccount for task executors
	executorServiceName, err := constants.GetServiceAccountName()
	if err != nil {
		return nil, maskAny(err)
	}
	// Get task executor annotation (if any)
	annTaskExecutorContainer := pod.GetAnnotations()[constants.AnnTaskExecutorContainer]
	var taskExecContainer *corev1.Container
	if annTaskExecutorContainer != "" {
		var c corev1.Container
		if err := json.Unmarshal([]byte(annTaskExecutorContainer), &c); err != nil {
			return nil, maskAny(err)
		}
		taskExecContainer = &c
	}
	// Get task labels annotation (if any)
	annTaskExecutorLabels := pod.GetAnnotations()[constants.AnnTaskExecutorLabels]
	var taskExecLabels map[string]string
	if annTaskExecutorLabels != "" {
		if err := json.Unmarshal([]byte(annTaskExecutorLabels), &taskExecLabels); err != nil {
			return nil, maskAny(err)
		}
	}

	return &executor{
		Client:                         client,
		Cache:                          cache,
		FileSystemClient:               fileSystem,
		log:                            log,
		taskSpec:                       taskSpec,
		pipeline:                       pipeline,
		outputAddressesMap:             outputAddressesMap,
		taskExecContainer:              taskExecContainer,
		taskExecLabels:                 taskExecLabels,
		executorServiceName:            executorServiceName,
		myPod:                          pod,
		dnsName:                        dnsName,
		namespace:                      pod.GetNamespace(),
		taskAgentServicesPort:          taskAgentServicesPort,
		podChanges:                     make(chan *corev1.Pod, changeQueueSize),
		tf:                             tf,
		outputPublisher:                outputPublisher,
		podGC:                          podGC,
		statistics:                     statistics,
		createFileSystemURIRequestChan: make(chan createFileSystemURIRequest),
	}, nil
}

type executor struct {
	client.Client
	cache.Cache
	fs.FileSystemClient
	mutex                          sync.Mutex
	log                            zerolog.Logger
	taskSpec                       *koalja.TaskSpec
	pipeline                       *koalja.Pipeline
	outputAddressesMap             map[string][]string
	taskExecContainer              *corev1.Container
	taskExecLabels                 map[string]string
	executorServiceName            string
	myPod                          *corev1.Pod
	dnsName                        string
	namespace                      string
	taskAgentServicesPort          int
	podChanges                     chan *corev1.Pod
	tf                             *templateFunctions
	outputPublisher                OutputPublisher
	podGC                          PodGarbageCollector
	statistics                     *tracking.TaskStatistics
	createFileSystemURIRequestChan chan createFileSystemURIRequest
}

type createFileSystemURIRequest struct {
	// Name of the task output that is data belongs to.
	OutputName string
	// Local path of the file/dir in the Volume
	LocalPath string
	// IsDir indicates if the URI is for a file (false) or a directory (true)
	IsDir bool

	// Response channel
	Response chan createFileSystemURIResponse
}

type createFileSystemURIResponse struct {
	// URI holds the created URI
	URI string
	// Error in case something went wrong
	Error error
}

const (
	changeQueueSize = 32
)

var (
	gvkPod = corev1.SchemeGroupVersion.WithKind("Pod")
)

// Run the executor until the given context is canceled.
func (e *executor) Run(ctx context.Context) error {
	defer close(e.podChanges)

	// Watch pods
	informer, err := e.Cache.GetInformerForKind(gvkPod)
	if err != nil {
		return maskAny(err)
	}

	getPod := func(obj interface{}) (*corev1.Pod, bool) {
		pod, ok := obj.(*corev1.Pod)
		if !ok {
			tombstone, ok := obj.(toolscache.DeletedFinalStateUnknown)
			if !ok {
				return nil, false
			}
			pod, ok = tombstone.Obj.(*corev1.Pod)
			return pod, ok
		}
		return pod, true
	}

	informer.AddEventHandler(toolscache.ResourceEventHandlerFuncs{
		UpdateFunc: func(oldObj, newObj interface{}) {
			if pod, ok := getPod(newObj); ok {
				e.log.Debug().Str("pod-name", pod.GetName()).Msg("pod updated notification")
				e.podChanges <- pod
			}
		},
		DeleteFunc: func(obj interface{}) {
			if pod, ok := getPod(obj); ok {
				e.log.Debug().Str("pod-name", pod.GetName()).Msg("pod deleted notification")
				e.podChanges <- pod
			}
		},
	})
	informer.Run(ctx.Done())

	return nil
}

// Execute on the task with the given snapshot as input.
func (e *executor) Execute(ctx context.Context, args *InputSnapshot) error {
	// Define the pod
	uid := uniuri.NewLen(6)
	podName := util.FixupKubernetesName(fmt.Sprintf("%s-x-%s-%s", e.pipeline.GetName(), e.taskSpec.Name, uid))
	execCont, err := e.createExecContainer()
	if err != nil {
		return maskAny(err)
	}
	if execCont.Name == "" {
		execCont.Name = "executor"
	}
	ownerRef := *metav1.NewControllerRef(e.myPod, gvkPod)
	podLabels := util.CloneLabels(e.taskExecLabels)
	podLabels["pipeline"] = e.pipeline.GetName()
	podLabels["task"] = e.taskSpec.Name
	// Prepare labels for Pod GC
	podGCLabels := util.CloneLabels(podLabels)
	// Prepare pod
	podLabels["uid"] = uid // UID is unique per pod and therefore not part of the GC selector
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:            podName,
			Namespace:       e.namespace,
			OwnerReferences: []metav1.OwnerReference{ownerRef},
			Labels:          podLabels,
		},
		Spec: corev1.PodSpec{
			Containers:         []corev1.Container{*execCont},
			RestartPolicy:      corev1.RestartPolicyNever,
			ServiceAccountName: e.executorServiceName,
		},
	}
	//util.SetIstioPodAnnotations(&pod.ObjectMeta, true)
	resources, outputProcessors, err := e.configureExecContainer(ctx, args, &pod.Spec.Containers[0], pod, ownerRef)
	defer func() {
		// Cleanup resources
		for _, r := range resources {
			e.log.Debug().
				Str("kind", r.GetObjectKind().GroupVersionKind().Kind).
				Msg("deleting resource")
			if err := e.Client.Delete(ctx, r); err != nil {
				e.log.Error().Err(err).Msg("Failed to delete resource")
			}
		}
	}()
	if err != nil {
		return maskAny(err)
	}

	// Prepare service proxy
	proxyConfigMapName, err := e.ensureProxyConfigMap(ctx, ownerRef)
	if err != nil {
		e.log.Error().Err(err).Msg("Failed to create proxy config")
		return maskAny(err)
	}
	proxyContainer, proxyVols, err := e.createProxyContainer(ctx, proxyConfigMapName)
	if err != nil {
		e.log.Error().Err(err).Msg("Failed to create proxy container")
		return maskAny(err)
	}
	if proxyContainer != nil {
		pod.Spec.Containers = append(pod.Spec.Containers, *proxyContainer)
		pod.Spec.Volumes = append(pod.Spec.Volumes, proxyVols...)
	}

	// Launch the pod
	if err := e.Client.Create(ctx, pod); err != nil {
		e.log.Error().Err(err).Msg("Failed to create execution pod")
		return maskAny(err)
	}

	// Prepare garbage collector
	e.podGC.Set(podGCLabels, pod.Namespace)

	// Prepare for output
	cfg := ExecutorOutputProcessorConfig{
		Snapshot: args,
	}
	deps := ExecutorOutputProcessorDependencies{
		Log:             e.log,
		Client:          e.Client,
		FileSystem:      e.FileSystemClient,
		OutputPublisher: e.outputPublisher,
	}
	createFileSystemURIRequestHandler := func(req createFileSystemURIRequest) {
		defer close(req.Response)
		log := e.log.With().
			Str("outputName", req.OutputName).
			Str("localPath", req.LocalPath).
			Bool("isDir", req.IsDir).
			Logger()
		log.Debug().Msg("handling createFileSystemURIRequest")

		// Find output processor
		var outputProcessor ExecutorOutputProcessor
		for _, op := range outputProcessors {
			if op.GetOutputName() == req.OutputName {
				outputProcessor = op
				break
			}
			log.Debug().
				Str("currentOutputProcessor", op.GetOutputName()).
				Msg("This output processor is not the one I'm looking for")
		}
		if outputProcessor == nil {
			log.Debug().Msg("No such output")
			req.Response <- createFileSystemURIResponse{
				Error: fmt.Errorf("Unknown output %s", req.OutputName),
			}
		} else {
			uri, err := outputProcessor.CreateFileURI(ctx, req.LocalPath, req.IsDir, cfg, deps)
			if err != nil {
				log.Debug().Err(err).Msg("CreateFileURI failed")
			}
			req.Response <- createFileSystemURIResponse{
				URI:   uri,
				Error: maskAny(err),
			}
		}
	}

	// Wait for the pod to finish
waitLoop:
	for {
		select {
		case req := <-e.createFileSystemURIRequestChan:
			createFileSystemURIRequestHandler(req)
		case change := <-e.podChanges:
			// Match pod name
			if change.GetName() != pod.Name {
				continue
			}
			// Match pod UID (in label)
			if lbls := change.GetLabels(); lbls != nil {
				if changeUID := lbls["uid"]; changeUID != uid {
					continue
				}
			}

			switch change.Status.Phase {
			case corev1.PodSucceeded:
				// We're done with a valid result
				e.log.Debug().Msg("pod succeeded")
				break waitLoop
			case corev1.PodFailed:
				// Pod has failed
				e.log.Warn().Msg("Pod failed")
				return fmt.Errorf("Pod has failed")
			}
			// Check executor container status
			for _, cs := range change.Status.ContainerStatuses {
				if cs.Name == execCont.Name {
					if cs.State.Terminated != nil {
						switch cs.State.Terminated.ExitCode {
						case 0:
							e.log.Debug().Msg("executor container succeeded")
							break waitLoop
						default:
							e.log.Warn().Int32("exitCode", cs.State.Terminated.ExitCode).Msg("Executor container failed")
							return fmt.Errorf("Executor container has failed")
						}
					}
				}
			}
		case <-ctx.Done():
			// Context canceled
			e.log.Debug().Msg("Context canceled, deleting pod")
			if err := e.Client.Delete(ctx, pod); err != nil {
				e.log.Error().Str("pod", pod.Name).Err(err).Msg("Failed to delete pod")
			}
			return maskAny(ctx.Err())
		}
	}

	// Process results
	for _, p := range outputProcessors {
		if err := p.Process(ctx, cfg, deps); err != nil {
			e.log.Error().Err(err).Msg("Failed to process output")
			return maskAny(err)
		}
	}

	// Remove pod
	if err := e.Client.Delete(ctx, pod); err != nil {
		e.log.Error().Str("pod", pod.Name).Err(err).Msg("Failed to delete pod")
	}

	return nil
}

// CreateFileURI creates a URI for the given file/dir
func (e *executor) CreateFileURI(ctx context.Context, in *ptask.CreateFileURIRequest) (*ptask.CreateFileURIResponse, error) {
	respChan := make(chan createFileSystemURIResponse, 1)
	req := createFileSystemURIRequest{
		OutputName: in.GetOutputName(),
		LocalPath:  in.GetLocalPath(),
		IsDir:      in.GetIsDir(),
		Response:   respChan,
	}
	// Try to send request
	select {
	case e.createFileSystemURIRequestChan <- req:
		// Request was queued
	case <-ctx.Done():
		// Context was canceled
		close(respChan)
		return nil, ctx.Err()
	}

	// Wait for response
	// Note: respChan is closed by receiver of e.createFileSystemURIRequestChan
	select {
	case resp := <-respChan:
		// Response received
		if resp.Error != nil {
			return nil, maskAny(resp.Error)
		}
		return &ptask.CreateFileURIResponse{
			URI: resp.URI,
		}, nil
	case <-ctx.Done():
		// Context was canceled
		return nil, ctx.Err()
	}
}

// createExecContainer creates a container to execute
func (e *executor) createExecContainer() (*corev1.Container, error) {
	if e.taskSpec.Executor != nil {
		// Use specified executor container
		return e.taskSpec.Executor.DeepCopy(), nil
	} else if e.taskExecContainer != nil {
		// Use container from annotation
		return e.taskExecContainer.DeepCopy(), nil
	} else {
		return nil, fmt.Errorf("No executor container specified")
	}
}

// createExecContainer creates a container to execute
func (e *executor) configureExecContainer(ctx context.Context, args *InputSnapshot, c *corev1.Container, pod *corev1.Pod, ownerRef metav1.OwnerReference) ([]runtime.Object, []ExecutorOutputProcessor, error) {
	// Execute argument templates
	inputs := make(map[string]interface{})
	var nodeName *string
	var resources []runtime.Object
	var outputProcessors []ExecutorOutputProcessor
	snapshotInputs := e.taskSpec.SnapshotInputs()
	for _, tis := range snapshotInputs {
		target := &ExecutorInputBuilderTarget{
			Container: c,
			Pod:       pod,
			NodeName:  nodeName,
		}
		if err := e.buildTaskInput(ctx, tis, args.GetSequence(tis.Name), ownerRef, target); err != nil {
			return resources, outputProcessors, maskAny(err)
		}
		if tis.GetMaxSequenceLength() > 1 {
			// Sequence with more than 1 annotated value is allowed.
			// Template uses sequence
			inputs[tis.Name] = target.TemplateData
		} else {
			// Only 1 annotated value at a time allowed.
			// Template uses this value
			if templateData := target.TemplateData; len(templateData) > 0 {
				inputs[tis.Name] = templateData[0]
			}
		}
		nodeName = target.NodeName
		resources = append(resources, target.Resources...)
	}
	outputs := make(map[string]interface{})
	for _, tos := range e.taskSpec.Outputs {
		target := &ExecutorOutputBuilderTarget{
			Container: c,
			Pod:       pod,
			NodeName:  nodeName,
		}
		if err := e.buildTaskOutput(ctx, tos, ownerRef, target); err != nil {
			return resources, outputProcessors, maskAny(err)
		}
		outputs[tos.Name] = target.TemplateData
		nodeName = target.NodeName
		resources = append(resources, target.Resources...)
		if target.OutputProcessor != nil {
			outputProcessors = append(outputProcessors, target.OutputProcessor)
		}
	}
	data := BuildPipelineDataMap(e.pipeline)
	data["task"] = e.taskSpec
	data["inputs"] = inputs
	data["outputs"] = outputs

	// Apply template on command
	for i, source := range c.Command {
		var err error
		c.Command[i], err = e.applyTemplate(source, data)
		if err != nil {
			return resources, outputProcessors, maskAny(err)
		}
	}
	// Apply template on arguments
	for i, source := range c.Args {
		var err error
		c.Args[i], err = e.applyTemplate(source, data)
		if err != nil {
			return resources, outputProcessors, maskAny(err)
		}
	}

	// Append container environment variables
	taskAgentServicesAddress := net.JoinHostPort(e.dnsName, strconv.Itoa(e.taskAgentServicesPort))
	c.Env = append(c.Env,
		// Pass address of OutputReadyNotifier
		corev1.EnvVar{
			Name:  constants.EnvOutputReadyNotifierAddress,
			Value: taskAgentServicesAddress,
		},
		// Pass address of OutputFileSystemService
		corev1.EnvVar{
			Name:  constants.EnvOutputFileSystemServiceAddress,
			Value: taskAgentServicesAddress,
		},
		// Pass address of Snapshot service
		corev1.EnvVar{
			Name:  constants.EnvSnapshotServiceAddress,
			Value: taskAgentServicesAddress,
		},
		// Pass address of FileSystem service
		corev1.EnvVar{
			Name:  constants.EnvFileSystemAddress,
			Value: os.Getenv(constants.EnvFileSystemAddress),
		},
		// Pass scheme of FileSystem service
		corev1.EnvVar{
			Name:  constants.EnvFileSystemScheme,
			Value: string(annotatedvalue.SchemeFile),
		},
		// Pass kubernetes namespace
		corev1.EnvVar{
			Name: constants.EnvNamespace,
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.namespace",
				},
			},
		},
		// Pass name of pod the executor is running in
		corev1.EnvVar{
			Name: constants.EnvPodName,
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		},
	)

	return resources, outputProcessors, nil
}

const (
	gobetweenConfFileName = "gobetween.toml"
)

// ensureProxyConfigMap ensures that a ConfigMap containing configuration for the
// proxy exists and it up to date.
// Returns: ConfigMap-name, error
func (e *executor) ensureProxyConfigMap(ctx context.Context, ownerRef metav1.OwnerReference) (string, error) {
	svc := e.taskSpec.Service
	if svc == nil {
		return "", nil
	}

	// Prepare config
	var config []string
	for _, sps := range svc.Ports {
		if sps.NeedsProxy() {
			config = append(config,
				fmt.Sprintf("[server.%s]", sps.Name),
				fmt.Sprintf("bind = \":%d\"", sps.Port),
				"protocol = \"tcp\"",
				fmt.Sprintf("[server.%s.discovery]", sps.Name),
				"kind = \"static\"",
				fmt.Sprintf("static_list = [\"127.0.0.1:%d weight 1\"]", sps.LocalPort),
			)
		}
	}
	if len(config) == 0 {
		// No ports need proxy
		return "", nil
	}

	// Build ConfigMap
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            e.myPod.Name,
			Namespace:       e.myPod.Namespace,
			OwnerReferences: []metav1.OwnerReference{ownerRef},
		},
		Data: map[string]string{
			gobetweenConfFileName: strings.Join(config, "\n"),
		},
	}

	// Ensure configmap exists & is up to date
	if err := util.EnsureConfigMap(ctx, e.log, e.Client, cm, "proxy config"); err != nil {
		e.log.Debug().Err(err).Msg("Failed to ensure proxy configmap")
		return "", maskAny(err)
	}

	return cm.Name, nil
}

// createProxyContainer creates a container that contains the service proxy.
func (e *executor) createProxyContainer(ctx context.Context, configMapName string) (*corev1.Container, []corev1.Volume, error) {
	svc := e.taskSpec.Service
	if svc == nil || configMapName == "" {
		return nil, nil, nil
	}
	configFileDir := "/proxy/config"
	configVolumeName := "proxy-config"
	configFilePath := path.Join(configFileDir, gobetweenConfFileName)
	c := &corev1.Container{
		Name:            "proxy",
		Image:           "yyyar/gobetween",
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command: []string{
			"gobetween",
			"from-file",
			configFilePath,
		},
		VolumeMounts: []corev1.VolumeMount{
			corev1.VolumeMount{
				Name:      configVolumeName,
				MountPath: configFileDir,
			},
		},
	}
	for _, sps := range svc.Ports {
		if sps.NeedsProxy() {
			c.Ports = append(c.Ports, corev1.ContainerPort{
				Name:          sps.Name,
				ContainerPort: sps.Port,
				Protocol:      corev1.ProtocolTCP,
			})
		}
	}

	vol := corev1.Volume{
		Name: configVolumeName,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: configMapName,
				},
			},
		},
	}

	return c, []corev1.Volume{vol}, nil
}

// buildTaskInput creates a template data element for the given input.
func (e *executor) buildTaskInput(ctx context.Context, tis koalja.TaskInputSpec, avSeq []*annotatedvalue.AnnotatedValue, ownerRef metav1.OwnerReference, target *ExecutorInputBuilderTarget) error {
	log := e.log.With().
		Str("input", tis.Name).
		Str("task", e.taskSpec.Name).
		Logger()
	log.Debug().Msg("Preparing input")
	config := ExecutorInputBuilderConfig{
		InputSpec: tis,
		TaskSpec:  *e.taskSpec,
		Pipeline:  e.pipeline,
		OwnerRef:  ownerRef,
	}
	deps := ExecutorInputBuilderDependencies{
		Log:        e.log,
		Client:     e.Client,
		FileSystem: e.FileSystemClient,
	}
	for index, av := range avSeq {
		config.AnnotatedValue = av
		config.AnnotatedValueIndex = index
		scheme := av.GetDataScheme()
		builder := GetExecutorInputBuilder(scheme)
		if builder == nil {
			return fmt.Errorf("No input builder found for scheme '%s'", scheme)
		}
		if err := builder.Build(ctx, config, deps, target); err != nil {
			return maskAny(err)
		}
	}
	return nil
}

// buildTaskOutput creates a template data element for the given input.
func (e *executor) buildTaskOutput(ctx context.Context, tos koalja.TaskOutputSpec, ownerRef metav1.OwnerReference, target *ExecutorOutputBuilderTarget) error {
	log := e.log.With().
		Str("output", tos.Name).
		Str("task", e.taskSpec.Name).
		Logger()
	log.Debug().Msg("Preparing output")
	tosType, _ := e.pipeline.Spec.TypeByName(tos.TypeRef)
	builder := GetExecutorOutputBuilder(tosType.Protocol)
	if builder == nil {
		return fmt.Errorf("No output builder found for protocol '%s'", tosType.Protocol)
	}
	config := ExecutorOutputBuilderConfig{
		OutputSpec: tos,
		TaskSpec:   *e.taskSpec,
		Pipeline:   e.pipeline,
		OwnerRef:   ownerRef,
	}
	deps := ExecutorOutputBuilderDependencies{
		Log:        e.log,
		Client:     e.Client,
		FileSystem: e.FileSystemClient,
	}
	if err := builder.Build(ctx, config, deps, target); err != nil {
		return maskAny(err)
	}
	return nil
}

// applyTemplate parses the given template source and executes it on the given data.
func (e *executor) applyTemplate(source string, data interface{}) (string, error) {
	raw, err := e.tf.ApplyTemplate(e.log, source, data)
	if err != nil {
		return "", maskAny(err)
	}
	result := string(raw)
	e.log.Debug().Str("source", source).Str("result", result).Msg("applied template")
	return result, nil
}
