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
	"context"

	"github.com/rs/zerolog"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// EnsureConfigMap creates of updates a ConfigMap.
func EnsureConfigMap(ctx context.Context, log zerolog.Logger, client client.Client, cm *corev1.ConfigMap, description string) error {
	// Check if ConfigMap already exists
	found := &corev1.ConfigMap{}
	if err := client.Get(ctx, types.NamespacedName{Name: cm.Name, Namespace: cm.Namespace}, found); err != nil && errors.IsNotFound(err) {
		log.Info().Msgf("Creating %s", description)
		if err := client.Create(ctx, cm); err != nil {
			log.Error().Err(err).Msgf("Failed to create %s", description)
			return err
		}
	} else if err != nil {
		return err
	} else {
		// Update the found object and write the result back if there are any changes
		if diff := ConfigMapEqual(*cm, *found); len(diff) > 0 {
			found.Data = cm.Data
			found.BinaryData = cm.BinaryData
			log.Info().Interface("diff", diff).Msgf("Updating %s", description)
			if err := client.Update(ctx, found); err != nil {
				return err
			}
		}
	}

	return nil
}

// EnsureRole creates of updates a Role.
func EnsureRole(ctx context.Context, log zerolog.Logger, client client.Client, role *rbacv1.Role, description string) error {
	// Check if Role already exists
	found := &rbacv1.Role{}
	if err := client.Get(ctx, types.NamespacedName{Name: role.Name, Namespace: role.Namespace}, found); err != nil && errors.IsNotFound(err) {
		log.Info().Msgf("Creating %s", description)
		if err := client.Create(ctx, role); err != nil {
			log.Error().Err(err).Msgf("Failed to create %s", description)
			return err
		}
	} else if err != nil {
		return err
	} else {
		// Update the found object and write the result back if there are any changes
		if diff := RoleEqual(*role, *found); len(diff) > 0 {
			found.Rules = role.Rules
			log.Info().Interface("diff", diff).Msgf("Updating %s", description)
			if err := client.Update(ctx, found); err != nil {
				return err
			}
		}
	}

	return nil
}

// EnsureRoleBinding creates of updates a RoleBinding.
func EnsureRoleBinding(ctx context.Context, log zerolog.Logger, client client.Client, roleBinding *rbacv1.RoleBinding, description string) error {
	// Check if RoleBinding already exists
	found := &rbacv1.RoleBinding{}
	if err := client.Get(ctx, types.NamespacedName{Name: roleBinding.Name, Namespace: roleBinding.Namespace}, found); err != nil && errors.IsNotFound(err) {
		log.Info().Msgf("Creating %s", description)
		if err := client.Create(ctx, roleBinding); err != nil {
			log.Error().Err(err).Msgf("Failed to create %s", description)
			return err
		}
	} else if err != nil {
		return err
	} else {
		// Update the found object and write the result back if there are any changes
		if diff := RoleBindingEqual(*roleBinding, *found); len(diff) > 0 {
			found.Subjects = roleBinding.Subjects
			found.RoleRef = roleBinding.RoleRef
			log.Info().Interface("diff", diff).Msgf("Updating %s", description)
			if err := client.Update(ctx, found); err != nil {
				return err
			}
		}
	}

	return nil
}

// EnsureClusterRole creates of updates a Role.
func EnsureClusterRole(ctx context.Context, log zerolog.Logger, client client.Client, role *rbacv1.ClusterRole, description string) error {
	// Check if Role already exists
	found := &rbacv1.ClusterRole{}
	if err := client.Get(ctx, types.NamespacedName{Name: role.Name, Namespace: role.Namespace}, found); err != nil && errors.IsNotFound(err) {
		log.Info().Msgf("Creating %s", description)
		if err := client.Create(ctx, role); err != nil {
			log.Error().Err(err).Msgf("Failed to create %s", description)
			return err
		}
	} else if err != nil {
		return err
	} else {
		// Update the found object and write the result back if there are any changes
		if diff := ClusterRoleEqual(*role, *found); len(diff) > 0 {
			found.Rules = role.Rules
			log.Info().Interface("diff", diff).Msgf("Updating %s", description)
			if err := client.Update(ctx, found); err != nil {
				return err
			}
		}
	}

	return nil
}

