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
	"github.com/vincent-petithory/dataurl"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/AljabrIO/koalja-operator/pkg/agent/task"
	"github.com/AljabrIO/koalja-operator/pkg/annotatedvalue"
	"github.com/AljabrIO/koalja-operator/pkg/util"
)

type dataInputBuilder struct{}

func init() {
	task.RegisterExecutorInputBuilder(annotatedvalue.SchemeData, dataInputBuilder{})
}

// Prepare input of a task for an input that uses an URI with data scheme.
func (b dataInputBuilder) Build(ctx context.Context, cfg task.ExecutorInputBuilderConfig, deps task.ExecutorInputBuilderDependencies, target *task.ExecutorInputBuilderTarget) error {
	uri := cfg.AnnotatedValue.GetData()
	deps.Log.Debug().
		Int("sequence-index", cfg.AnnotatedValueIndex).
		Str("uri", uri).
		Str("input", cfg.InputSpec.Name).
		Str("task", cfg.TaskSpec.Name).
		Msg("Preparing data input value")

	// Decode data
	durl, err := dataurl.DecodeString(uri)
	if err != nil {
		deps.Log.Warn().Err(err).Msg("Failed to parse data URI")
		return maskAny(err)
	}

	// Create ConfigMap holding the data
	localPath := uniuri.New()
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            util.FixupKubernetesName(fmt.Sprintf("%s-%s-%s-%d-%s", cfg.Pipeline.Name, cfg.TaskSpec.Name, cfg.InputSpec.Name, cfg.AnnotatedValueIndex, uniuri.NewLen(8))),
			Namespace:       cfg.Pipeline.Namespace,
			OwnerReferences: []metav1.OwnerReference{cfg.OwnerRef},
		},
		BinaryData: map[string][]byte{
			localPath: durl.Data,
		},
	}
	if err := deps.Client.Create(ctx, configMap); err != nil {
		deps.Log.Warn().Err(err).Msg("Failed to create data ConfigMap")
		return maskAny(err)
	}
	// Add ConfigMap to resources for deletion list
	target.Resources = append(target.Resources, configMap)

	// Add volume for the pod
	volName := util.FixupKubernetesName(fmt.Sprintf("input-%s-%d", cfg.InputSpec.Name, cfg.AnnotatedValueIndex))
	defaultMode := int32(0444)
	vol := corev1.Volume{
		Name: volName,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: configMap.GetName(),
				},
				DefaultMode: &defaultMode,
			},
		},
	}
	target.Pod.Spec.Volumes = append(target.Pod.Spec.Volumes, vol)

	// Map volume in container fs namespace
	mountPath := filepath.Join("/koalja", "inputs", cfg.InputSpec.Name, strconv.Itoa(cfg.AnnotatedValueIndex))
	target.Container.VolumeMounts = append(target.Container.VolumeMounts, corev1.VolumeMount{
		Name:      volName,
		ReadOnly:  true,
		MountPath: mountPath,
	})

	// Create template data
	target.TemplateData = append(target.TemplateData, map[string]interface{}{
		"volumeName": "",
		"mountPath":  mountPath,
		"nodeName":   "",
		"path":       filepath.Join(mountPath, localPath),
	})

	return nil
}
