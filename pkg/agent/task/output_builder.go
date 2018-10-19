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
	"github.com/AljabrIO/koalja-operator/pkg/fs/client"
)

// ExecutorOutputBuilder contains all functions used to build up a single
// task output for a task Executor.
type ExecutorOutputBuilder interface {
	Build(ExecutorOutputBuilderConfig, ExecutorOutputBuilderDependencies, *ExecutorOutputBuilderTarget) error
}

// ExecutorOutputBuilderConfig is the input for ExecutorOutputBuilder.Build.
type ExecutorOutputBuilderConfig struct {
	OutputSpec   koalja.TaskOutputSpec
	TaskSpec     koalja.TaskSpec
	PipelineSpec koalja.PipelineSpec
}

// ExecutorOutputBuilderDependencies holds dependencies that are available during
// the invocation of ExecutorOutputBuilder.Build.
type ExecutorOutputBuilderDependencies struct {
	FileSystem client.FileSystemClient
}

// ExecutorOutputBuilderTarget is hold the references to where ExecutorOutputBuilder.Build
// must returns its results.
type ExecutorOutputBuilderTarget struct {
	Pod          *corev1.Pod
	TemplateData map[string]interface{}
}

var (
	registeredExecutorOutputBuilders = map[koalja.Protocol]ExecutorOutputBuilder{}
)

// RegisterExecutorOutputBuilder registers a builder for the given protocol.
func RegisterExecutorOutputBuilder(protocol koalja.Protocol, builder ExecutorOutputBuilder) {
	registeredExecutorOutputBuilders[protocol] = builder
}

// GetExecutorOutputBuilder returns the builder registered for the given protocol.
// Returns nil if not found.
func GetExecutorOutputBuilder(protocol koalja.Protocol) ExecutorOutputBuilder {
	return registeredExecutorOutputBuilders[protocol]
}
