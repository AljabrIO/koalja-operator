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

// LinkSpec holds the specification of a single link between tasks
type LinkSpec struct {
	// Name of the link
	Name string `json:"name"`
	// Source of the link
	Source *LinkSourceSpec `json:"source,omitempty"`
	// SourceRef specifies the source of the link as `taskName/outputName`.
	// SourceRef must be empty when Source is specified.
	SourceRef string `json:"sourceRef,omitempty"`
	// DestinationRef specifies the destination of the link as `taskName/inputName`
	DestinationRef string `json:"destinationRef"`
}

// LinkSourceSpec holds the specification
type LinkSourceSpec struct {
	// Type of source
	Type LinkSourceType `json:"type"`
}

// LinkSourceType indicates the type of source of a link
type LinkSourceType string

const (
	// LinkSourceTypeFileDrop indicates that the source of a link is a manual file drop.
	LinkSourceTypeFileDrop LinkSourceType = "FileDrop"
)

// Validate the link in the context of the given pipeline spec.
// Return an error when an issue is found, nil when all ok.
func (ls LinkSpec) Validate(ps PipelineSpec) error {
	if err := ValidateName(ls.Name); err != nil {
		return maskAny(err)
	}
	if ls.Source == nil && ls.SourceRef == "" {
		return errors.Wrapf(ErrValidation, "Source or SourceRef expected in link '%s'", ls.Name)
	} else if ls.Source != nil && ls.SourceRef != "" {
		return errors.Wrapf(ErrValidation, "Source or SourceRef expected in link '%s', not both", ls.Name)
	} else if ls.SourceRef != "" {
		if _, _, err := SplitTaskRef(ls.SourceRef); err != nil {
			return maskAny(err)
		}
	}
	if ls.DestinationRef == "" {
		return errors.Wrapf(ErrValidation, "DestinationRef expected in link '%s'", ls.Name)
	} else if _, _, err := SplitTaskRef(ls.DestinationRef); err != nil {
		return maskAny(err)
	}
	return nil
}
