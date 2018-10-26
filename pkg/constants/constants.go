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

	"github.com/AljabrIO/koalja-operator/pkg/util"
)

const (
	// AgentAPIPort is the TCP port used to serve the API of agents.
	AgentAPIPort = 6275

	// EventRegistryAPIPort is the TCP port used to serve the API of an event registry.
	EventRegistryAPIPort = 6276

	// EnvAPIPort is the name of the environment variable used to pass the
	// TCP port the agent should list on for its API.
	EnvAPIPort = "KOALJA_API_PORT"

	// EnvNamespace is the name of the environment variable used to pass the
	// namespace of the running agent/sidecar.
	EnvNamespace = "KOALJA_NAMESPACE"

	// EnvPodName is the name of the environment variable used to pass the
	// name of the Pod to the running container.
	EnvPodName = "KOALJA_POD_NAME"

	// EnvDNSName is the name of the environment variable used to pass the
	// DNS name of itself to a running agent.
	EnvDNSName = "KOALJA_DNS_NAME"

	// EnvPipelineName is the name of the environment variable used to pass the
	// name of the pipeline to an agent/sidecar.
	EnvPipelineName = "KOALJA_PIPELINE_NAME"

	// EnvAgentRegistryAddress is the name of the environment variable used to pass the
	// address of the AgentRegistry to a container.
	EnvAgentRegistryAddress = "KOALJA_AGENT_REGISTRY_ADDRESS"

	// EnvEventRegistryAddress is the name of the environment variable used to pass the
	// address of the EventRegistry to a container.
	EnvEventRegistryAddress = "KOALJA_EVENT_REGISTRY_ADDRESS"

	// EnvFileSystemAddress is the name of the environment variable used to pass the
	// address of the FileSystem to a container.
	EnvFileSystemAddress = "KOALJA_FILESYSTEM_ADDRESS"

	// EnvOutputReadyNotifierAddress is the name of the environment variable used to pass the
	// address of the OutputReadyNotifier to a container.
	EnvOutputReadyNotifierAddress = "KOALJA_OUTPUT_READ_NOTIFIER_ADDRESS"

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

	// AnnInputLinkAddressPrefix is the prefix of an annotation key used to pass
	// the address of an input link to a task.
	// Full annotation key: AnnInputLinkAddressPrefix + InputName
	// Annotation value: <host>:<port>
	AnnInputLinkAddressPrefix = "koalja.aljabr.io/input-link-address-"

	// AnnOutputLinkAddressesPrefix is the prefix of an annotation key used to pass
	// comma-separated list of addresses of output links to a task.
	// Full annotation key: AnnOutputLinkAddressesPrefix + OutputName
	// Annotation value: <host1>:<port1>[, <host1:port2> ...]
	AnnOutputLinkAddressesPrefix = "koalja.aljabr.io/output-link-addresses-"

	// AnnTaskExecutorContainer is the annotation key used to pass
	// the container (as JSON) of the executor of the task to the task.
	AnnTaskExecutorContainer = "koalja.aljabr.io/task-executor-container"

	// LabelServiceType is the label key used to pass the type of service
	// to a Service.
	LabelServiceType = "koalja.aljabr.io/serviceType"

	// ServiceTypeFilesystem is a possible value for labels with key LabelServiceType.
	ServiceTypeFilesystem = "FileSystem"

	// LabelSpecHash is the label key used to contain a hash of its
	// entire specification. This is used for improved change detection.
	LabelSpecHash = "koalja.aljabr.io/specHash"
)

// CreateInputLinkAddressAnnotationName creates a full annotation name
// using AnnInputLinkAddressPrefix.
func CreateInputLinkAddressAnnotationName(inputName string) string {
	return AnnInputLinkAddressPrefix + util.FixupKubernetesName(inputName)
}

// CreateOutputLinkAddressesAnnotationName creates a full annotation name
// using AnnOutputLinkAddressesPrefix.
func CreateOutputLinkAddressesAnnotationName(inputName string) string {
	return AnnOutputLinkAddressesPrefix + util.FixupKubernetesName(inputName)
}

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

// GetPodName loads the name of the pod in which the agent is running.
func GetPodName() (string, error) {
	ns := os.Getenv(EnvPodName)
	if ns == "" {
		return "", fmt.Errorf("Environment variable '%s' not set", EnvPodName)
	}
	return ns, nil
}

// GetDNSName loads the DNS name of the running agent.
func GetDNSName() (string, error) {
	ns := os.Getenv(EnvDNSName)
	if ns == "" {
		return "", fmt.Errorf("Environment variable '%s' not set", EnvDNSName)
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
