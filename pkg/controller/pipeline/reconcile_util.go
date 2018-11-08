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
	"fmt"
	"net"
	"strconv"
	"time"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/AljabrIO/koalja-operator/pkg/constants"
	"github.com/AljabrIO/koalja-operator/pkg/util"
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

// SetAgentContainerDefaults applies default values to a container used to run an agent
func SetAgentContainerDefaults(c *corev1.Container, addHTTPPort bool) {
	if c.Name == "" {
		c.Name = "agent"
	}
	if len(c.Ports) == 0 {
		c.Ports = []corev1.ContainerPort{
			corev1.ContainerPort{
				Name:          "grpc-api",
				ContainerPort: constants.AgentAPIPort,
			},
		}
		if addHTTPPort {
			c.Ports = []corev1.ContainerPort{
				corev1.ContainerPort{
					Name:          "http-api",
					ContainerPort: constants.AgentAPIHTTPPort,
				},
			}
		}
	}
}

// SetAnnotatedValueRegistryContainerDefaults applies default values to a container used to run an annotated value registry
func SetAnnotatedValueRegistryContainerDefaults(c *corev1.Container) {
	if c.Name == "" {
		c.Name = "registry"
	}
	if len(c.Ports) == 0 {
		c.Ports = []corev1.ContainerPort{
			corev1.ContainerPort{
				Name:          "grpc-registry",
				ContainerPort: constants.AnnotatedValueRegistryAPIPort,
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
	return util.FixupKubernetesName(pipelineName + "-p")
}

// CreatePipelineAgentDNSName returns the DNS of the pipeline agent
// for the given pipeline.
func CreatePipelineAgentDNSName(pipelineName, namespace string) string {
	return fmt.Sprintf("%s.%s.svc", CreatePipelineAgentName(pipelineName), namespace)
}

// CreateLinkAgentName returns the name of the link agent
// for the given pipeline + link.
func CreateLinkAgentName(pipelineName, linkName string) string {
	return util.FixupKubernetesName(pipelineName + "-l-" + linkName)
}

// CreateLinkAgentDNSName returns the DNS name of the link agent
// for the given pipeline + link.
func CreateLinkAgentDNSName(pipelineName, linkName, namespace string) string {
	return fmt.Sprintf("%s.%s.svc", CreateLinkAgentName(pipelineName, linkName), namespace)
}

// CreateLinkAgentEventSourceAddress returns the address (host:port) of the event source
// for the given link int the given pipeline.
func CreateLinkAgentEventSourceAddress(pipelineName, linkName, namespace string) string {
	return net.JoinHostPort(CreateLinkAgentDNSName(pipelineName, linkName, namespace), strconv.Itoa(constants.AgentAPIPort))
}

// CreateLinkAgentEventPublisherAddress returns the address (host:port) of the event publisher
// for the given link int the given pipeline.
func CreateLinkAgentEventPublisherAddress(pipelineName, linkName, namespace string) string {
	return net.JoinHostPort(CreateLinkAgentDNSName(pipelineName, linkName, namespace), strconv.Itoa(constants.AgentAPIPort))
}

// CreateTaskAgentName returns the name of the task agent
// for the given pipeline + task.
func CreateTaskAgentName(pipelineName, taskName string) string {
	return util.FixupKubernetesName(pipelineName + "-t-" + taskName)
}

// CreateTaskAgentDNSName returns the DNS name of the task agent
// for the given pipeline + task.
func CreateTaskAgentDNSName(pipelineName, taskName, namespace string) string {
	return fmt.Sprintf("%s.%s.svc", CreateTaskAgentName(pipelineName, taskName), namespace)
}

// CreateTaskExecutorServiceName returns the name of the task executor service
// for the given pipeline + task.
func CreateTaskExecutorServiceName(pipelineName, taskName string) string {
	return util.FixupKubernetesName(pipelineName + "-x-" + taskName)
}

// CreateAgentRegistryAddress returns the address (host:port) of the agent registry
// for the given pipeline.
func CreateAgentRegistryAddress(pipelineName, namespace string) string {
	return net.JoinHostPort(CreatePipelineAgentDNSName(pipelineName, namespace), strconv.Itoa(constants.AgentAPIPort))
}

// CreateAnnotatedValueRegistryAddress returns the address (host:port) of the annotated value registry
// for the given pipeline.
func CreateAnnotatedValueRegistryAddress(pipelineName, namespace string) string {
	return net.JoinHostPort(CreatePipelineAgentDNSName(pipelineName, namespace), strconv.Itoa(constants.AnnotatedValueRegistryAPIPort))
}

// CreatePipelineAgentEventPublisherAddress returns the address (host:port) of the event publisher of the pipeline agent
// for the given pipeline.
func CreatePipelineAgentEventPublisherAddress(pipelineName, namespace string) string {
	return net.JoinHostPort(CreatePipelineAgentDNSName(pipelineName, namespace), strconv.Itoa(constants.AgentAPIPort))
}

// CreateServiceDNSName returns the DNS name of the given service.
func CreateServiceDNSName(s corev1.Service) string {
	return fmt.Sprintf("%s.%s.svc", s.GetName(), s.GetNamespace())
}

// CreateServiceAddress returns the address of the given service.
func CreateServiceAddress(s corev1.Service) string {
	return net.JoinHostPort(CreateServiceDNSName(s), strconv.Itoa(int(s.Spec.Ports[0].Port)))
}

// CreatePipelineAgentsServiceAccountName returns the name of the ServiceAccount
// used as identity of all pipeline, link & task agents of the pipeline.
func CreatePipelineAgentsServiceAccountName(pipelineName string) string {
	return util.FixupKubernetesName(pipelineName + "-agents")
}

// CreatePipelineExecutorsServiceAccountName returns the name of the ServiceAccount
// used as identity of all task executors of the pipeline.
func CreatePipelineExecutorsServiceAccountName(pipelineName string) string {
	return util.FixupKubernetesName(pipelineName + "-executors")
}

// CreatePipelineAgentsRoleName returns the name of the Role
// that provides permissions for to the ServiceAccount that is used as identity of all
// pipeline, link & task agents of the pipeline.
func CreatePipelineAgentsRoleName(pipelineName string) string {
	return util.FixupKubernetesName(pipelineName + "-agents")
}

// CreatePipelineAgentsClusterRoleName returns the name of the ClusterRole
// that provides permissions for to the ServiceAccount that is used as identity of all
// pipeline, link & task agents of the pipeline.
func CreatePipelineAgentsClusterRoleName(pipelineName, namespace string) string {
	return util.FixupKubernetesName(pipelineName + "-agents-" + namespace)
}

// CreatePipelineExecutorsRoleName returns the name of the Role
// that provides permissions for to the ServiceAccount that is used as identity of all
// task executors of the pipeline.
func CreatePipelineExecutorsRoleName(pipelineName string) string {
	return util.FixupKubernetesName(pipelineName + "-executors")
}

// CreatePipelineAgentsRoleBindingName returns the name of the RoleBinding
// that provides permissions for to the ServiceAccount that is used as identity of all
// pipeline, link & task agents of the pipeline.
func CreatePipelineAgentsRoleBindingName(pipelineName string) string {
	return util.FixupKubernetesName(pipelineName + "-agents")
}

// CreatePipelineAgentsClusterRoleBindingName returns the name of the ClusterRoleBinding
// that provides permissions for to the ServiceAccount that is used as identity of all
// pipeline, link & task agents of the pipeline.
func CreatePipelineAgentsClusterRoleBindingName(pipelineName, namespace string) string {
	return util.FixupKubernetesName(pipelineName + "-agents-" + namespace)
}

// CreatePipelineExecutorsRoleBindingName returns the name of the ClusterRoleBinding
// that provides permissions for to the ServiceAccount that is used as identity of all
// task executors of the pipeline.
func CreatePipelineExecutorsRoleBindingName(pipelineName string) string {
	return util.FixupKubernetesName(pipelineName + "-executors")
}
