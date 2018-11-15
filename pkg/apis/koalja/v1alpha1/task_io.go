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
	"github.com/AljabrIO/koalja-operator/pkg/util"
	"github.com/pkg/errors"
)

// TaskInputSpec holds the specification of a single input of a task.
//
// The task will execute when for the following condition is met for every input of the task.
// - The number of fetched annotated values is >= MinSequenceLength.
//
// If the number of fetched annotated values equals to MaxSequenceLength
// the following behavior applies:
// - If SnapshotPolicy == "All", fetching annotated values for this input is
//   paused (to be resumed after execution of the task).
// - If SnapshotPolicy == "Latest", newly fetched annotated values replace
//   the ones that have been fetched before.
//   For example (MinSequenceLength=3, MaxSequenceLength=3, SnapshotPolicy="Latest"):
//   If the sequence (of annotated values) for this input is A1,A2,A3
//   then the arrival of A4 will change the sequence for this input to A2,A3,A4
//   and A1 will be acknowledged.
type TaskInputSpec struct {
	// Name of the input
	Name string `json:"name" protobuf:"bytes,1,req,name=name"`
	// Reference to the type of the input
	TypeRef string `json:"typeRef" protobuf:"bytes,2,req,name=typeRef"`
	// SnapshotPolicy determines how to sample annotated values into a tuple
	// that is the input for task execution.
	// Defaults to "All".
	SnapshotPolicy InputSnapshotPolicy `json:"snapshotPolicy,omitempty" protobuf:"bytes,3,opt,name=snapshotPolicy"`
	// MinSequenceLength determines the minimum number of annotated values to fetch
	// before this input is considered ready for execution.
	// Defaults to 1.
	MinSequenceLength *int64 `json:"minSequenceLength,omitempty" protobuf:"bytes,4,req,name=minSequenceLength"`
	// MaxSequenceLength determines the maximum number of annotated values to fetch
	// before this input is considered ready for execution.
	// Defaults to MinSequenceLength.
	MaxSequenceLength *int64 `json:"maxSequenceLength,omitempty" protobuf:"bytes,5,req,name=maxSequenceLength"`
}

// InputSnapshotPolicy determines how to sample annotated values into a tuple
// that is the input for task execution.
type InputSnapshotPolicy string

const (
	// InputSnapshotPolicyAll indicates that all annotated values of this input
	// must yield the execution of the task.
	InputSnapshotPolicyAll InputSnapshotPolicy = "All"
	// InputSnapshotPolicyLatest indicates that all the latest annotated value of this input
	// is used for the execution of the task.
	InputSnapshotPolicyLatest InputSnapshotPolicy = "Latest"
)

// TaskOutputSpec holds the specification of a single output of a task
type TaskOutputSpec struct {
	// Name of the output
	Name string `json:"name" protobuf:"bytes,1,req,name=name"`
	// Reference to the type of the output
	TypeRef string `json:"typeRef" protobuf:"bytes,2,req,name=typeRef"`
	// Ready indicates when this output is ready and available for the
	// next task in the pipeline.
	// Defaults to "Succeeded".
	Ready OutputReadiness `json:"ready" protobuf:"bytes,3,opt,name=ready"`
	// Options is an optional set of key-value pairs used to pass data to a task executor.
	Options map[string]string `json:"options,omitempty" protobuf:"bytes,4,req,name=options"`
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

// GetMinSequenceLength returns MinSequenceLength with a default of 1.
func (tis TaskInputSpec) GetMinSequenceLength() int {
	return int(util.Int64OrDefault(tis.MinSequenceLength, 1))
}

// GetMaxSequenceLength returns MaxSequenceLength with a default of MinSequenceLength.
func (tis TaskInputSpec) GetMaxSequenceLength() int {
	return int(util.Int64OrDefault(tis.MaxSequenceLength, int64(tis.GetMinSequenceLength())))
}

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
	if tis.GetMinSequenceLength() < 1 {
		return errors.Wrapf(ErrValidation, "MinSequenceLength must be >= 1")
	}
	if tis.GetMaxSequenceLength() < tis.GetMinSequenceLength() {
		return errors.Wrapf(ErrValidation, "MaxSequenceLength must be >= MinSequenceLength")
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
