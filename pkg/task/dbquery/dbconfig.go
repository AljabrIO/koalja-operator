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

package dbquery

import (
	"fmt"

	"github.com/rs/zerolog"
	yaml "gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
)

const (
	databaseConfigFileName = "config.yaml"
)

// DatabaseConfig holds the configuration of the database to query.
type DatabaseConfig struct {
	// Type of the database
	Type DatabaseType `yaml:"type"`
	// Address used to reach the database
	Address string `yaml:"address,omitempty"`
	// Name of secret containing username+password used for authentication
	AuthenticationSecret string `yaml:"authenticationSecret,omitempty"`
	// Name of the database (in case address holds multiple databases)
	Database string `yaml:"database,omitempty"`
}

// DatabaseType indicates a type of database
type DatabaseType string

const (
	// DatabaseTypeArangoDB is the type used for ArangoDB databases.
	DatabaseTypeArangoDB DatabaseType = "ArangoDB"
)

// NewDatabaseConfigFromConfigMap loads a DatabaseConfig from the given configmap.
func NewDatabaseConfigFromConfigMap(log zerolog.Logger, configMap *corev1.ConfigMap) (*DatabaseConfig, error) {
	rawConfig, found := configMap.Data[databaseConfigFileName]
	if !found {
		log.Error().Msgf("%s missing in ConfigMap", databaseConfigFileName)
		return nil, maskAny(fmt.Errorf("No config with key '%s' found in database ConfigMap", databaseConfigFileName))
	}
	var dbCfg DatabaseConfig
	if err := yaml.Unmarshal([]byte(rawConfig), &dbCfg); err != nil {
		log.Error().Err(err).Msg("Failed to parse DatabaseConfig")
		return nil, maskAny(err)
	}
	return &dbCfg, nil
}
