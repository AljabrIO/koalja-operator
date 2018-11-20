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

package annotatedvalue

import "strings"

// GetDataScheme returns the scheme of the Data URI.
func (av *AnnotatedValue) GetDataScheme() Scheme {
	return GetDataScheme(av.GetData())
}

// GetDataScheme returns the scheme of the Data URI.
func GetDataScheme(data string) Scheme {
	if idx := strings.Index(data, ":"); idx > 0 {
		return Scheme(data[:idx])
	}
	return ""
}
