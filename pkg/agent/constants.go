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

package agent

const (
	// AgentAPIPort is the TCP port used to serve the API of agents.
	AgentAPIPort = 6275

	// EnvAPIPort is the name of the environment variable use to pass the
	// TCP port the agent should list on for its API.
	EnvAPIPort = "KOALJA_API_PORT"

	// EnvPipelineName is the name of the environment variable use to pass the
	// name of the pipeline to an agent.
	EnvPipelineName = "KOALJA_PIPELINE_NAME"

	// EnvLinkName is the name of the environment variable use to pass the
	// name of the link to an agent.
	EnvLinkName = "KOALJA_LINK_NAME"

	// EnvTaskName is the name of the environment variable use to pass the
	// name of the task to an agent.
	EnvTaskName = "KOALJA_TASK_NAME"
)
