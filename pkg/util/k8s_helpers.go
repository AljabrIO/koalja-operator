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

package util

import (
	corev1 "k8s.io/api/core/v1"
)

// GetVolumeWithForPVC returns the volume in the given podspec that uses the given PVC name.
// If no such volume is found, false is returned.
func GetVolumeWithForPVC(podSpec *corev1.PodSpec, pvcName string) (*corev1.Volume, bool) {
	for i, v := range podSpec.Volumes {
		if pvcSrc := v.PersistentVolumeClaim; pvcSrc != nil {
			if pvcSrc.ClaimName == pvcName {
				return &podSpec.Volumes[i], true
			}
		}
	}
	return nil, false
}
