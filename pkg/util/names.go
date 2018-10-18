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

package util

import (
	"crypto/sha1"
	"fmt"
	"strings"
)

const (
	maxNameLength = 63
)

// FixupKubernetesName converts the given name to a valid kubernetes
// resource name.
func FixupKubernetesName(name string) string {
	result := strings.ToLower(name)
	if result != name || len(result) > maxNameLength {
		// Add hash
		h := sha1.Sum([]byte(name))
		suffix := fmt.Sprintf("-%0x", h)[:8]
		if len(result)+len(suffix) > maxNameLength {
			result = result[:maxNameLength-len(suffix)]
		}
		result = result + suffix
	}
	return result
}
