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

	contour "github.com/heptio/contour/apis/contour/v1beta1"
)

// IngressRouteEqual returns zero-length result when the given objects are the same.
func IngressRouteEqual(spec, actual contour.IngressRoute) []Diff {
	return append(
		ObjectMetaEqual("metadata", spec.ObjectMeta, actual.ObjectMeta),
		IngressRouteSpecEqual("spec", spec.Spec, actual.Spec)...)
}

// IngressRouteSpecEqual returns zero-length result when the given objects are the same.
func IngressRouteSpecEqual(prefix string, spec, actual contour.IngressRouteSpec) []Diff {
	return append(
		VirtualHostEqual(prefix+".virtualhost", spec.VirtualHost, actual.VirtualHost),
		ContourRoutesEqual(prefix+".routes", spec.Routes, actual.Routes)...)
}

// VirtualHostEqual returns zero-length result when the given objects are the same.
func VirtualHostEqual(prefix string, spec, actual *contour.VirtualHost) []Diff {
	if spec == nil && actual == nil {
		return nil
	}
	if spec == nil && actual != nil {
		return []Diff{Diff(fmt.Sprintf("%s not wanted", prefix))}
	} else if spec != nil && actual == nil {
		return []Diff{Diff(fmt.Sprintf("%s missing", prefix))}
	}
	return append(
		StringEqual(prefix+".fqdn", spec.Fqdn, actual.Fqdn),
		TLSEqual(prefix+".tls", spec.TLS, actual.TLS)...)
}

// TLSEqual returns zero-length result when the given objects are the same.
func TLSEqual(prefix string, spec, actual *contour.TLS) []Diff {
	if spec == nil && actual == nil {
		return nil
	}
	if spec == nil && actual != nil {
		return []Diff{Diff(fmt.Sprintf("%s not wanted", prefix))}
	} else if spec != nil && actual == nil {
		return []Diff{Diff(fmt.Sprintf("%s missing", prefix))}
	}
	return append(
		StringEqual(prefix+".secretName", spec.SecretName, actual.SecretName),
		StringEqual(prefix+".minimumProtocolVersion", spec.MinimumProtocolVersion, actual.MinimumProtocolVersion)...)
}

// ContourRoutesEqual returns zero-length result when the given objects are the same.
func ContourRoutesEqual(prefix string, spec, actual []contour.Route) []Diff {
	var result []Diff
	for i, sr := range spec {
		if i < len(actual) {
			if d := ContourRouteEqual(fmt.Sprintf("%s[%d]", prefix, i), sr, actual[i]); len(d) > 0 {
				result = append(result, d...)
			}
		} else {
			result = append(result, Diff(fmt.Sprintf("%s[%d] missing", prefix, i)))
		}
	}
	if len(actual) > len(spec) {
		result = append(result, Diff(fmt.Sprintf("additional routes found in %s", prefix)))
	}
	return result
}

// ContourRouteEqual returns zero-length result when the given objects are the same.
func ContourRouteEqual(prefix string, spec, actual contour.Route) []Diff {
	return append(append(append(append(
		StringEqual(prefix+".match", spec.Match, actual.Match),
		BoolEqual(prefix+".enableWebsockets", spec.EnableWebsockets, actual.EnableWebsockets)...),
		BoolEqual(prefix+".permitInsecure", spec.PermitInsecure, actual.PermitInsecure)...),
		StringEqual(prefix+".prefixRewrite", spec.PrefixRewrite, actual.PrefixRewrite)...),
		ContourServicesEqual(prefix+".services", spec.Services, actual.Services)...)
}

// ContourServicesEqual returns zero-length result when the given objects are the same.
func ContourServicesEqual(prefix string, spec, actual []contour.Service) []Diff {
	var result []Diff
	for i, sr := range spec {
		if i < len(actual) {
			if d := ContourServiceEqual(fmt.Sprintf("%s[%d]", prefix, i), sr, actual[i]); len(d) > 0 {
				result = append(result, d...)
			}
		} else {
			result = append(result, Diff(fmt.Sprintf("%s[%d] missing", prefix, i)))
		}
	}
	if len(actual) > len(spec) {
		result = append(result, Diff(fmt.Sprintf("additional services found in %s", prefix)))
	}
	return result
}

// ContourServiceEqual returns zero-length result when the given objects are the same.
func ContourServiceEqual(prefix string, spec, actual contour.Service) []Diff {
	return append(append(append(
		StringEqual(prefix+".name", spec.Name, actual.Name),
		IntEqual(prefix+".port", spec.Port, actual.Port)...),
		IntEqual(prefix+".weight", spec.Weight, actual.Weight)...),
		StringEqual(prefix+".strategy", spec.Strategy, actual.Strategy)...)
}
