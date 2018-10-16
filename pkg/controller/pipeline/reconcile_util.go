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

package pipeline

import (
	"crypto/sha1"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/AljabrIO/koalja-operator/pkg/constants"
)

const (
	maxNameLength = 63
)

// MergeReconcileResult combines the given results, taking the smaller
// after timeout.
func MergeReconcileResult(a, b reconcile.Result) reconcile.Result {
	var after time.Duration
	if a.RequeueAfter > 0 {
		after = a.RequeueAfter
	}
	if b.RequeueAfter > 0 && b.RequeueAfter < after {
		after = b.RequeueAfter
	}
	return reconcile.Result{
		Requeue:      a.Requeue || b.Requeue,
		RequeueAfter: after,
	}
}

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

// SetAgentContainerDefaults applies default values to a container used to run an agent
func SetAgentContainerDefaults(c *corev1.Container) {
	if c.Name == "" {
		c.Name = "agent"
	}
	if len(c.Ports) == 0 {
		c.Ports = []corev1.ContainerPort{
			corev1.ContainerPort{
				Name:          "api",
				ContainerPort: constants.AgentAPIPort,
			},
		}
	}
}

// SetContainerEnvVars sets the given environment variables into the given container
func SetContainerEnvVars(c *corev1.Container, vars map[string]string) {
	for k, v := range vars {
		found := false
		for i, ev := range c.Env {
			if ev.Name == k {
				c.Env[i].Value = v
				c.Env[i].ValueFrom = nil
				found = true
				break
			}
		}
		if !found {
			c.Env = append(c.Env, corev1.EnvVar{
				Name:  k,
				Value: v,
			})
		}
	}
	c.Env = append(c.Env,
		corev1.EnvVar{
			Name: constants.EnvNamespace,
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.namespace",
				},
			},
		},
		corev1.EnvVar{
			Name: constants.EnvPodName,
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		},
	)
}

// CreatePipelineAgentName returns the name of the pipeline agent
// for the given pipeline.
func CreatePipelineAgentName(pipelineName string) string {
	return FixupKubernetesName(pipelineName + "-pl-agent")
}

// CreatePipelineAgentDNSName returns the DNS of the pipeline agent
// for the given pipeline.
func CreatePipelineAgentDNSName(pipelineName, namespace string) string {
	return fmt.Sprintf("%s.%s.svc", CreatePipelineAgentName(pipelineName), namespace)
}

// CreateLinkAgentName returns the name of the link agent
// for the given pipeline + link.
func CreateLinkAgentName(pipelineName, linkName string) string {
	return FixupKubernetesName(pipelineName + "-link-" + linkName + "-agent")
}

// CreateLinkAgentDNSName returns the DNS name of the link agent
// for the given pipeline + link.
func CreateLinkAgentDNSName(pipelineName, linkName, namespace string) string {
	return fmt.Sprintf("%s.%s.svc", CreateLinkAgentName(pipelineName, linkName), namespace)
}

// CreateTaskAgentName returns the name of the task agent
// for the given pipeline + task.
func CreateTaskAgentName(pipelineName, taskName string) string {
	return FixupKubernetesName(pipelineName + "-task-" + taskName + "-agent")
}

// CreateEventRegistryAddress returns the address (host:port) of the event registry
// for the given pipeline.
func CreateEventRegistryAddress(pipelineName, namespace string) string {
	return net.JoinHostPort(CreatePipelineAgentDNSName(pipelineName, namespace), strconv.Itoa(constants.EventRegistryAPIPort))
}
