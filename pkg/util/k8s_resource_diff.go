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
	"fmt"
	"reflect"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Diff is a human readable change notification
type Diff string

// BoolEqual returns zero-length result when the given bools are the same.
func BoolEqual(prefix string, spec, actual bool) []Diff {
	if spec == actual {
		return nil
	}
	return []Diff{Diff(prefix + fmt.Sprintf(" expected %v got %v", spec, actual))}
}

// IntEqual returns zero-length result when the given ints are the same.
func IntEqual(prefix string, spec, actual int) []Diff {
	if spec == actual {
		return nil
	}
	return []Diff{Diff(prefix + fmt.Sprintf(" expected %d got %d", spec, actual))}
}

// Int32Equal returns zero-length result when the given ints are the same.
func Int32Equal(prefix string, spec, actual int32) []Diff {
	if spec == actual {
		return nil
	}
	return []Diff{Diff(prefix + fmt.Sprintf(" expected %d got %d", spec, actual))}
}

// StringEqual returns zero-length result when the given strings are the same.
func StringEqual(prefix string, spec, actual string) []Diff {
	if spec == actual {
		return nil
	}
	return []Diff{Diff(prefix + fmt.Sprintf(" expected '%s' got '%s'", spec, actual))}
}

// StringsEqual returns zero-length result when the given strings are the same.
func StringsEqual(prefix string, spec, actual []string) []Diff {
	if reflect.DeepEqual(spec, actual) {
		return nil
	}
	return []Diff{Diff(prefix)}
}

// LabelsEqual returns zero-length result when the given labels are the same.
func LabelsEqual(prefix string, spec, actual map[string]string) []Diff {
	var result []Diff
	for sk, sv := range spec {
		av := actual[sk]
		if sv != av {
			result = append(result, Diff(fmt.Sprintf("%s.%s expected '%s' got '%s'", prefix, sk, sv, av)))
		}
	}
	return result
}

// SelectorEqual returns zero-length result when the given selectors are the same.
func SelectorEqual(prefix string, spec, actual map[string]string) []Diff {
	var result []Diff
	for sk, sv := range spec {
		av := actual[sk]
		if sv != av {
			result = append(result, Diff(fmt.Sprintf("%s.%s expected '%s' got '%s'", prefix, sk, sv, av)))
		}
	}
	if len(actual) > len(spec) {
		result = append(result, Diff(fmt.Sprintf("additional selectors found in %s", prefix)))
	}
	return result
}

// ObjectMetaEqual returns zero-length result when the given objects have the same name,
// namespace and labels.
func ObjectMetaEqual(prefix string, spec, actual metav1.ObjectMeta) []Diff {
	return append(append(append(
		StringEqual(prefix+".name", spec.GetName(), actual.GetName()),
		StringEqual(prefix+".namespace", spec.GetNamespace(), actual.GetNamespace())...),
		LabelsEqual(prefix+".annotations", spec.GetAnnotations(), actual.GetAnnotations())...),
		LabelsEqual(prefix+".labels", spec.GetLabels(), actual.GetLabels())...)
}

// ServiceAccountEqual returns zero-length result when the given objects have the same specs.
func ServiceAccountEqual(spec, actual corev1.ServiceAccount) []Diff {
	return ObjectMetaEqual("metadata", spec.ObjectMeta, actual.ObjectMeta)
}

// RoleEqual returns zero-length result when the given objects have the same specs.
func RoleEqual(spec, actual rbacv1.Role) []Diff {
	return append(ObjectMetaEqual("metadata", spec.ObjectMeta, actual.ObjectMeta),
		PolicyRulesEqual("rules", spec.Rules, actual.Rules)...)
}

// ClusterRoleEqual returns zero-length result when the given objects have the same specs.
func ClusterRoleEqual(spec, actual rbacv1.ClusterRole) []Diff {
	return append(ObjectMetaEqual("metadata", spec.ObjectMeta, actual.ObjectMeta),
		PolicyRulesEqual("rules", spec.Rules, actual.Rules)...)
}

// PolicyRulesEqual returns zero-length result when the given objects have the same specs.
func PolicyRulesEqual(prefix string, spec, actual []rbacv1.PolicyRule) []Diff {
	for i, sr := range spec {
		var result []Diff
		found := false
		for j, ar := range actual {
			if diff := PolicyRuleEqual(prefix, sr, ar); len(diff) == 0 {
				// Found match
				found = true
				break
			} else {
				if j < i {
					result = diff
				}
			}
		}
		if !found {
			if len(result) > 0 {
				return result
			}
			return []Diff{Diff(prefix + fmt.Sprintf("missing policy rule %d: %v", i, sr))}
		}
	}
	return nil
}

