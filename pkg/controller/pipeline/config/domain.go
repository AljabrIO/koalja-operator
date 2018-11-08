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
	"strings"

	yaml "gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/AljabrIO/koalja-operator/pkg/constants"
)

// NewDomain loads domain configuration for a pipeline in the given namespace.
// If no domain config is found in that namespace, the koalja-system
// namespace is used.
// +kubebuilder:rbac:groups=,resources=configmaps,verbs=get;list;watch
func NewDomain(ctx context.Context, c client.Reader, ns string) (*Domain, error) {
	var configMap corev1.ConfigMap
	key := client.ObjectKey{
		Name:      constants.ConfigMapDomain,
		Namespace: ns,
	}
	if err := c.Get(ctx, key, &configMap); errors.IsNotFound(err) {
		// Try koalja-system namespace
		key.Namespace = constants.NamespaceKoaljaSystem
		if err := c.Get(ctx, key, &configMap); errors.IsNotFound(err) {
			// No configmap found, provide a default one
			return &Domain{
				domains: map[string]labelSelector{
					"example.com": labelSelector{},
				},
			}, nil
		} else if err != nil {
			return nil, maskAny(err)
		}
	} else if err != nil {
		return nil, maskAny(err)
	}
	// Parse config map
	domain, err := newDomainFromConfigMap(&configMap)
	if err != nil {
		return nil, maskAny(err)
	}
	return domain, nil
}

// Domain name configuration
type Domain struct {
	domains map[string]labelSelector
}

// newDomainFromConfigMap creates a Domain from the supplied ConfigMap
func newDomainFromConfigMap(configMap *corev1.ConfigMap) (*Domain, error) {
	c := Domain{domains: map[string]labelSelector{}}
	hasDefault := false
	for k, v := range configMap.Data {
		labelSelector := labelSelector{}
		err := yaml.Unmarshal([]byte(v), &labelSelector)
		if err != nil {
			return nil, err
		}
		c.domains[k] = labelSelector
		if len(labelSelector.Selector) == 0 {
			hasDefault = true
		}
	}
	if !hasDefault {
		return nil, maskAny(fmt.Errorf("Config %#v must have a default domain", configMap.Data))
	}
	return &c, nil
}

// LookupDomain returns a domain given a set of labels.
func (c *Domain) LookupDomain(labels map[string]string) string {
	domain := ""
	specificity := -1

	for k, selector := range c.domains {
		// Ignore if selector doesn't match, or decrease the specificity.
		if !selector.matches(labels) || selector.specificity() < specificity {
			continue
		}
		if selector.specificity() > specificity || strings.Compare(k, domain) < 0 {
			domain = k
			specificity = selector.specificity()
		}
	}

	return domain
}

// labelSelector represents map of {key,value} pairs. A single {key,value} in the
// map is equivalent to a requirement key == value. The requirements are ANDed.
type labelSelector struct {
	Selector map[string]string `json:"selector,omitempty"`
}

// specificity returns a number indicating how specific this selector is.
// High numbers mean more specific.
func (s *labelSelector) specificity() int {
	return len(s.Selector)
}

// matches returns whether the given labels meet the requirement of the selector.
func (s *labelSelector) matches(labels map[string]string) bool {
	for label, expectedValue := range s.Selector {
		value, ok := labels[label]
		if !ok || expectedValue != value {
			return false
		}
	}
	return true
}
