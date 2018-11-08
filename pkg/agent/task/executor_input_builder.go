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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/AljabrIO/koalja-operator/pkg/annotatedvalue"
	koalja "github.com/AljabrIO/koalja-operator/pkg/apis/koalja/v1alpha1"
	fs "github.com/AljabrIO/koalja-operator/pkg/fs/client"
	"github.com/rs/zerolog"
)

// ExecutorInputBuilder contains all functions used to build up a single
// task input for a task Executor.
type ExecutorInputBuilder interface {
	Build(context.Context, ExecutorInputBuilderConfig, ExecutorInputBuilderDependencies, *ExecutorInputBuilderTarget) error
}

// ExecutorInputBuilderConfig is the input for ExecutorInputBuilder.Build.
type ExecutorInputBuilderConfig struct {
	// Input to build for
	InputSpec koalja.TaskInputSpec
	// Task (containing InputSpec) to build for
	TaskSpec koalja.TaskSpec
	// Pipeline (containing TaskSpec) to build for
	Pipeline *koalja.Pipeline
	// AnnotatedValue on input specified by InputSpec
	AnnotatedValue *annotatedvalue.AnnotatedValue
	// Owner reference for created resources
	OwnerRef metav1.OwnerReference
}

// ExecutorInputBuilderDependencies holds dependencies that are available during
// the invocation of ExecutorInputBuilder.Build.
type ExecutorInputBuilderDependencies struct {
	Log        zerolog.Logger
	Client     client.Client
	FileSystem fs.FileSystemClient
}

// ExecutorInputBuilderTarget is hold the references to where ExecutorInputBuilder.Build
// must returns its results.
type ExecutorInputBuilderTarget struct {
	// Container that will be executed. Created by Executor
	Container *corev1.Container
	// Pod that contains the container that will be executed. Created by Executor
	Pod *corev1.Pod
	// Name of the Node on which the Pod must execute (optional). Set by input builders.
	NodeName *string
	// Template data used when executing argument & command templates. Set by input builders.
	TemplateData map[string]interface{}
	// Resources created for this input. Will be removed after execution.
	Resources []runtime.Object
}

var (
	registeredExecutorInputBuilders = map[koalja.Protocol]ExecutorInputBuilder{}
)

// RegisterExecutorInputBuilder registers a builder for the given protocol.
func RegisterExecutorInputBuilder(protocol koalja.Protocol, builder ExecutorInputBuilder) {
	registeredExecutorInputBuilders[protocol] = builder
}

// GetExecutorInputBuilder returns the builder registered for the given protocol.
// Returns nil if not found.
func GetExecutorInputBuilder(protocol koalja.Protocol) ExecutorInputBuilder {
	return registeredExecutorInputBuilders[protocol]
}
