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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/AljabrIO/koalja-operator/pkg/constants"
	"github.com/AljabrIO/koalja-operator/pkg/util"
	"github.com/rs/zerolog"
)

type pvConfig struct {
	// Name of the PersistentVolume/Claim
	Name string
	// Namespace to deploy the PVC into
	Namespace string
	// StorageClassName used for the PV/PVC
	StorageClassName string
	// Bucket to mount
	Bucket BucketConfig
}

func createPVLabels(name string) map[string]string {
	return map[string]string{
		"name": name,
		"app":  "koalja-s3-fs-storage-volume",
	}
}

func createPVCapacity() corev1.ResourceList {
	q := resource.NewScaledQuantity(8, resource.Peta)
	return corev1.ResourceList{
		corev1.ResourceStorage: *q,
	}
}

// createOrUpdatePersistentVolume ensures that the PersistentVolume & Claim for the given
// config is created & up to date.
func createOrUpdatePersistentVolume(ctx context.Context, log zerolog.Logger, client client.Client, cfg pvConfig) error {
	// Create PersistentVolume (resource)
	pvLabels := createPVLabels(cfg.Name)
	pv := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:   cfg.Name,
			Labels: pvLabels,
		},
		Spec: corev1.PersistentVolumeSpec{
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany},
			Capacity:         createPVCapacity(),
			StorageClassName: cfg.StorageClassName,
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				FlexVolume: &corev1.FlexPersistentVolumeSource{
					Driver: constants.FlexVolumeS3VendorName + "/" + constants.FlexVolumeS3DriverName,
					SecretRef: &corev1.SecretReference{
						Name:      cfg.Bucket.SecretName,
						Namespace: cfg.Namespace,
					},
					Options: map[string]string{
						constants.FlexVolumeOptionS3BucketKey:   cfg.Bucket.Name,
						constants.FlexVolumeOptionS3EndpointKey: cfg.Bucket.EndpointAsURL(),
					},
				},
			},
		},
	}

	// Create or update PV
	if err := util.EnsurePersistentVolume(ctx, log, client, pv, "Storage Flex PersistentVolume"); err != nil {
		return maskAny(err)
	}

	return nil
}

// createOrUpdatePersistentVolumeClaim ensures that the PersistentVolumeClaim for the given
// config is created & up to date.
func createOrUpdatePersistentVolumeClaim(ctx context.Context, log zerolog.Logger, client client.Client, cfg pvConfig) error {
	// Create PVC
	storageClassName := cfg.StorageClassName
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cfg.Name,
			Namespace: cfg.Namespace,
			Labels:    createPVLabels(cfg.Name),
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany},
			Resources: corev1.ResourceRequirements{
				Requests: createPVCapacity(),
			},
			StorageClassName: &storageClassName,
			VolumeName:       cfg.Name,
		},
	}

	// Create or update PVC
	if err := util.EnsurePersistentVolumeClaim(ctx, log, client, pvc, "Storage Flex PersistentVolumeClaim"); err != nil {
		return maskAny(err)
	}

	return nil
}
