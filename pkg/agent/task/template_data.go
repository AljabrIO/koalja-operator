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

package task

import (
	koalja "github.com/AljabrIO/koalja-operator/pkg/apis/koalja/v1alpha1"
	"github.com/AljabrIO/koalja-operator/pkg/controller/pipeline"
)

// BuildPipelineDataMap builds a (template) data structure for the pipeline
// elements.
func BuildPipelineDataMap(pl *koalja.Pipeline) map[string]interface{} {
	taskBuilder := func(taskSpec *koalja.TaskSpec) map[string]interface{} {
		d := map[string]interface{}{}
		if taskSpec.Service != nil {
			d["service"] = map[string]interface{}{
				"name": pipeline.CreateTaskExecutorDNSName(pl.GetName(), taskSpec.Name, pl.GetNamespace()),
			}
		}
		return d
	}

	tasksData := make(map[string]interface{})
	for _, task := range pl.Spec.Tasks {
		tasksData[task.Name] = taskBuilder(&task)
	}

	data := map[string]interface{}{
		"tasks": tasksData,
		// TODO add type & link data
	}
	return data
}