// EnsureClusterRoleBinding creates of updates a RoleBinding.
func EnsureClusterRoleBinding(ctx context.Context, log zerolog.Logger, client client.Client, roleBinding *rbacv1.ClusterRoleBinding, description string) error {
	// Check if RoleBinding already exists
	found := &rbacv1.ClusterRoleBinding{}
	if err := client.Get(ctx, types.NamespacedName{Name: roleBinding.Name, Namespace: roleBinding.Namespace}, found); err != nil && errors.IsNotFound(err) {
		log.Info().Msgf("Creating %s", description)
		if err := client.Create(ctx, roleBinding); err != nil {
			log.Error().Err(err).Msgf("Failed to create %s", description)
			return err
		}
	} else if err != nil {
		return err
	} else {
		// Update the found object and write the result back if there are any changes
		if diff := ClusterRoleBindingEqual(*roleBinding, *found); len(diff) > 0 {
			found.Subjects = roleBinding.Subjects
			found.RoleRef = roleBinding.RoleRef
			log.Info().Interface("diff", diff).Msgf("Updating %s", description)
			if err := client.Update(ctx, found); err != nil {
				return err
			}
		}
	}

	return nil
}

// EnsureService creates of updates a Service.
func EnsureService(ctx context.Context, log zerolog.Logger, client client.Client, service *corev1.Service, description string) error {
	// Check if Service already exists
	found := &corev1.Service{}
	if err := client.Get(ctx, types.NamespacedName{Name: service.Name, Namespace: service.Namespace}, found); err != nil && errors.IsNotFound(err) {
		log.Info().Msgf("Creating %s", description)
		if err := client.Create(ctx, service); err != nil {
			log.Error().Err(err).Msgf("Failed to create %s", description)
			return err
		}
	} else if err != nil {
		return err
	} else {
		// Update the found object and write the result back if there are any changes
		if diff := ServiceEqual(*service, *found); len(diff) > 0 {
			service.Spec.ClusterIP = found.Spec.ClusterIP
			found.Spec = service.Spec
			log.Info().Interface("diff", diff).Msgf("Updating %s", description)
			if err := client.Update(ctx, found); err != nil {
				return err
			}
		}
	}

	return nil
}

// EnsureServiceAccount creates of updates a ServiceAccount.
func EnsureServiceAccount(ctx context.Context, log zerolog.Logger, client client.Client, serviceAccount *corev1.ServiceAccount, description string) error {
	// Check if Service already exists
	found := &corev1.ServiceAccount{}
	if err := client.Get(ctx, types.NamespacedName{Name: serviceAccount.Name, Namespace: serviceAccount.Namespace}, found); err != nil && errors.IsNotFound(err) {
		log.Info().Msgf("Creating %s", description)
		if err := client.Create(ctx, serviceAccount); err != nil {
			log.Error().Err(err).Msgf("Failed to create %s", description)
			return err
		}
	} else if err != nil {
		return err
	} else {
		// Update the found object and write the result back if there are any changes
		if diff := ServiceAccountEqual(*serviceAccount, *found); len(diff) > 0 {
			found.Secrets = serviceAccount.Secrets
			found.ImagePullSecrets = serviceAccount.ImagePullSecrets
			found.AutomountServiceAccountToken = serviceAccount.AutomountServiceAccountToken
			log.Info().Interface("diff", diff).Msgf("Updating %s", description)
			if err := client.Update(ctx, found); err != nil {
				return err
			}
		}
	}

	return nil
}

