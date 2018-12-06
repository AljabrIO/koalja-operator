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
	"time"

	"github.com/rs/zerolog"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/AljabrIO/koalja-operator/pkg/constants"
	"github.com/AljabrIO/koalja-operator/pkg/task"
	taskclient "github.com/AljabrIO/koalja-operator/pkg/task/client"
	"github.com/AljabrIO/koalja-operator/pkg/util"
	"github.com/AljabrIO/koalja-operator/pkg/util/retry"
)

// Config holds the configuration arguments of the service.
type Config struct {
	// Local directory path where to drop large data files
	LargeDataFolder string
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
	ofsClient taskclient.OutputFileSystemServiceClient
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
	ofsClient, err := taskclient.CreateOutputFileSystemServiceClient()
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
		ofsClient: ofsClient,
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

	// Fetch authentication secret (if needed)
	var authSecret *corev1.Secret
	if dbCfg.AuthenticationSecret != "" {
		authSecret = &corev1.Secret{}
		key := client.ObjectKey{Name: dbCfg.AuthenticationSecret, Namespace: s.Namespace}
		if err := s.k8sClient.Get(ctx, key, authSecret); err != nil {
			s.log.Error().Err(err).Str("secret", dbCfg.AuthenticationSecret).Msg("Failed to fetch authentication secret")
			return maskAny(err)
		}
	}

	// Run query
	deps := QueryDependencies{
		Log:                  s.log,
		FileSystemClient:     s.ofsClient,
		FileSystemScheme:     s.fsScheme,
		OutputReady:          s.outputReady,
		KubernetesClient:     s.k8sClient,
		AuthenticationSecret: authSecret,
	}
	if err := db.Query(ctx, s.Config, *dbCfg, deps); err != nil {
		s.log.Error().Err(err).Msg("Failed to run Query")
		return maskAny(err)
	}

	return nil
}

const (
	maxOutputReadyFailures = 10
)

// outputReady is to be called by a database for publishing output notifications.
// Returns: (annotatedValueID, error)
func (s *Service) outputReady(ctx context.Context, annotatedValueData, outputName string) (string, error) {
	delay := time.Millisecond * 100
	recentFailures := 0
	for {
		if resp, err := s.ornClient.OutputReady(ctx, &task.OutputReadyRequest{
			AnnotatedValueData: annotatedValueData,
			OutputName:         outputName,
		}); err != nil {
			recentFailures++
			if recentFailures > maxOutputReadyFailures {
				s.log.Error().Err(err).Msg("OutputReady failed too many times")
				return "", maskAny(err)
			}
			s.log.Debug().Err(err).Msg("OutputReady attempt failed")
		} else if resp.Accepted {
			// Output was accepted
			return resp.GetAnnotatedValueID(), nil
		} else {
			// OutputReady call succeeded, but output was not (yet) accepted
			s.log.Debug().Err(err).Msg("OutputReady did not accept our value. Wait and try again...")
			recentFailures = 0
		}
		// Output was not accepted, or call failed, try again soon.
		select {
		case <-time.After(delay):
			// Try again
			delay = util.Backoff(delay, 1.5, time.Minute)
		case <-ctx.Done():
			// Context canceled
			s.log.Debug().Err(ctx.Err()).Msg("Context canceled during outputReady")
			return "", ctx.Err()
		}
	}
}
