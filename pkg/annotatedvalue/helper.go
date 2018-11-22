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

import (
	"strings"

	types "github.com/gogo/protobuf/types"
)

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

// IsMatch returns true if the given annotated value matches the given request.
func (av *AnnotatedValue) IsMatch(req *GetRequest) bool {
	// Filter on ID
	if ids := req.GetIDs(); len(ids) > 0 {
		avID := av.GetID()
		found := false
		for _, id := range ids {
			if id == avID {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	// Filter on task
	if tasks := req.GetSourceTasks(); len(tasks) > 0 {
		avTask := av.GetSourceTask()
		found := false
		for _, t := range tasks {
			if t == avTask {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	// Filter on data
	if datas := req.GetData(); len(datas) > 0 {
		avData := av.GetData()
		found := false
		for _, d := range datas {
			if d == avData {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	// Filter on CreatedAfter
	if tsPB := req.GetCreatedAfter(); tsPB != nil {
		ts, err1 := types.TimestampFromProto(tsPB)
		avCreatedAt, err2 := types.TimestampFromProto(av.GetCreatedAt())
		if err1 != nil || err2 != nil || avCreatedAt.Before(ts) {
			return false
		}
	}
	// Filter on CreatedBefore
	if tsPB := req.GetCreatedBefore(); tsPB != nil {
		ts, err1 := types.TimestampFromProto(tsPB)
		avCreatedAt, err2 := types.TimestampFromProto(av.GetCreatedAt())
		if err1 != nil || err2 != nil || avCreatedAt.After(ts) {
			return false
		}
	}
	return true
}
