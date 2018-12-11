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

import "github.com/pkg/errors"

// ServiceSpec holds the specification of a service provided by a task.
type ServiceSpec struct {
	Ports []ServicePortSpec `json:"ports,omitempty" protobuf:"bytes,1,req,name=ports"`
}

// ServicePortSpec holds the specification of a single port exposed by a service task.
type ServicePortSpec struct {
	// Name of the port
	Name string `json:"name" protobuf:"bytes,1,req,name=name"`
	// Port number of the port
	Port int32 `json:"port" protobuf:"int32,2,req,name=port"`
}

// Validate the port spec.
func (sps ServicePortSpec) Validate() error {
	if err := ValidateName(sps.Name); err != nil {
		return maskAny(err)
	}
	if sps.Port < 1 || sps.Port > 32*1024 {
		return errors.Wrapf(ErrValidation, "Port out of range: %d", sps.Port)
	}
	return nil
}

// Validate the service spec.
func (s ServiceSpec) Validate() error {
	for _, sps := range s.Ports {
		if err := sps.Validate(); err != nil {
			return maskAny(err)
		}
	}
	if len(s.Ports) == 0 {
		return errors.Wrapf(ErrValidation, "Expected at least 1 port")
	}
	return nil
}
