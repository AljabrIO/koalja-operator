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
	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
)

// TaskSpec holds the specification of a single task
type TaskSpec struct {
	// Name of the task
	Name string `json:"name"`
	// Type of task
	Type TaskType `json:"type,omitempty"`
	// Inputs of the task
	Inputs []TaskInputSpec `json:"inputs,omitempty"`
	// Outputs of the task
	Outputs []TaskOutputSpec `json:"outputs"`
	// Executor holds the spec of the execution part of the task
	Executor *core.Container `json:"executor,omitempty"`
}

// TaskType identifies a well know type of task.
type TaskType string

// InputByName returns the input of the task that has the given name.
// Returns false if not found.
func (ts TaskSpec) InputByName(name string) (TaskInputSpec, bool) {
	for _, x := range ts.Inputs {
		if x.Name == name {
			return x, true
		}
	}
	return TaskInputSpec{}, false
}

// OutputByName returns the output of the task that has the given name.
// Returns false if not found.
func (ts TaskSpec) OutputByName(name string) (TaskOutputSpec, bool) {
	for _, x := range ts.Outputs {
		if x.Name == name {
			return x, true
		}
	}
	return TaskOutputSpec{}, false
}

// Validate the task in the context of the given pipeline spec.
// Return an error when an issue is found, nil when all ok.
func (ts TaskSpec) Validate(ps PipelineSpec) error {
	if err := ValidateName(ts.Name); err != nil {
		return maskAny(err)
	}
	if ts.Executor == nil && ts.Type == "" {
		return errors.Wrapf(ErrValidation, "Executor or Type expected in task '%s'", ts.Name)
	}
	if len(ts.Outputs) == 0 {
		return errors.Wrapf(ErrValidation, "Task '%s' must have at least 1 output", ts.Name)
	}
	names := make(map[string]struct{})
	for _, x := range ts.Inputs {
		if _, found := names[x.Name]; found {
			return errors.Wrapf(ErrValidation, "Duplicate input name '%s' in task '%s'", x.Name, ts.Name)
		}
		names[x.Name] = struct{}{}
		if err := x.Validate(ps); err != nil {
			return maskAny(err)
		}
	}
	names = make(map[string]struct{})
	for _, x := range ts.Outputs {
		if _, found := names[x.Name]; found {
			return errors.Wrapf(ErrValidation, "Duplicate output name '%s' in task '%s'", x.Name, ts.Name)
		}
		names[x.Name] = struct{}{}
		if err := x.Validate(ps); err != nil {
			return maskAny(err)
		}
	}
	if ts.Executor != nil {
		if ts.Executor.Image == "" {
			return errors.Wrapf(ErrValidation, "Executor of task '%s' must have an image", ts.Name)
		}
	}
	if len(ts.Inputs) > 0 {
		hasAllPolicy := false
		for _, ti := range ts.Inputs {
			if ti.SnapshotPolicy.IsAll() {
				hasAllPolicy = true
				break
			}
		}
		if !hasAllPolicy {
			return errors.Wrapf(ErrValidation, "Task '%s' must have at least 1 input with an '%s' snapshot policy", ts.Name, InputSnapshotPolicyAll)
		}
	}
	return nil
}
