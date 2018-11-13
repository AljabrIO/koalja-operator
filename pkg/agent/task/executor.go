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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net"
	"os"
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
	"github.com/AljabrIO/koalja-operator/pkg/tracking"
	"github.com/AljabrIO/koalja-operator/pkg/util"
)

// Executor describes the API implemented by a task executor
type Executor interface {
	// Run the executor until the given context is canceled.
	Run(ctx context.Context) error
	// Execute on the task with the given snapshot as input.
	Execute(context.Context, *InputSnapshot) error
}

// NewExecutor initializes a new Executor.
func NewExecutor(log zerolog.Logger, client client.Client, cache cache.Cache, fileSystem fs.FileSystemClient,
	pipeline *koalja.Pipeline, taskSpec *koalja.TaskSpec, pod *corev1.Pod, outputReadyNotifierPort int,
	podGC PodGarbageCollector, outputPublisher OutputPublisher, statistics *tracking.TaskStatistics) (Executor, error) {
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
		Client:                  client,
		Cache:                   cache,
		FileSystemClient:        fileSystem,
		log:                     log,
		taskSpec:                taskSpec,
		pipeline:                pipeline,
		outputAddressesMap:      outputAddressesMap,
		taskExecContainer:       taskExecContainer,
		taskExecLabels:          taskExecLabels,
		executorServiceName:     executorServiceName,
		myPod:                   pod,
		dnsName:                 dnsName,
		namespace:               pod.GetNamespace(),
		outputReadyNotifierPort: outputReadyNotifierPort,
		podChangeQueues:         make(map[string]chan *corev1.Pod),
		outputPublisher:         outputPublisher,
		podGC:                   podGC,
		statistics:              statistics,
	}, nil
}

type executor struct {
	client.Client
	cache.Cache
	fs.FileSystemClient
	mutex                   sync.Mutex
	log                     zerolog.Logger
	taskSpec                *koalja.TaskSpec
	pipeline                *koalja.Pipeline
	outputAddressesMap      map[string][]string
	taskExecContainer       *corev1.Container
	taskExecLabels          map[string]string
	executorServiceName     string
	myPod                   *corev1.Pod
	dnsName                 string
	namespace               string
	outputReadyNotifierPort int
	podChangeQueues         map[string]chan *corev1.Pod
	outputPublisher         OutputPublisher
	podGC                   PodGarbageCollector
	statistics              *tracking.TaskStatistics
}

const (
	changeQueueSize = 32
)

var (
	gvkPod = corev1.SchemeGroupVersion.WithKind("Pod")
)

// Run the executor until the given context is canceled.
func (e *executor) Run(ctx context.Context) error {
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
				e.mutex.Lock()
				changeQueue := e.podChangeQueues[pod.GetName()]
				e.mutex.Unlock()
				if changeQueue != nil {
					e.log.Debug().Str("pod-name", pod.GetName()).Msg("pod updated notification")
					changeQueue <- pod
				}
			}
		},
		DeleteFunc: func(obj interface{}) {
			if pod, ok := getPod(obj); ok {
				e.mutex.Lock()
				changeQueue := e.podChangeQueues[pod.GetName()]
				e.mutex.Unlock()
				if changeQueue != nil {
					e.log.Debug().Str("pod-name", pod.GetName()).Msg("pod deleted notification")
					changeQueue <- pod
				}
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

	// Prepare change queue
	changeQueue := make(chan *corev1.Pod, changeQueueSize)
	defer func() {
		e.mutex.Lock()
		delete(e.podChangeQueues, pod.GetName())
		e.mutex.Unlock()
		close(changeQueue)
	}()
	e.mutex.Lock()
	e.podChangeQueues[pod.GetName()] = changeQueue
	e.mutex.Unlock()

	// Launch the pod
	if err := e.Client.Create(ctx, pod); err != nil {
		e.log.Error().Err(err).Msg("Failed to create execution pod")
		return maskAny(err)
	}

	// Prepare garbage collector
	e.podGC.Set(podGCLabels, pod.Namespace)

	// Wait for the pod to finish
waitLoop:
	for {
		change := <-changeQueue
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
	}

	// Process results
	cfg := ExecutorOutputProcessorConfig{
		Snapshot: args,
	}
	deps := ExecutorOutputProcessorDependencies{
		Log:             e.log,
		Client:          e.Client,
		FileSystem:      e.FileSystemClient,
		OutputPublisher: e.outputPublisher,
	}
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
	for _, tis := range e.taskSpec.Inputs {
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
			inputs[tis.Name] = target.TemplateData[0]
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
	data := map[string]interface{}{
		"task":    e.taskSpec,
		"inputs":  inputs,
		"outputs": outputs,
	}
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
	c.Env = append(c.Env,
		// Pass address of OutputReadyNotifier
		corev1.EnvVar{
			Name:  constants.EnvOutputReadyNotifierAddress,
			Value: net.JoinHostPort(e.dnsName, strconv.Itoa(e.outputReadyNotifierPort)),
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
	t, err := template.New("x").Parse(source)
	if err != nil {
		e.log.Debug().Err(err).Str("source", source).Msg("Failed to parse template")
		return "", maskAny(err)
	}
	w := &bytes.Buffer{}
	if err := t.Execute(w, data); err != nil {
		e.log.Debug().Err(err).Str("source", source).Msg("Failed to execute template")
		return "", maskAny(err)
	}
	return w.String(), nil
}
