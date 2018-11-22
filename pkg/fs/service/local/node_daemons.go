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

package local

import (
	"context"
	"strconv"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/AljabrIO/koalja-operator/pkg/constants"
	"github.com/AljabrIO/koalja-operator/pkg/util"
	"github.com/rs/zerolog"
)

type nodeDaemonConfig struct {
	// Name of the daemonset
	Name string
	// Namespace to deploy the daemonset into
	Namespace string
	// Image to use
	Image string
	// Address of node registry
	RegistryAddress string
	// Local path on the nodes where volumes will be created
	NodeVolumePath string
}

func createNodeDaemonLabels(name string) map[string]string {
	return map[string]string{
		"name": name,
		"app":  "koalja-local-fs-node-daemon",
	}
}

// createOrUpdateNodeDaemonset ensures that the DaemonSet that runs all node processes
// is create & up to date.
func createOrUpdateNodeDaemonset(ctx context.Context, log zerolog.Logger, client client.Client,
	cfg nodeDaemonConfig, owner metav1.Object, scheme *runtime.Scheme) error {
	// Create container (spec)
	nodeVolumeName := "node-volume"
	c := corev1.Container{
		Name:  "node",
		Image: cfg.Image,
		Command: []string{
			"/apps/services",
			"filesystem",
			"node",
			"--registry=" + cfg.RegistryAddress,
			"--port=" + strconv.Itoa(constants.AgentAPIPort),
		},
		Ports: []corev1.ContainerPort{
			corev1.ContainerPort{
				ContainerPort: int32(constants.AgentAPIPort),
			},
		},
		Env: []corev1.EnvVar{
			corev1.EnvVar{
				Name: constants.EnvNodeName,
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "spec.nodeName",
					},
				},
			},
			corev1.EnvVar{
				Name: constants.EnvPodIP,
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "status.podIP",
					},
				},
			},
		},
		VolumeMounts: []corev1.VolumeMount{
			corev1.VolumeMount{
				Name:      nodeVolumeName,
				MountPath: cfg.NodeVolumePath,
			},
		},
	}
	// Create DaemonSet (resource)
	dsLabels := createNodeDaemonLabels(cfg.Name)
	hostPathType := corev1.HostPathDirectoryOrCreate
	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cfg.Name,
			Namespace: cfg.Namespace,
			Labels:    dsLabels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: dsLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: dsLabels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						c,
					},
					Volumes: []corev1.Volume{
						corev1.Volume{
							Name: nodeVolumeName,
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: cfg.NodeVolumePath,
									Type: &hostPathType,
								},
							},
						},
					},
				},
			},
		},
	}

	// Set owner ref
	if err := controllerutil.SetControllerReference(owner, ds, scheme); err != nil {
		return maskAny(err)
	}

	// Create or update daemonset
	if err := util.EnsureDaemonSet(ctx, log, client, ds, "Node Daemon"); err != nil {
		return maskAny(err)
	}

	return nil
}
