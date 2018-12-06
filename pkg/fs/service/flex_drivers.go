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

package service

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	agents "github.com/AljabrIO/koalja-operator/pkg/apis/agents/v1alpha1"
	"github.com/AljabrIO/koalja-operator/pkg/util"
)

func createFlexDriverInstallDaemonLabels(name string) map[string]string {
	return map[string]string{
		"name": name,
		"app":  "koalja-fs-flex-driver-install",
	}
}

// createFlexDriverInstallDaemonsets ensures that the DaemonSets that install
// all flex volume drivers are created & up to date.
func (s *Service) createFlexDriverInstallDaemonsets(ctx context.Context) error {
	// Fetch all FlexVolumeDrivers
	var list agents.FlexVolumeDriverList
	if err := s.c.List(ctx, nil, &list); err != nil {
		return maskAny(err)
	}

	for _, item := range list.Items {
		driverName := item.GetName()
		if err := s.createFlexDriverInstallDaemonset(ctx, driverName, *item.Spec.Container); err != nil {
			s.log.Error().Err(err).Str("driver", driverName).Msg("Failed to install Flex Volume Driver Installer DaemonSet")
			return maskAny(err)
		}
	}

	return nil
}

// createFlexDriverInstallDaemonsets ensures that the DaemonSets that install
// all flex volume drivers are created & up to date.
func (s *Service) createFlexDriverInstallDaemonset(ctx context.Context, driverName string, container corev1.Container) error {
	// Create DaemonSet (resource)
	dsLabels := createFlexDriverInstallDaemonLabels(driverName)
	privileged := true
	if container.Name == "" {
		container.Name = "installer"
	}
	container.SecurityContext = &corev1.SecurityContext{
		Privileged: &privileged,
	}
	volName := "volume-plugins"
	localPluginDir := "/flex-plugin-dir"
	container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
		Name:      volName,
		MountPath: localPluginDir,
	})
	container.Env = append(container.Env, corev1.EnvVar{
		Name:  "FLEX_VOLUME_PLUGIN_DIR",
		Value: localPluginDir,
	})
	hostPathType := corev1.HostPathDirectoryOrCreate
	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      driverName,
			Namespace: s.ns,
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
					Containers: []corev1.Container{container},
					Volumes: []corev1.Volume{
						corev1.Volume{
							Name: volName,
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: s.volumePluginDir,
									Type: &hostPathType,
								},
							},
						},
					},
				},
			},
		},
	}

	// Create or update daemonset
	if err := util.EnsureDaemonSet(ctx, s.log, s.c, ds, "Flex Driver Installer Daemon"); err != nil {
		return maskAny(err)
	}

	return nil
}
