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

package s3

import (
	"context"
	"fmt"

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
	// Local path on the nodes where volumes will be mounted
	NodeRootMountPath string
	// Buckets to mount
	Buckets []BucketConfig
}

func createNodeDaemonLabels(name string) map[string]string {
	return map[string]string{
		"name": name,
		"app":  "koalja-s3-fs-node-daemon",
	}
}

// createOrUpdateNodeDaemonset ensures that the DaemonSet that runs all node processes
// is create & up to date.
func createOrUpdateNodeDaemonset(ctx context.Context, log zerolog.Logger, client client.Client,
	cfg nodeDaemonConfig, owner metav1.Object, scheme *runtime.Scheme) error {
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
				Spec: corev1.PodSpec{},
			},
		},
	}

	bidirectional := corev1.MountPropagationBidirectional
	privileged := true
	for bcIdx, bc := range cfg.Buckets {
		// Create container for each bucket
		volName := fmt.Sprintf("volume-%d", bcIdx)
		volMountPath := bc.NodeMountPoint(cfg.NodeRootMountPath)
		c := corev1.Container{
			Name:  fmt.Sprintf("mount-%d", bcIdx),
			Image: cfg.Image,
			Ports: []corev1.ContainerPort{
				corev1.ContainerPort{
					ContainerPort: int32(constants.AgentAPIPort),
				},
			},
			Env: []corev1.EnvVar{
				corev1.EnvVar{
					Name:  "BUCKET",
					Value: bc.Name,
				},
				corev1.EnvVar{
					Name:  "ENDPOINT",
					Value: bc.Endpoint,
				},
				corev1.EnvVar{
					Name: "AWS_ACCESS_KEY_ID",
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: bc.SecretName,
							},
							Key: constants.SecretKeyS3AccessKey,
						},
					},
				},
				corev1.EnvVar{
					Name: "AWS_SECRET_ACCESS_KEY",
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: bc.SecretName,
							},
							Key: constants.SecretKeyS3SecretKey,
						},
					},
				},
			},
			SecurityContext: &corev1.SecurityContext{
				Privileged: &privileged,
			},
			VolumeMounts: []corev1.VolumeMount{
				corev1.VolumeMount{
					Name:             volName,
					MountPath:        "/mnt/s3",
					MountPropagation: &bidirectional,
				},
			},
		}
		// Create volume for bucket
		v := corev1.Volume{
			Name: volName,
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: volMountPath,
					Type: &hostPathType,
				},
			},
		}
		// Add container & volume
		ds.Spec.Template.Spec.Containers = append(ds.Spec.Template.Spec.Containers, c)
		ds.Spec.Template.Spec.Volumes = append(ds.Spec.Template.Spec.Volumes, v)
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
