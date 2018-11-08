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

package protocols

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/dchest/uniuri"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/AljabrIO/koalja-operator/pkg/agent/task"
	"github.com/AljabrIO/koalja-operator/pkg/annotatedvalue"
	koalja "github.com/AljabrIO/koalja-operator/pkg/apis/koalja/v1alpha1"
	"github.com/AljabrIO/koalja-operator/pkg/fs"
	"github.com/AljabrIO/koalja-operator/pkg/util"
)

type fileInputBuilder struct{}
type fileOutputBuilder struct{}

func init() {
	task.RegisterExecutorInputBuilder(koalja.ProtocolFile, fileInputBuilder{})
	task.RegisterExecutorOutputBuilder(koalja.ProtocolFile, fileOutputBuilder{})
}

// Prepare input of a task for an input of File protocol.
func (b fileInputBuilder) Build(ctx context.Context, cfg task.ExecutorInputBuilderConfig, deps task.ExecutorInputBuilderDependencies, target *task.ExecutorInputBuilderTarget) error {
	uri := cfg.AnnotatedValue.GetData()
	deps.Log.Debug().
		Str("uri", uri).
		Str("input", cfg.InputSpec.Name).
		Str("task", cfg.TaskSpec.Name).
		Msg("Preparing file input")

	// Prepare readonly volume for URI
	resp, err := deps.FileSystem.CreateVolumeForRead(ctx, &fs.CreateVolumeForReadRequest{
		URI:   uri,
		Owner: &cfg.OwnerRef,
	})
	if err != nil {
		return maskAny(err)
	}
	// TODO handle case where node is different
	if nodeName := resp.GetNodeName(); nodeName != "" {
		target.NodeName = &nodeName
	}

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
	pvcName := util.FixupKubernetesName(fmt.Sprintf("%s-%s-%s-%s", cfg.Pipeline.GetName(), cfg.TaskSpec.Name, cfg.InputSpec.Name, uniuri.NewLen(6)))
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
	volName := util.FixupKubernetesName(fmt.Sprintf("input-%s", cfg.InputSpec.Name))
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

	// Map volume in container fs namespace
	mountPath := filepath.Join("/koalja", "inputs", cfg.InputSpec.Name)
	target.Container.VolumeMounts = append(target.Container.VolumeMounts, corev1.VolumeMount{
		Name:      volName,
		ReadOnly:  true,
		MountPath: mountPath,
	})

	// Create template data
	target.TemplateData = map[string]interface{}{
		"volumeName": resp.GetVolumeName(),
		"mountPath":  mountPath,
		"nodeName":   resp.GetNodeName(),
		"path":       filepath.Join(mountPath, resp.GetLocalPath()),
	}

	return nil
}

// Prepare output of a task for an output of File protocol.
func (b fileOutputBuilder) Build(ctx context.Context, cfg task.ExecutorOutputBuilderConfig, deps task.ExecutorOutputBuilderDependencies, target *task.ExecutorOutputBuilderTarget) error {
	// Prepare volume for output
	var nodeName string
	if target.NodeName != nil {
		nodeName = *target.NodeName
	}
	resp, err := deps.FileSystem.CreateVolumeForWrite(ctx, &fs.CreateVolumeForWriteRequest{
		EstimatedCapacity: 0,
		NodeName:          nodeName,
		Owner:             &cfg.OwnerRef,
	})
	if err != nil {
		return maskAny(err)
	}

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
	pvcName := util.FixupKubernetesName(fmt.Sprintf("%s-%s-%s-%s", cfg.Pipeline.GetName(), cfg.TaskSpec.Name, cfg.OutputSpec.Name, uniuri.NewLen(6)))
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
	volName := util.FixupKubernetesName(fmt.Sprintf("output-%s", cfg.OutputSpec.Name))
	vol := corev1.Volume{
		Name: volName,
		VolumeSource: corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: pvcName,
				ReadOnly:  false,
			},
		},
	}
	target.Pod.Spec.Volumes = append(target.Pod.Spec.Volumes, vol)

	// Map volume in container fs namespace
	mountPath := filepath.Join("/koalja", "outputs", cfg.OutputSpec.Name)
	target.Container.VolumeMounts = append(target.Container.VolumeMounts, corev1.VolumeMount{
		Name:      volName,
		ReadOnly:  false,
		MountPath: mountPath,
	})

	// Create template data
	localPath := "output"
	target.TemplateData = map[string]interface{}{
		"volumeName": resp.GetVolumeName(),
		"mountPath":  mountPath,
		"nodeName":   resp.GetNodeName(),
		"path":       filepath.Join(mountPath, localPath),
	}

	// Prepare output processor
	if cfg.OutputSpec.Ready.IsSucceeded() {
		target.OutputProcessor = &fileOutputProcessor{
			OutputName: cfg.OutputSpec.Name,
			VolumeName: resp.GetVolumeName(),
			MountPath:  mountPath,
			NodeName:   resp.GetNodeName(),
			LocalPath:  localPath,
		}
	}

	return nil
}

type fileOutputProcessor struct {
	OutputName string
	VolumeName string
	MountPath  string
	NodeName   string
	LocalPath  string
}

// Process a single file output
func (p *fileOutputProcessor) Process(ctx context.Context, cfg task.ExecutorOutputProcessorConfig, deps task.ExecutorOutputProcessorDependencies) error {
	log := deps.Log
	log.Debug().Msg("creating URI for output")
	resp, err := deps.FileSystem.CreateFileURI(ctx, &fs.CreateFileURIRequest{
		VolumeName: p.VolumeName,
		NodeName:   p.NodeName,
		LocalPath:  p.LocalPath,
		IsDir:      false, // TODO
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to create file URI")
		return maskAny(err)
	}
	// Publish annotated value
	av := annotatedvalue.AnnotatedValue{Data: resp.GetURI()}
	publishedAv, err := deps.OutputPublisher.Publish(ctx, p.OutputName, av, cfg.Snapshot)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create file URI")
		return maskAny(err)
	}
	log.Debug().
		Str("event-id", publishedAv.GetID()).
		Msg("published annotated value")

	return nil
}
