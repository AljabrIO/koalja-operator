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
	"context"
	"fmt"

	"github.com/rs/zerolog"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/AljabrIO/koalja-operator/pkg/constants"
	fs "github.com/AljabrIO/koalja-operator/pkg/fs/client"
	taskclient "github.com/AljabrIO/koalja-operator/pkg/task/client"
	"github.com/AljabrIO/koalja-operator/pkg/util/retry"
)

// Config holds the configuration arguments of the service.
type Config struct {
	// Local directory path where to drop large data files
	LargeDataFolder string
	// Name of volume that contains LargeDataFolder
	VolumeName string
	// Mount path of volume that contains LargeDataFolder
	MountPath string
	// Name of node we're running on
	NodeName string
	// Name of the task output we're serving
	OutputName string
	// DatabaseConfigMap is the name of the ConfigMap containing the
	// type & access settings of the database to to run the query in
	DatabaseConfigMap string
	// Query to execute
	Query string
	// Namespace we're running in
	Namespace string
}

// Service loop of the FileDrop task.
type Service struct {
	Config
	log       zerolog.Logger
	fsClient  fs.FileSystemClient
	fsScheme  string
	ornClient taskclient.OutputReadyNotifierClient
	scheme    *runtime.Scheme
	k8sClient client.Client
}

// NewService initializes a new service.
func NewService(cfg Config, log zerolog.Logger, config *rest.Config, scheme *runtime.Scheme) (*Service, error) {
	// Check arguments
	if cfg.LargeDataFolder == "" {
		return nil, fmt.Errorf("LargeDataFolder expected")
	}
	if cfg.VolumeName == "" {
		return nil, fmt.Errorf("VolumeName expected")
	}
	if cfg.MountPath == "" {
		return nil, fmt.Errorf("MountPath expected")
	}
	if cfg.NodeName == "" {
		return nil, fmt.Errorf("NodeName expected")
	}
	if cfg.OutputName == "" {
		return nil, fmt.Errorf("OutputName expected")
	}
	if cfg.DatabaseConfigMap == "" {
		return nil, fmt.Errorf("DatabaseConfigMap expected")
	}
	if cfg.Query == "" {
		return nil, fmt.Errorf("Query expected")
	}

	// Create service clients
	if cfg.Namespace == "" {
		namespace, err := constants.GetNamespace()
		if err != nil {
			log.Error().Err(err).Msg("Failed to get namespace")
			return nil, maskAny(err)
		}
		cfg.Namespace = namespace
	}
	var k8sClient client.Client
	ctx := context.Background()
	if err := retry.Do(ctx, func(ctx context.Context) error {
		var err error
		k8sClient, err = client.New(config, client.Options{Scheme: scheme})
		return err
	}, retry.Timeout(constants.TimeoutK8sClient)); err != nil {
		return nil, err
	}
	fsClient, err := fs.NewFileSystemClient()
	if err != nil {
		log.Error().Err(err).Msg("Failed to create filesystem client")
		return nil, maskAny(err)
	}
	fsScheme, err := constants.GetFileSystemScheme()
	if err != nil {
		log.Error().Err(err).Msg("Failed to load FileSystem scheme")
		return nil, maskAny(err)
	}
	ornClient, err := taskclient.CreateOutputReadyNotifierClient()
	if err != nil {
		log.Error().Err(err).Msg("Failed to create output ready notifier client")
		return nil, maskAny(err)
	}
	return &Service{
		Config:    cfg,
		log:       log,
		fsClient:  fsClient,
		fsScheme:  fsScheme,
		ornClient: ornClient,
		k8sClient: k8sClient,
		scheme:    scheme,
	}, nil
}

// Run th service until the given context is canceled
func (s *Service) Run(ctx context.Context) error {
	// Load config map
	var configMap corev1.ConfigMap
	key := client.ObjectKey{Name: s.Config.DatabaseConfigMap, Namespace: s.Config.Namespace}
	if err := s.k8sClient.Get(ctx, key, &configMap); err != nil {
		s.log.Error().Err(err).Msg("Failed to load database config map")
		return maskAny(err)
	}
	// Parse database config
	dbCfg, err := NewDatabaseConfigFromConfigMap(s.log, &configMap)
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to parse DatabaseConfig")
		return maskAny(err)
	}

	// Find DB implementation
	db, err := GetDatabaseByType(dbCfg.Type)
	if err != nil {
		s.log.Error().Err(err).Str("type", string(dbCfg.Type)).Msg("Database type not found")
		return maskAny(err)
	}

	// Run query
	deps := QueryDependencies{
		Log:                       s.log,
		FileSystemClient:          s.fsClient,
		FileSystemScheme:          s.fsScheme,
		OutputReadyNotifierClient: s.ornClient,
		KubernetesClient:          s.k8sClient,
	}
	if err := db.Query(ctx, s.Config, *dbCfg, deps); err != nil {
		s.log.Error().Err(err).Msg("Failed to run Query")
		return maskAny(err)
	}

	return nil
}
