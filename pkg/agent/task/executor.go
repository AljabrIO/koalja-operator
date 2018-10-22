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

	koalja "github.com/AljabrIO/koalja-operator/pkg/apis/koalja/v1alpha1"
	"github.com/AljabrIO/koalja-operator/pkg/constants"
	"github.com/AljabrIO/koalja-operator/pkg/event"
	fs "github.com/AljabrIO/koalja-operator/pkg/fs/client"
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
	pipeline *koalja.Pipeline, taskSpec *koalja.TaskSpec, pod *corev1.Pod, outputReadyNotifierPort int) (Executor, error) {
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

	return &executor{
		Client:                  client,
		Cache:                   cache,
		FileSystemClient:        fileSystem,
		log:                     log,
		taskSpec:                taskSpec,
		pipeline:                pipeline,
		outputAddressesMap:      outputAddressesMap,
		dnsName:                 dnsName,
		namespace:               pod.GetNamespace(),
		outputReadyNotifierPort: outputReadyNotifierPort,
		podChangeQueues:         make(map[string]chan *corev1.Pod),
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
	dnsName                 string
	namespace               string
	outputReadyNotifierPort int
	podChangeQueues         map[string]chan *corev1.Pod
}

const (
	changeQueueSize = 32
)

// Run the executor until the given context is canceled.
func (e *executor) Run(ctx context.Context) error {
	// Watch pods
	informer, err := e.Cache.GetInformerForKind(corev1.SchemeGroupVersion.WithKind("Pod"))
	if err != nil {
		return maskAny(err)
	}
	informer.AddEventHandler(toolscache.ResourceEventHandlerFuncs{
		UpdateFunc: func(oldObj, newObj interface{}) {
			if pod, ok := newObj.(*corev1.Pod); ok {
				e.mutex.Lock()
				changeQueue := e.podChangeQueues[pod.GetName()]
				e.mutex.Unlock()
				if changeQueue != nil {
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
	podName := util.FixupKubernetesName(e.taskSpec.Name + "-" + uniuri.NewLen(6))
	execCont, err := e.createExecContainer()
	if err != nil {
		return maskAny(err)
	}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: e.namespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{*execCont},
		},
	}
	resources, err := e.configureExecContainer(ctx, args, &pod.Spec.Containers[0], pod)
	defer func() {
		// Cleanup resources
		for _, r := range resources {
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

	// Wait for the pod to finish
waitLoop:
	for {
		change := <-changeQueue
		switch change.Status.Phase {
		case corev1.PodSucceeded:
			// We're done with a valid result
			break waitLoop
		case corev1.PodFailed:
			// Pod has failed
			return fmt.Errorf("Pod has failed")
		}
	}

	// TODO collect results

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
	} else {
		// Find container using task type.
		return nil, fmt.Errorf("TODO Not implemented yet")
	}
}

// createExecContainer creates a container to execute
func (e *executor) configureExecContainer(ctx context.Context, args *InputSnapshot, c *corev1.Container, pod *corev1.Pod) ([]runtime.Object, error) {
	// Execute argument templates
	inputs := make(map[string]interface{})
	var nodeName *string
	var resources []runtime.Object
	for _, tis := range e.taskSpec.Inputs {
		target := &ExecutorInputBuilderTarget{
			Container: c,
			Pod:       pod,
			NodeName:  nodeName,
		}
		if err := e.buildTaskInput(ctx, tis, args.Get(tis.Name), target); err != nil {
			return resources, maskAny(err)
		}
		inputs[tis.Name] = target.TemplateData
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
		if err := e.buildTaskOutput(ctx, tos, target); err != nil {
			return resources, maskAny(err)
		}
		outputs[tos.Name] = target.TemplateData
		nodeName = target.NodeName
		resources = append(resources, target.Resources...)
	}
	data := map[string]interface{}{
		"inputs":  inputs,
		"outputs": outputs,
	}
	// Apply template on command
	for i, source := range c.Command {
		var err error
		c.Command[i], err = e.applyTemplate(source, data)
		if err != nil {
			return resources, maskAny(err)
		}
	}
	// Apply template on arguments
	for i, source := range c.Args {
		var err error
		c.Args[i], err = e.applyTemplate(source, data)
		if err != nil {
			return resources, maskAny(err)
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
	)

	return resources, nil
}

// buildTaskInput creates a template data element for the given input.
func (e *executor) buildTaskInput(ctx context.Context, tis koalja.TaskInputSpec, evt *event.Event, target *ExecutorInputBuilderTarget) error {
	tisType, _ := e.pipeline.Spec.TypeByName(tis.TypeRef)
	builder := GetExecutorInputBuilder(tisType.Protocol)
	if builder == nil {
		return fmt.Errorf("No input builder found for protocol '%s'", tisType.Protocol)
	}
	config := ExecutorInputBuilderConfig{
		InputSpec: tis,
		TaskSpec:  *e.taskSpec,
		Pipeline:  e.pipeline,
		Event:     evt,
	}
	deps := ExecutorInputBuilderDependencies{
		Log:        e.log,
		Client:     e.Client,
		FileSystem: e.FileSystemClient,
	}
	if err := builder.Build(ctx, config, deps, target); err != nil {
		return maskAny(err)
	}
	return nil
}

// buildTaskOutput creates a template data element for the given input.
func (e *executor) buildTaskOutput(ctx context.Context, tos koalja.TaskOutputSpec, target *ExecutorOutputBuilderTarget) error {
	tosType, _ := e.pipeline.Spec.TypeByName(tos.TypeRef)
	builder := GetExecutorOutputBuilder(tosType.Protocol)
	if builder == nil {
		return fmt.Errorf("No output builder found for protocol '%s'", tosType.Protocol)
	}
	config := ExecutorOutputBuilderConfig{
		OutputSpec: tos,
		TaskSpec:   *e.taskSpec,
		Pipeline:   e.pipeline,
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
		return "", maskAny(err)
	}
	w := &bytes.Buffer{}
	if err := t.Execute(w, data); err != nil {
		return "", maskAny(err)
	}
	return w.String(), nil
}
