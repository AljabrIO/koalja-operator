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

// TypeSpec holds the specification of a single type of data
type TypeSpec struct {
	// Name of the type
	Name string `json:"name" protobuf:"bytes,1,req"`
	// Protocol indicating how to get to the data
	Protocol Protocol `json:"protocol" protobuf:"bytes,2,req"`
	// Format of the content of the data
	Format Format `json:"format,omitempty" protobuf:"bytes,3,opt"`
}

// Protocol of a data type.
// Indicates how to get to the data.
type Protocol string

const (
	// ProtocolFile indicates that the data is found in a single file on a filesystem
	ProtocolFile Protocol = "File"
)

// Format of a data type.
// Indicates the format of the content of the data.
type Format string

// Validate the data in the context of the given pipeline spec.
// Return an error when an issue is found, nil when all ok.
func (ts TypeSpec) Validate(ps PipelineSpec) error {
	if err := ValidateName(ts.Name); err != nil {
		return maskAny(err)
	}
	return nil
}
