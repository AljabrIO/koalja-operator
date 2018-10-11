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

// TaskExecutorSpec holds the specification of the execution part of a task
type TaskExecutorSpec struct {
	// Container holds the image name of the container to execute
	Container string `json:"container"`
	// Args passed as commandline to the container
	Args []string `json:"args,omitempty"`
}

// Validate the task executor in the context of the given pipeline spec.
// Return an error when an issue is found, nil when all ok.
func (tes TaskExecutorSpec) Validate(ps PipelineSpec) error {
	if tes.Container == "" {
		return errors.Wrapf(ErrValidation, "Container expected")
	}
	return nil
}
