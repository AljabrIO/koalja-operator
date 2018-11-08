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

// AnnotatedValueRegistrySpec defines the desired state of AnnotatedValueRegistry
type AnnotatedValueRegistrySpec struct {
	// Container used to run the annotated value registry
	Container *core.Container `json:"container"`
}

// AnnotatedValueRegistryStatus defines the observed state of AnnotatedValueRegistry
type AnnotatedValueRegistryStatus struct {
	// Empty on purpose
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AnnotatedValueRegistry is the Schema for the annotatedvaluesregistries API
// +k8s:openapi-gen=true
type AnnotatedValueRegistry struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AnnotatedValueRegistrySpec   `json:"spec,omitempty"`
	Status AnnotatedValueRegistryStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AnnotatedValueRegistryList contains a list of AnnotatedValueRegistry
type AnnotatedValueRegistryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AnnotatedValueRegistry `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AnnotatedValueRegistry{}, &AnnotatedValueRegistryList{})
}