// PolicyRuleEqual returns zero-length result when the given objects have the same specs.
func PolicyRuleEqual(prefix string, spec, actual rbacv1.PolicyRule) []Diff {
	return append(StringsEqual(prefix+".verbs", spec.Verbs, actual.Verbs),
		append(StringsEqual(prefix+".apiGroups", spec.APIGroups, actual.APIGroups),
			append(StringsEqual(prefix+".resources", spec.Resources, actual.Resources),
				append(StringsEqual(prefix+".resourceNames", spec.ResourceNames, actual.ResourceNames),
					StringsEqual(prefix+".nonResourceURLs", spec.NonResourceURLs, actual.NonResourceURLs)...)...)...)...)
}

// RoleBindingEqual returns zero-length result when the given objects have the same specs.
func RoleBindingEqual(spec, actual rbacv1.RoleBinding) []Diff {
	return append(ObjectMetaEqual("metadata", spec.ObjectMeta, actual.ObjectMeta),
		append(SubjectsEqual("subjects", spec.Subjects, actual.Subjects),
			RoleRefEqual("roleRef", spec.RoleRef, actual.RoleRef)...)...)
}

// ClusterRoleBindingEqual returns zero-length result when the given objects have the same specs.
func ClusterRoleBindingEqual(spec, actual rbacv1.ClusterRoleBinding) []Diff {
	return append(ObjectMetaEqual("metadata", spec.ObjectMeta, actual.ObjectMeta),
		append(SubjectsEqual("subjects", spec.Subjects, actual.Subjects),
			RoleRefEqual("roleRef", spec.RoleRef, actual.RoleRef)...)...)
}

// RoleRefEqual returns zero-length result when the given objects have the same specs.
func RoleRefEqual(prefix string, spec, actual rbacv1.RoleRef) []Diff {
	return append(StringEqual(prefix+".apiGroup", spec.APIGroup, actual.APIGroup),
		append(StringEqual(prefix+".kind", spec.Kind, actual.Kind),
			StringEqual(prefix+".name", spec.Name, actual.Name)...)...)
}

// SubjectsEqual returns zero-length result when the given objects have the same specs.
func SubjectsEqual(prefix string, spec, actual []rbacv1.Subject) []Diff {
	for i, sr := range spec {
		var result []Diff
		found := false
		for j, ar := range actual {
			if diff := SubjectEqual(prefix, sr, ar); len(diff) == 0 {
				// Found match
				found = true
				break
			} else {
				if j < i {
					result = diff
				}
			}
		}
		if !found {
			if len(result) > 0 {
				return result
			}
			return []Diff{Diff(prefix + fmt.Sprintf("missing subject %d: %v", i, sr))}
		}
	}
	return nil
}

// SubjectEqual returns zero-length result when the given objects have the same specs.
func SubjectEqual(prefix string, spec, actual rbacv1.Subject) []Diff {
	return append(StringEqual(prefix+".apiGroup", spec.APIGroup, actual.APIGroup),
		append(StringEqual(prefix+".kind", spec.Kind, actual.Kind),
			append(StringEqual(prefix+".name", spec.Name, actual.Name),
				StringEqual(prefix+".namespace", spec.Namespace, actual.Namespace)...)...)...)
}

// StatefulSetEqual returns zero-length result when the given objects have the same specs.
func StatefulSetEqual(spec, actual appsv1.StatefulSet) []Diff {
	return append(
		ObjectMetaEqual("metadata", spec.ObjectMeta, actual.ObjectMeta),
		StatefulSetSpecEqual("spec", spec.Spec, actual.Spec)...)
}

// StatefulSetSpecEqual returns zero-length result when the given objects have the same pod template specs.
func StatefulSetSpecEqual(prefix string, spec, actual appsv1.StatefulSetSpec) []Diff {
	return PodTemplateSpecEqual(prefix+".template", spec.Template, actual.Template)
}

// DaemonSetEqual returns zero-length result when the given objects have the same specs.
func DaemonSetEqual(spec, actual appsv1.DaemonSet) []Diff {
	return append(
		ObjectMetaEqual("metadata", spec.ObjectMeta, actual.ObjectMeta),
		DaemonSetSpecEqual("spec", spec.Spec, actual.Spec)...)
}

// DaemonSetSpecEqual returns zero-length result when the given objects have the same pod template specs.
func DaemonSetSpecEqual(prefix string, spec, actual appsv1.DaemonSetSpec) []Diff {
	return PodTemplateSpecEqual(prefix+".template", spec.Template, actual.Template)
}

// PodTemplateSpecEqual returns zero-length result when the given objects have the same pod specs.
func PodTemplateSpecEqual(prefix string, spec, actual corev1.PodTemplateSpec) []Diff {
	return append(
		ObjectMetaEqual(prefix+".metadata", spec.ObjectMeta, actual.ObjectMeta),
		PodSpecEqual(prefix+".spec", spec.Spec, actual.Spec)...)
}