// EnsureStatefulSet creates of updates a StatefulSet.
func EnsureStatefulSet(ctx context.Context, log zerolog.Logger, client client.Client, statefulSet *appsv1.StatefulSet, description string) error {
	// Check if StatefulSet already exists
	found := &appsv1.StatefulSet{}
	if err := client.Get(ctx, types.NamespacedName{Name: statefulSet.Name, Namespace: statefulSet.Namespace}, found); err != nil && errors.IsNotFound(err) {
		log.Info().Msgf("Creating %s", description)
		if err := client.Create(ctx, statefulSet); err != nil {
			log.Error().Err(err).Msgf("Failed to create %s", description)
			return err
		}
	} else if err != nil {
		return err
	} else {
		// Update the found object and write the result back if there are any changes
		if diff := StatefulSetEqual(*statefulSet, *found); len(diff) > 0 {
			found.Spec = statefulSet.Spec
			log.Info().Interface("diff", diff).Msgf("Updating %s", description)
			if err := client.Update(ctx, found); err != nil {
				return err
			}
		}
	}

	return nil
}

// EnsureDaemonSet creates of updates a DaemonSet.
func EnsureDaemonSet(ctx context.Context, log zerolog.Logger, client client.Client, daemonSet *appsv1.DaemonSet, description string) error {
	// Check if StatefulSet already exists
	found := &appsv1.DaemonSet{}
	if err := client.Get(ctx, types.NamespacedName{Name: daemonSet.Name, Namespace: daemonSet.Namespace}, found); err != nil && errors.IsNotFound(err) {
		log.Info().Msgf("Creating %s", description)
		if err := client.Create(ctx, daemonSet); err != nil {
			log.Error().Err(err).Msgf("Failed to create %s", description)
			return err
		}
	} else if err != nil {
		return err
	} else {
		// Update the found object and write the result back if there are any changes
		if diff := DaemonSetEqual(*daemonSet, *found); len(diff) > 0 {
			found.Spec = daemonSet.Spec
			log.Info().Interface("diff", diff).Msgf("Updating %s", description)
			if err := client.Update(ctx, found); err != nil {
				return err
			}
		}
	}

	return nil
}

// EnsurePersistentVolume creates of updates a PersistentVolume.
func EnsurePersistentVolume(ctx context.Context, log zerolog.Logger, client client.Client, pv *corev1.PersistentVolume, description string) error {
	// Check if PersistentVolume already exists
	found := &corev1.PersistentVolume{}
	if err := client.Get(ctx, types.NamespacedName{Name: pv.Name}, found); err != nil && errors.IsNotFound(err) {
		log.Info().Msgf("Creating %s", description)
		if err := client.Create(ctx, pv); err != nil {
			log.Error().Err(err).Msgf("Failed to create %s", description)
			return err
		}
	} else if err != nil {
		return err
	} else {
		// Update the found object and write the result back if there are any changes
		if diff := PersistentVolumeEqual(*pv, *found); len(diff) > 0 {
			found.Spec = pv.Spec
			log.Info().Interface("diff", diff).Msgf("Updating %s", description)
			if err := client.Update(ctx, found); err != nil {
				return err
			}
		}
	}

	return nil
}

// EnsurePersistentVolumeClaim creates of updates a PersistentVolumeClaim.
func EnsurePersistentVolumeClaim(ctx context.Context, log zerolog.Logger, client client.Client, pvc *corev1.PersistentVolumeClaim, description string) error {
	// Check if PersistentVolume already exists
	found := &corev1.PersistentVolumeClaim{}
	if err := client.Get(ctx, types.NamespacedName{Name: pvc.Name, Namespace: pvc.Namespace}, found); err != nil && errors.IsNotFound(err) {
		log.Info().Msgf("Creating %s", description)
		if err := client.Create(ctx, pvc); err != nil {
			log.Error().Err(err).Msgf("Failed to create %s", description)
			return err
		}
	} else if err != nil {
		return err
	} else {
		// Update the found object and write the result back if there are any changes
		if diff := PersistentVolumeClaimEqual(*pvc, *found); len(diff) > 0 {
			found.Spec = pvc.Spec
			log.Info().Interface("diff", diff).Msgf("Updating %s", description)
			if err := client.Update(ctx, found); err != nil {
				return err
			}
		}
	}

	return nil
}
