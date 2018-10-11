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

// TaskInputSpec holds the specification of a single input of a task
type TaskInputSpec struct {
	// Name of the input
	Name string `json:"name"`
	// Reference to the type of the input
	TypeRef string `json:"typeRef"`
}

// TaskOutputSpec holds the specification of a single output of a task
type TaskOutputSpec struct {
	// Name of the output
	Name string `json:"name"`
	// Reference to the type of the output
	TypeRef string `json:"typeRef"`
}

// Validate the task input in the context of the given pipeline spec.
// Return an error when an issue is found, nil when all ok.
func (tis TaskInputSpec) Validate(ps PipelineSpec) error {
	if err := ValidateName(tis.Name); err != nil {
		return maskAny(err)
	}
	if tis.TypeRef == "" {
		return errors.Wrapf(ErrValidation, "TypeRef expected")
	}
	if _, found := ps.TypeByName(tis.TypeRef); !found {
		return errors.Wrapf(ErrValidation, "TypeRef '%s' not found", tis.TypeRef)
	}
	return nil
}

// Validate the task output in the context of the given pipeline spec.
// Return an error when an issue is found, nil when all ok.
func (tos TaskOutputSpec) Validate(ps PipelineSpec) error {
	if err := ValidateName(tos.Name); err != nil {
		return maskAny(err)
	}
	if tos.TypeRef == "" {
		return errors.Wrapf(ErrValidation, "TypeRef expected")
	}
	if _, found := ps.TypeByName(tos.TypeRef); !found {
		return errors.Wrapf(ErrValidation, "TypeRef '%s' not found", tos.TypeRef)
	}
	return nil
}
