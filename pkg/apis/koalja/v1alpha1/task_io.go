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
type TaskInputSpec struct {
	// Name of the input
	Name string `json:"name" protobuf:"bytes,1,req,name=name"`
	// Reference to the type of the input
	TypeRef string `json:"typeRef" protobuf:"bytes,2,req,name=typeRef"`
	// RequiredSequenceLength determines the minimum & maximum number of annotated values to fetch
	// before this input is considered ready for execution.
	// Semantics depend on the SnapshotPolicy of the task.
	// Defaults to 1.
	// If this value is set, MinSequenceLength & MaxSequenceLength are no longer used.
	RequiredSequenceLength *int64 `json:"requiredSequenceLength,omitempty" protobuf:"bytes,6,req,name=requiredSequenceLength"`
	// MinSequenceLength determines the minimum number of annotated values to fetch
	// before this input is considered ready for execution.
	// Semantics depend on the SnapshotPolicy of the task.
	// Defaults to 1.
	MinSequenceLength *int64 `json:"minSequenceLength,omitempty" protobuf:"bytes,4,req,name=minSequenceLength"`
	// MaxSequenceLength determines the maximum number of annotated values to fetch
	// before this input is considered ready for execution.
	// Semantics depend on the SnapshotPolicy of the task.
	// Defaults to MinSequenceLength.
	MaxSequenceLength *int64 `json:"maxSequenceLength,omitempty" protobuf:"bytes,5,req,name=maxSequenceLength"`
	// Slide specifies the maximum number of annotated values to slide
	// out of the current snapshot when task execution is done.
	// This property is not relevant when task.SnapshotPolicy != SlidingWindow.
	// Defaults to 1.
	Slide *int64 `json:"slide,omitempty" protobuf:"bytes,7,req,name=slide"`
	// MergeInto specifies the name of another input of this task.
	// If set, all annotated values that are coming in through this input are merge into
	// the stream of annotated values coming in to the link with that name.
	// Inputs that have MergeInto set will not yield any arguments for the task executor.
	MergeInto string `json:"mergeInto,omitempty" protobuf:"bytes,8,req,name=mergeInto"`
}

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
	if tis.RequiredSequenceLength != nil {
		return int(*tis.RequiredSequenceLength)
	}
	return int(util.Int64OrDefault(tis.MinSequenceLength, 1))
}

// GetMaxSequenceLength returns MaxSequenceLength with a default of MinSequenceLength.
func (tis TaskInputSpec) GetMaxSequenceLength() int {
	if tis.RequiredSequenceLength != nil {
		return int(*tis.RequiredSequenceLength)
	}
	return int(util.Int64OrDefault(tis.MaxSequenceLength, int64(tis.GetMinSequenceLength())))
}

// GetSlide returns Slide with a default of 1.
func (tis TaskInputSpec) GetSlide() int {
	return int(util.Int64OrDefault(tis.Slide, 1))
}

// HasMergeInto returns true if MergeInto is set to a non-empty value.
func (tis TaskInputSpec) HasMergeInto() bool {
	return tis.MergeInto != ""
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
func (tis TaskInputSpec) Validate(ps PipelineSpec, ts TaskSpec) error {
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
	switch ts.SnapshotPolicy {
	case SnapshotPolicyAllNew:
		// Min/Max is unrestricted
	case SnapshotPolicySwapNew4Old:
		// Min=Max=1
		if tis.GetMinSequenceLength() != 1 {
			return errors.Wrapf(ErrValidation, "MinSequenceLength must be equal to 1")
		}
		if tis.GetMaxSequenceLength() != 1 {
			return errors.Wrapf(ErrValidation, "MaxSequenceLength must be equal to 1")
		}
	case SnapshotPolicySlidingWindow:
		// Min=Max
		if tis.GetMinSequenceLength() != tis.GetMaxSequenceLength() {
			return errors.Wrapf(ErrValidation, "MaxSequenceLength must be equal to MinSequenceLength")
		}
	}
	if tis.GetSlide() < 1 {
		return errors.Wrapf(ErrValidation, "Slide must be >= 1")
	}
	if tis.GetSlide() > tis.GetMinSequenceLength() {
		return errors.Wrapf(ErrValidation, "Slide must be <= MinSequenceLength")
	}
	if _, found := ps.TypeByName(tis.TypeRef); !found {
		return errors.Wrapf(ErrValidation, "TypeRef '%s' not found", tis.TypeRef)
	}
	if tis.HasMergeInto() {
		if mergeInto, found := ts.InputByName(tis.MergeInto); !found {
			return errors.Wrapf(ErrValidation, "MergeInto input '%s' not found", tis.MergeInto)
		} else if mergeInto.HasMergeInto() {
			return errors.Wrapf(ErrValidation, "MergeInto refers to input '%s' that has also set MergeInto", tis.MergeInto)
		} else if tis.TypeRef != mergeInto.TypeRef {
			return errors.Wrapf(ErrValidation, "MergeInto refers to input '%s' that has another type", tis.MergeInto)
		}
		if tis.MergeInto == tis.Name {
			return errors.Wrapf(ErrValidation, "MergeInto input '%s' must not use the name of the same input", tis.MergeInto)
		}
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
