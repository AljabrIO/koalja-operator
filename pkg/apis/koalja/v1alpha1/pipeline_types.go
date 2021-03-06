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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PipelineSpec defines the desired state of Pipeline
type PipelineSpec struct {
	// Tasks of the pipeline
	Tasks []TaskSpec `json:"tasks,omitempty" protobuf:"bytes,1,rep,name=tasks"`
	// Links between tasks of the pipeline
	Links []LinkSpec `json:"links,omitempty" protobuf:"bytes,2,rep,name=links"`
	// Types of input/output data of tasks
	Types []TypeSpec `json:"types,omitempty" protobuf:"bytes,3,rep,name=types"`
}

// PipelineStatus defines the observed state of Pipeline
type PipelineStatus struct {
	// Domain name used for the pipeline
	Domain string `json:"domain,omitempty" protobuf:"bytes,1,opt,name=domain"`
	// Revision hash of the current specification of the pipeline
	Revision string `json:"revision,omitempty" protobuf:"bytes,2,opt,name=revision"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Pipeline is the Schema for the pipelines API
// +k8s:openapi-gen=true
type Pipeline struct {
	metav1.TypeMeta   `json:",inline" protobuf:"-"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec   PipelineSpec   `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
	Status PipelineStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PipelineList contains a list of Pipeline
type PipelineList struct {
	metav1.TypeMeta `json:",inline" protobuf:"-"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Items           []Pipeline `json:"items" protobuf:"bytes,2,rep,name=items"`
}

func init() {
	SchemeBuilder.Register(&Pipeline{}, &PipelineList{})
}

// TaskByName returns the task of the pipeline that has the given name.
// Returns false if not found.
func (ps PipelineSpec) TaskByName(name string) (TaskSpec, bool) {
	for _, x := range ps.Tasks {
		if x.Name == name {
			return x, true
		}
	}
	return TaskSpec{}, false
}

// LinkByName returns the link of the pipeline that has the given name.
// Returns false if not found.
func (ps PipelineSpec) LinkByName(name string) (LinkSpec, bool) {
	for _, x := range ps.Links {
		if x.Name == name {
			return x, true
		}
	}
	return LinkSpec{}, false
}

// LinksBySourceRef returns all links of the pipeline that has the given source ref.
func (ps PipelineSpec) LinksBySourceRef(ref string) []LinkSpec {
	var result []LinkSpec
	for _, x := range ps.Links {
		if x.SourceRef == ref {
			result = append(result, x)
		}
	}
	return result
}

// LinkByDestinationRef returns the link of the pipeline that has the given destination ref.
// Returns false if not found.
func (ps PipelineSpec) LinkByDestinationRef(ref string) (LinkSpec, bool) {
	for _, x := range ps.Links {
		if x.DestinationRef == ref {
			return x, true
		}
	}
	return LinkSpec{}, false
}

// TypeByName returns the (data)type of the pipeline that has the given name.
// Returns false if not found.
func (ps PipelineSpec) TypeByName(name string) (TypeSpec, bool) {
	for _, x := range ps.Types {
		if x.Name == name {
			return x, true
		}
	}
	return TypeSpec{}, false
}

// Validate the pipeline spec.
// Return an error when an issue is found, nil when all ok.
func (ps PipelineSpec) Validate() error {
	for _, x := range ps.Tasks {
		if err := x.Validate(ps); err != nil {
			return maskAny(err)
		}
	}
	for _, x := range ps.Links {
		if err := x.Validate(ps); err != nil {
			return maskAny(err)
		}
	}
	for _, x := range ps.Types {
		if err := x.Validate(ps); err != nil {
			return maskAny(err)
		}
	}
	return nil
}
