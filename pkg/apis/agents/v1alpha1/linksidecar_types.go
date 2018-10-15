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

// LinkSidecarSpec defines the desired state of LinkSidecar
type LinkSidecarSpec struct {
	// Protocol that this endpoint supports (required)
	Protocol string `json:"protocol"`
	// Format of data that this endpoint supports (optional)
	Format string `json:"format,omitempty"`
	// Container used to run the endpoint
	Container *core.Container `json:"container"`
}

// LinkSidecarStatus defines the observed state of LinkSidecar
type LinkSidecarStatus struct {
	// Empty on purpose
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// LinkSidecar is the Schema for the linksidecars API
// +k8s:openapi-gen=true
type LinkSidecar struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LinkSidecarSpec   `json:"spec,omitempty"`
	Status LinkSidecarStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// LinkSidecarList contains a list of LinkSidecar
type LinkSidecarList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LinkSidecar `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LinkSidecar{}, &LinkSidecarList{})
}
