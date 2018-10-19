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
	corev1 "k8s.io/api/core/v1"

	koalja "github.com/AljabrIO/koalja-operator/pkg/apis/koalja/v1alpha1"
	"github.com/AljabrIO/koalja-operator/pkg/event"
	"github.com/AljabrIO/koalja-operator/pkg/fs/client"
)

// ExecutorInputBuilder contains all functions used to build up a single
// task input for a task Executor.
type ExecutorInputBuilder interface {
	Build(ExecutorInputBuilderConfig, ExecutorInputBuilderDependencies, *ExecutorInputBuilderTarget) error
}

// ExecutorInputBuilderConfig is the input for ExecutorInputBuilder.Build.
type ExecutorInputBuilderConfig struct {
	InputSpec    koalja.TaskInputSpec
	TaskSpec     koalja.TaskSpec
	PipelineSpec koalja.PipelineSpec
	Event        *event.Event
}

// ExecutorInputBuilderDependencies holds dependencies that are available during
// the invocation of ExecutorInputBuilder.Build.
type ExecutorInputBuilderDependencies struct {
	FileSystem client.FileSystemClient
}

// ExecutorInputBuilderTarget is hold the references to where ExecutorInputBuilder.Build
// must returns its results.
type ExecutorInputBuilderTarget struct {
	Pod          *corev1.Pod
	TemplateData map[string]interface{}
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
