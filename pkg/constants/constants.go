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

package constants

import (
	"fmt"
	"os"
	"strconv"
)

const (
	// AgentAPIPort is the TCP port used to serve the API of agents.
	AgentAPIPort = 6275

	// LinkSidecarAPIPort is the TCP port used to serve the API of link sidecars.
	LinkSidecarAPIPort = 6276

	// EnvAPIPort is the name of the environment variable used to pass the
	// TCP port the agent should list on for its API.
	EnvAPIPort = "KOALJA_API_PORT"

	// EnvNamespace is the name of the environment variable used to pass the
	// namespace of the running agent/sidecar.
	EnvNamespace = "KOALJA_NAMESPACE"

	// EnvPipelineName is the name of the environment variable used to pass the
	// name of the pipeline to an agent/sidecar.
	EnvPipelineName = "KOALJA_PIPELINE_NAME"

	// EnvLinkName is the name of the environment variable used to pass the
	// name of the link to an agent/sidecar.
	EnvLinkName = "KOALJA_LINK_NAME"

	// EnvTaskName is the name of the environment variable used to pass the
	// name of the task to an agent.
	EnvTaskName = "KOALJA_TASK_NAME"

	// EnvProtocol is the name of the environment variable used to pass the
	// protocol of a link endpoint to a link sidecar.
	EnvProtocol = "KOALJA_PROTOCOL"

	// EnvFormat is the name of the environment variable used to pass the
	// format of a link endpoint to a link sidecar.
	EnvFormat = "KOALJA_FORMAT"

	// EnvLinkSide is the name of the environment variable used to pass the
	// side of a link to a link sidecar.
	// Possible values are LinkSideSource & LinkSideDestination.
	EnvLinkSide = "KOALJA_LINK_SIDE"

	// LinkSideSource is a value for EnvLinkSide, indicating that the sidecar serves
	// as the source of a link
	LinkSideSource = "Source"
	// LinkSideDestination is a value for EnvLinkSide, indicating that the sidecar serves
	// as the destination of a link
	LinkSideDestination = "Destination"
)

// GetAPIPort returns the port to listen on for agents/sidecars, found in the environment,
func GetAPIPort() (int, error) {
	portStr := os.Getenv(EnvAPIPort)
	result, err := strconv.Atoi(portStr)
	if err != nil {
		return 0, err
	}
	return result, nil
}

// GetNamespace loads the namespace in which the agent is running.
func GetNamespace() (string, error) {
	ns := os.Getenv(EnvNamespace)
	if ns == "" {
		return "", fmt.Errorf("Environment variable '%s' not set", EnvNamespace)
	}
	return ns, nil
}

// GetPipelineName loads the pipeline name passed to the agent identified by a name in the environment.
func GetPipelineName() (string, error) {
	name := os.Getenv(EnvPipelineName)
	if name == "" {
		return "", fmt.Errorf("Environment variable '%s' not set", EnvPipelineName)
	}
	return name, nil
}

// GetLinkName loads the link name passed to the agent identified by a name in the environment.
func GetLinkName() (string, error) {
	name := os.Getenv(EnvLinkName)
	if name == "" {
		return "", fmt.Errorf("Environment variable '%s' not set", EnvLinkName)
	}
	return name, nil
}

// GetTaskName loads the task name passed to the agent identified by a name in the environment.
func GetTaskName() (string, error) {
	name := os.Getenv(EnvTaskName)
	if name == "" {
		return "", fmt.Errorf("Environment variable '%s' not set", EnvTaskName)
	}
	return name, nil
}
