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

package input

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/dchest/uniuri"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/AljabrIO/koalja-operator/pkg/agent/task"
	"github.com/AljabrIO/koalja-operator/pkg/annotatedvalue"
	"github.com/AljabrIO/koalja-operator/pkg/fs"
	"github.com/AljabrIO/koalja-operator/pkg/util"
)

type fileInputBuilder struct{}

func init() {
	task.RegisterExecutorInputBuilder(annotatedvalue.SchemeFile, fileInputBuilder{})
}

// Prepare input of a task for an input that uses a koalja-file scheme.
func (b fileInputBuilder) Build(ctx context.Context, cfg task.ExecutorInputBuilderConfig, deps task.ExecutorInputBuilderDependencies, target *task.ExecutorInputBuilderTarget) error {
	uri := cfg.AnnotatedValue.GetData()
	deps.Log.Debug().
		Int("sequence-index", cfg.AnnotatedValueIndex).
		Str("uri", uri).
		Str("input", cfg.InputSpec.Name).
		Str("task", cfg.TaskSpec.Name).
		Msg("Preparing file input value")

	// Prepare readonly volume for URI
	resp, err := deps.FileSystem.CreateVolumeForRead(ctx, &fs.CreateVolumeForReadRequest{
		URI:       uri,
		Owner:     &cfg.OwnerRef,
		Namespace: cfg.Pipeline.GetNamespace(),
	})
	if err != nil {
		return maskAny(err)
	}
	// TODO handle case where node is different
	if nodeName := resp.GetNodeName(); nodeName != "" {
		target.NodeName = &nodeName
	}

	// Mount PVC or HostPath, depending on result
	volName := util.FixupKubernetesName(fmt.Sprintf("input-%s-%d", cfg.InputSpec.Name, cfg.AnnotatedValueIndex))
	if resp.GetVolumeName() != "" {
		// Get created PersistentVolume
		var pv corev1.PersistentVolume
		pvKey := client.ObjectKey{
			Name: resp.GetVolumeName(),
		}
		if err := deps.Client.Get(ctx, pvKey, &pv); err != nil {
			deps.Log.Warn().Err(err).Msg("Failed to get PersistentVolume")
			return maskAny(err)
		}

		// Add PV to resources for deletion list (if needed)
		if resp.DeleteAfterUse {
			target.Resources = append(target.Resources, &pv)
		}

		// Create PVC
		pvcName := util.FixupKubernetesName(fmt.Sprintf("%s-%s-%s-%d-%s", cfg.Pipeline.GetName(), cfg.TaskSpec.Name, cfg.InputSpec.Name, cfg.AnnotatedValueIndex, uniuri.NewLen(6)))
		storageClassName := pv.Spec.StorageClassName
		pvc := corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:            pvcName,
				Namespace:       cfg.Pipeline.GetNamespace(),
				OwnerReferences: []metav1.OwnerReference{cfg.OwnerRef},
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: pv.Spec.AccessModes,
				VolumeName:  resp.GetVolumeName(),
				Resources: corev1.ResourceRequirements{
					Requests: pv.Spec.Capacity,
				},
				StorageClassName: &storageClassName,
			},
		}
		if err := deps.Client.Create(ctx, &pvc); err != nil {
			return maskAny(err)
		}
		target.Resources = append(target.Resources, &pvc)

		// Add volume for the pod
		vol := corev1.Volume{
			Name: volName,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: pvcName,
					ReadOnly:  true,
				},
			},
		}
		target.Pod.Spec.Volumes = append(target.Pod.Spec.Volumes, vol)
	} else if resp.GetVolumeClaimName() != "" {
		// Add PVC to resources for deletion list (if needed)
		if resp.DeleteAfterUse {
			// Get created PersistentVolume
			var pvc corev1.PersistentVolumeClaim
			pvcKey := client.ObjectKey{
				Name:      resp.GetVolumeClaimName(),
				Namespace: cfg.Pipeline.GetNamespace(),
			}
			if err := deps.Client.Get(ctx, pvcKey, &pvc); err != nil {
				deps.Log.Warn().Err(err).Msg("Failed to get PersistentVolumeClaim")
				return maskAny(err)
			}
			target.Resources = append(target.Resources, &pvc)
		}

		// Add volume for the pod
		vol := corev1.Volume{
			Name: volName,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: resp.GetVolumeClaimName(),
					ReadOnly:  true,
				},
			},
		}
		target.Pod.Spec.Volumes = append(target.Pod.Spec.Volumes, vol)
	} else if resp.GetVolumePath() != "" {
		// Mount VolumePath as HostPath volume
		dirType := corev1.HostPathDirectoryOrCreate
		// Add volume for the pod
		vol := corev1.Volume{
			Name: volName,
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: resp.GetVolumePath(),
					Type: &dirType,
				},
			},
		}
		target.Pod.Spec.Volumes = append(target.Pod.Spec.Volumes, vol)
		// Ensure pod is schedule on node
		if nodeName := resp.GetNodeName(); nodeName != "" {
			if target.Pod.Spec.NodeName == "" {
				target.Pod.Spec.NodeName = nodeName
			} else if target.Pod.Spec.NodeName != nodeName {
				// Found conflict
				deps.Log.Error().
					Str("pod-nodeName", target.Pod.Spec.NodeName).
					Str("pod-nodeNameRequest", nodeName).
					Msg("Conflicting pod node spec")
				return maskAny(fmt.Errorf("Conflicting Node assignment"))
			}
		}
	} else {
		// No valid respond
		return maskAny(fmt.Errorf("FileSystem service return invalid response"))
	}

	// Map volume in container fs namespace
	mountPath := filepath.Join("/koalja", "inputs", cfg.InputSpec.Name, strconv.Itoa(cfg.AnnotatedValueIndex))
	target.Container.VolumeMounts = append(target.Container.VolumeMounts, corev1.VolumeMount{
		Name:      volName,
		ReadOnly:  true,
		MountPath: mountPath,
		SubPath:   resp.GetSubPath(),
	})

	// Create template data
	target.TemplateData = append(target.TemplateData, map[string]interface{}{
		"volumeName":      resp.GetVolumeName(),
		"volumeClaimName": resp.GetVolumeClaimName(),
		"volumePath":      resp.GetVolumePath(),
		"mountPath":       mountPath,
		"subPath":         resp.GetSubPath(),
		"nodeName":        resp.GetNodeName(),
		"path":            filepath.Join(mountPath, resp.GetLocalPath()),
	})

	return nil
}
