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

package filedrop

import (
	"context"
	"fmt"

	"github.com/AljabrIO/koalja-operator/pkg/constants"

	"github.com/rs/zerolog"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"

	taskclient "github.com/AljabrIO/koalja-operator/pkg/task/client"
)

// Config holds the configuration arguments of the service.
type Config struct {
	// Local directory path where to drop files
	DropFolder string
	// Mount path of volume that contains DropFolder
	MountPath string
	// Name of the task output we're serving
	OutputName string
}

// Service loop of the FileDrop task.
type Service struct {
	Config
	log       zerolog.Logger
	ofsClient taskclient.OutputFileSystemServiceClient
	fsScheme  string
	ornClient taskclient.OutputReadyNotifierClient
	scheme    *runtime.Scheme
}

// NewService initializes a new service.
func NewService(cfg Config, log zerolog.Logger, config *rest.Config, scheme *runtime.Scheme) (*Service, error) {
	// Check arguments
	if cfg.DropFolder == "" {
		return nil, fmt.Errorf("DropFolder expected")
	}
	if cfg.MountPath == "" {
		return nil, fmt.Errorf("MountPath expected")
	}
	if cfg.OutputName == "" {
		return nil, fmt.Errorf("OutputName expected")
	}

	// Create service clients
	ofsClient, err := taskclient.CreateOutputFileSystemServiceClient()
	if err != nil {
		log.Error().Err(err).Msg("Failed to create output filesystem client")
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
		scheme:    scheme,
	}, nil
}

// Run th service until the given context is canceled
func (s *Service) Run(ctx context.Context) error {
	// Run webserver
	if err := s.runServer(ctx); err != nil {
		return maskAny(err)
	}
	return nil
}
