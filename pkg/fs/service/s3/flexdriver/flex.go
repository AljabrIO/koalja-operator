//
// Copyright © 2018 Aljabr, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package main

import (
	"encoding/json"
	"fmt"
)

type FlexStatus string

const (
	FlexStatusSuccess      FlexStatus = "Success"
	FlexStatusNotSupported FlexStatus = "Not supported"
	FlexStatusFailure      FlexStatus = "Failure"
)

type FlexOutput struct {
	Status       FlexStatus        `json:"status,omitempty"`
	Message      string            `json:"message,omitempty"`
	Device       string            `json:"device,omitempty"`
	VolumeName   string            `json:"volumename,omitempty"`
	Attached     bool              `json:"attached,omitempty"`
	Capabilities *FlexCapabilities `json:"capabilities,omitempty"`
}

type FlexCapabilities struct {
	Attach bool `json:"attach"`
}

func sendOutput(o FlexOutput) {
	encoded, _ := json.Marshal(o)
	fmt.Print(string(encoded))
}

func parseJSONOptions(jsonOptions string) (map[string]string, error) {
	var result map[string]string
	if err := json.Unmarshal([]byte(jsonOptions), &result); err != nil {
		return nil, err
	}
	return result, nil
}
