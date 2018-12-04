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

package s3

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"net/url"
	"reflect"
	"sort"
	"strings"

	yaml "gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/AljabrIO/koalja-operator/pkg/constants"
)

// NewStorageConfig loads storage configuration for a S3 FileSystem service in the given namespace.
// If no storage config is found in that namespace, the koalja-system
// namespace is used.
// +kubebuilder:rbac:groups=,resources=configmaps,verbs=get;list;watch
func NewStorageConfig(ctx context.Context, c client.Reader, ns string) (*StorageConfig, error) {
	var configMap corev1.ConfigMap
	key := client.ObjectKey{
		Name:      constants.ConfigMapS3Storage,
		Namespace: ns,
	}
	if err := c.Get(ctx, key, &configMap); errors.IsNotFound(err) {
		// Try koalja-system namespace
		key.Namespace = constants.NamespaceKoaljaSystem
		if err := c.Get(ctx, key, &configMap); err != nil {
			return nil, maskAny(err)
		}
	} else if err != nil {
		return nil, maskAny(err)
	}
	// Parse config map
	sc, err := newStorageConfigFromConfigMap(ctx, &configMap, c, ns)
	if err != nil {
		return nil, maskAny(err)
	}
	return sc, nil
}

// StorageConfig contains all buckets available to the S3 filesystem.
type StorageConfig struct {
	Buckets []BucketConfig
}

// Equals returns true if the given StorageConfig's are the same.
func (sc StorageConfig) Equals(other StorageConfig) bool {
	return reflect.DeepEqual(sc, other)
}

// BucketConfig contains all information needed to reach a single S3 bucket.
type BucketConfig struct {
	// Name of the bucket
	Name string `yaml:"name"`
	// Endpoint of the server used to reach this bucket
	Endpoint string `yaml:"endpoint,omitempty"`
	// Secure is set to true for using a secure connection, false for plain text connection.
	Secure bool `yaml:"secure,omitempty"`
	// Name of the secret containing the access & secret key for the bucket
	SecretName string `yaml:"secretName"`
	// isDefault is set to true for the bucket that is listed with the "default" key
	// in the ConfigMap.
	isDefault bool
	// accessKey of the server. Loaded from Secret
	accessKey string
	// secretKey of the server. Loaded from Secret
	secretKey string
}

// newStorageConfigFromConfigMap creates a StorageConfig from the supplied ConfigMap
func newStorageConfigFromConfigMap(ctx context.Context, configMap *corev1.ConfigMap, c client.Reader, ns string) (*StorageConfig, error) {
	var sc StorageConfig
	hasDefault := false
	for k, v := range configMap.Data {
		var bc BucketConfig
		err := yaml.Unmarshal([]byte(v), &bc)
		if err != nil {
			return nil, err
		}
		bc.fixEndpoint()
		bc.isDefault = k == "default"

		// Try loading the secret
		var secret corev1.Secret
		key := client.ObjectKey{
			Name:      bc.SecretName,
			Namespace: configMap.GetNamespace(),
		}
		if err := c.Get(ctx, key, &secret); err != nil {
			return nil, maskAny(err)
		}
		if raw, found := secret.Data[constants.SecretKeyS3AccessKey]; found {
			bc.accessKey = string(raw)
		} else {
			return nil, maskAny(fmt.Errorf("Config %#v refers to Secret '%s' that has no '%s' field", configMap.Data, bc.SecretName, constants.SecretKeyS3AccessKey))
		}
		if raw, found := secret.Data[constants.SecretKeyS3AccessKey]; found {
			bc.secretKey = string(raw)
		} else {
			return nil, maskAny(fmt.Errorf("Config %#v refers to Secret '%s' that has no '%s' field", configMap.Data, bc.SecretName, constants.SecretKeyS3SecretKey))
		}

		// Add to config
		hasDefault = hasDefault || bc.isDefault
		sc.Buckets = append(sc.Buckets, bc)
	}
	if !hasDefault {
		return nil, maskAny(fmt.Errorf("Config %#v must have a default bucket", configMap.Data))
	}
	sort.Slice(sc.Buckets, func(i, j int) bool {
		a, b := sc.Buckets[i], sc.Buckets[j]
		if a.isDefault && !b.isDefault {
			return true
		}
		if !a.isDefault && b.isDefault {
			return false
		}
		return strings.Compare(a.Name, b.Name) < 0
	})
	return &sc, nil
}

// fixEndpoint removes the scheme from the endpoint
func (bc *BucketConfig) fixEndpoint() {
	if u, err := url.Parse(bc.Endpoint); err == nil {
		bc.Endpoint = u.Host
		if strings.ToLower(u.Scheme) == "https" {
			bc.Secure = true
		}
	}
}

// hash returns a lower-case hash that uniquely identifies the bucket name & endpoint.
func (bc BucketConfig) hash() string {
	source := strings.ToLower(bc.Name + "/" + bc.Endpoint)
	hash := sha1.Sum([]byte(source))
	return strings.ToLower(hex.EncodeToString(hash[0:6]))
}

// PersistentVolumeName returns the name of the PV for this bucket.
func (bc BucketConfig) PersistentVolumeName(prefix string) string {
	return prefix + "-" + bc.hash()
}

// Matches returns true if this bucket config equals the given endpoint & name.
func (bc BucketConfig) Matches(endpoint, bucketName string) bool {
	return strings.ToLower(endpoint) == strings.ToLower(bc.Endpoint) &&
		strings.ToLower(bucketName) == strings.ToLower(bc.Name)
}

// EndpointAsURL returns the endpoint of the bucket including scheme.
func (bc BucketConfig) EndpointAsURL() string {
	prefix := "http://"
	if bc.Secure {
		prefix = "https://"
	}
	return prefix + bc.Endpoint
}
