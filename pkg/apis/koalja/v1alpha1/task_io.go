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
	Name string `json:"name" protobuf:"bytes,1,req"`
	// Reference to the type of the input
	TypeRef string `json:"typeRef" protobuf:"bytes,2,req"`
	// SnapshotPolicy determines how to sample events into a tuple
	// that is the input for task execution.
	// Defaults to "All".
	SnapshotPolicy InputSnapshotPolicy `json:"snapshotPolicy,omitempty" protobuf:"bytes,3,opt"`
}

// InputSnapshotPolicy determines how to sample events into a tuple
// that is the input for task execution.
type InputSnapshotPolicy string

const (
	// InputSnapshotPolicyAll indicates that all events of this input
	// must yield the execution of the task.
	InputSnapshotPolicyAll InputSnapshotPolicy = "All"
	// InputSnapshotPolicyLatest indicates that all the latest event of this input
	// is used for the execution of the task.
	InputSnapshotPolicyLatest InputSnapshotPolicy = "Latest"
)

// TaskOutputSpec holds the specification of a single output of a task
type TaskOutputSpec struct {
	// Name of the output
	Name string `json:"name" protobuf:"bytes,1,req"`
	// Reference to the type of the output
	TypeRef string `json:"typeRef" protobuf:"bytes,2,req"`
	// Ready indicates when this output is ready and available for the
	// next task in the pipeline.
	// Defaults to "Succeeded".
	Ready OutputReadiness `json:"ready" protobuf:"bytes,3,opt"`
}

// OutputReadiness specifies when an output of a task is ready for the
// next task in the pipeline.
type OutputReadiness string

const (
	// OutputReadyAuto indicates that the output is automatically pushed
	// into the output link by the executor.
	OutputReadyAuto OutputReadiness = "Auto"
	// OutputReadySucceeded indicates that the output is available to be pushed
	// into the output link when the executor has terminated succesfully.
	OutputReadySucceeded OutputReadiness = "Succeeded"
)

// IsAll returns true when the given policy is "All"
func (isp InputSnapshotPolicy) IsAll() bool { return isp == InputSnapshotPolicyAll || isp == "" }

// IsLatest returns true when the given policy is "Latest"
func (isp InputSnapshotPolicy) IsLatest() bool { return isp == InputSnapshotPolicyLatest }

// Validate that the given readiness is a valid value.
func (isp InputSnapshotPolicy) Validate() error {
	switch isp {
	case InputSnapshotPolicyAll, InputSnapshotPolicyLatest, "":
		return nil
	default:
		return errors.Wrapf(ErrValidation, "Invalid InputSnapshotPolicy '%s'", string(isp))
	}
}

// Validate that the given readiness is a valid value.
func (or OutputReadiness) Validate() error {
	switch or {
	case OutputReadyAuto, OutputReadySucceeded, "":
		return nil
	default:
		return errors.Wrapf(ErrValidation, "Invalid OutputReadiness '%s'", string(or))
	}
}

// IsAuto returns true when the given readiness is "Auto"
func (or OutputReadiness) IsAuto() bool { return or == OutputReadyAuto }

// IsSucceeded returns true when the given readiness is "Succeeded"
func (or OutputReadiness) IsSucceeded() bool { return or == OutputReadySucceeded || or == "" }

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
	if err := tis.SnapshotPolicy.Validate(); err != nil {
		return maskAny(err)
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
	if err := tos.Ready.Validate(); err != nil {
		return maskAny(err)
	}
	return nil
}
