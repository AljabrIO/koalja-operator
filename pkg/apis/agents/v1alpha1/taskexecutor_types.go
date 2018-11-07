/*
Copyright 2018 Aljabr Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TaskExecutorSpec defines the desired state of TaskExecutor
type TaskExecutorSpec struct {
	// Type of task this is an executor for
	Type TaskType `json:"type"`
	// Container used to run the executor
	Container *core.Container `json:"container"`
	// HTTP routes provided by this executor
	Routes []NetworkRoute `json:"routes,omitempty"`
}

// NetworkRoute defines a single network route.
// If a task uses a task type, a Service will be created for the executor
// and all routes will be configured.
type NetworkRoute struct {
	// Name of the route
	Name string `json:"name"`
	// Port is the port on the container that this route forwards to.
	Port int `json:"port"`
	// EnableWebsockets configures websocket support on this route
	EnableWebsockets bool `json:"enableWebsockets,omitempty"`
	// Indicates that during forwarding, the matched prefix (or path) should be swapped with this value
	PrefixRewrite string `json:"prefixRewrite,omitempty"`
}

// TaskExecutorStatus defines the observed state of TaskExecutor
type TaskExecutorStatus struct {
	// Empty on purpose
}

// TaskType identifies a well know type of task.
type TaskType string

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TaskExecutor is the Schema for the taskexecutors API
// +k8s:openapi-gen=true
type TaskExecutor struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TaskExecutorSpec   `json:"spec,omitempty"`
	Status TaskExecutorStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TaskExecutorList contains a list of TaskExecutor
type TaskExecutorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TaskExecutor `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TaskExecutor{}, &TaskExecutorList{})
}
