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

package protocols

import (
	"github.com/AljabrIO/koalja-operator/pkg/agent/task"
	koalja "github.com/AljabrIO/koalja-operator/pkg/apis/koalja/v1alpha1"
)

type fileInputBuilder struct{}

func init() {
	task.RegisterExecutorInputBuilder(koalja.ProtocolFile, fileInputBuilder{})
}

// Build input settings for an input of File protocol.
func (b fileInputBuilder) Build(cfg task.ExecutorInputBuilderConfig, deps task.ExecutorInputBuilderDependencies, target *task.ExecutorInputBuilderTarget) error {
	return nil
}