// PodSpecEqual returns zero-length result when the given objects have the same containers.
func PodSpecEqual(prefix string, spec, actual corev1.PodSpec) []Diff {
	return append(
		ContainersEqual(prefix+".initContainers", spec.InitContainers, actual.InitContainers),
		ContainersEqual(prefix+".containers", spec.Containers, actual.Containers)...)
}

// ContainersEqual returns zero-length result when the all specs containers are equal
// to the actual containers.
// More actual containers are allowed.
func ContainersEqual(prefix string, spec, actual []corev1.Container) []Diff {
	var result []Diff
	for i, sc := range spec {
		found := false
		for _, ac := range actual {
			if sc.Name == ac.Name {
				result = append(result, ContainerEqual(prefix+fmt.Sprintf(".[%d]", i), sc, ac)...)
				found = true
				break
			}
		}
		if !found {
			result = append(result, Diff(prefix+fmt.Sprintf(".[%d] not found", i)))
		}
	}
	return result
}

// ContainerEqual returns zero-length result when the given objects have the same image,
// image policy.
func ContainerEqual(prefix string, spec, actual corev1.Container) []Diff {
	return append(append(append(
		StringEqual(prefix+".image", spec.Image, actual.Image),
		StringsEqual(prefix+".command", spec.Command, actual.Command)...),
		StringsEqual(prefix+".args", spec.Args, actual.Args)...),
		EnvVarsEqual(prefix+".env", spec.Env, actual.Env)...)
}

// EnvVarsEqual returns zero-length result when the given lists are equal.
func EnvVarsEqual(prefix string, spec, actual []corev1.EnvVar) []Diff {
	var result []Diff
	for i, sv := range spec {
		found := false
		for _, av := range actual {
			if sv.Name == av.Name {
				result = append(result, EnvVarEqual(prefix+fmt.Sprintf(".[%d]", i), sv, av)...)
				found = true
			}
		}
		if !found {
			result = append(result, Diff(prefix+fmt.Sprintf(".[%d] not found", i)))
		}
	}
	if len(actual) > len(spec) {
		result = append(result, Diff(prefix+fmt.Sprintf(". %d entries added", len(actual)-len(spec))))
	}
	return result
}

// EnvVarEqual returns zero-length result when the given objects have the same key & value.
func EnvVarEqual(prefix string, spec, actual corev1.EnvVar) []Diff {
	return append(
		StringEqual(prefix+".name", spec.Name, actual.Name),
		StringEqual(prefix+".value", spec.Value, actual.Value)...)
}

// ServiceEqual returns zero-length result when the given objects have the same spec.
func ServiceEqual(spec, actual corev1.Service) []Diff {
	return append(
		ObjectMetaEqual("metadata", spec.ObjectMeta, actual.ObjectMeta),
		ServiceSpecEqual("spec", spec.Spec, actual.Spec)...)
}

// ServiceSpecEqual returns zero-length result when the given objects have the same spec.
func ServiceSpecEqual(prefix string, spec, actual corev1.ServiceSpec) []Diff {
	return append(append(append(
		StringEqual(prefix+".type", string(spec.Type), string(actual.Type)),
		BoolEqual(prefix+".publishNotReadyAddresses", spec.PublishNotReadyAddresses, actual.PublishNotReadyAddresses)...),
		LabelsEqual(prefix+".selector", spec.Selector, actual.Selector)...),
		ServicePortsEqual(prefix+".ports", spec.Ports, actual.Ports)...)
}

// ServicePortsEqual returns zero-length result when the given objects are the same.
// Adding ports is allowed.
func ServicePortsEqual(prefix string, spec, actual []corev1.ServicePort) []Diff {
	var result []Diff
	for i, sc := range spec {
		found := false
		for _, ac := range actual {
			if sc.Name == ac.Name {
				result = append(result, ServicePortEqual(prefix+fmt.Sprintf(".[%d]", i), sc, ac)...)
				found = true
				break
			}
		}
		if !found {
			result = append(result, Diff(prefix+fmt.Sprintf(".[%d] not found", i)))
		}
	}
	return result
}

// ServicePortEqual returns zero-length result when the given objects are the same.
func ServicePortEqual(prefix string, spec, actual corev1.ServicePort) []Diff {
	return append(append(append(
		StringEqual(prefix+".protocol", string(spec.Protocol), string(actual.Protocol)),
		Int32Equal(prefix+".port", spec.Port, actual.Port)...),
		StringEqual(prefix+".targetPort", spec.TargetPort.String(), actual.TargetPort.String())...),
		Int32Equal(prefix+".nodePort", spec.NodePort, actual.NodePort)...)
}
