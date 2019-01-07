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

package config

import (
	"context"
	"fmt"

	yaml "gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/AljabrIO/koalja-operator/pkg/constants"
)

// NewServices loads services configuration for a pipeline in the given namespace.
// If no services config is found in that namespace, the koalja-system
// namespace is used.
// +kubebuilder:rbac:groups=,resources=configmaps,verbs=get;list;watch
func NewServices(ctx context.Context, c client.Reader, ns string) (*Services, error) {
	var configMap corev1.ConfigMap
	key := client.ObjectKey{
		Name:      constants.ConfigMapServices,
		Namespace: ns,
	}
	if err := c.Get(ctx, key, &configMap); errors.IsNotFound(err) {
		// Try koalja-system namespace
		key.Namespace = constants.NamespaceKoaljaSystem
		if err := c.Get(ctx, key, &configMap); errors.IsNotFound(err) {
			// No configmap found, provide a default one
			empty := &Services{}
			empty.setDefaults()
			return empty, nil
		} else if err != nil {
			return nil, maskAny(err)
		}
	} else if err != nil {
		return nil, maskAny(err)
	}
	// Parse config map
	result, err := newServicesFromConfigMap(&configMap)
	if err != nil {
		return nil, maskAny(err)
	}
	result.setDefaults()
	return result, nil
}

// Services preferences configuration
type Services struct {
	// AnnotatedValueRegistryName is the name of the AnnotatedValueRegistry resources used for pipelines.
	AnnotatedValueRegistryName string `json:"annotated-value-registry-name,omitempty" yaml:"annotated-value-registry-name,omitempty"`
}

// setDefaults fills empty values with default values
func (s *Services) setDefaults() {
	if s.AnnotatedValueRegistryName == "" {
		s.AnnotatedValueRegistryName = "stub-annotatedvalue-registry"
	}
}

// newServicesFromConfigMap creates a Services from the supplied ConfigMap
func newServicesFromConfigMap(configMap *corev1.ConfigMap) (*Services, error) {
	data, found := configMap.Data[constants.ConfigMapKeyServicesConfig]
	if !found {
		return nil, maskAny(fmt.Errorf("Config %#v must have a %s key", configMap.Data, constants.ConfigMapKeyServicesConfig))
	}
	var result Services
	err := yaml.Unmarshal([]byte(data), &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
