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

package rest

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/vincent-petithory/dataurl"
	"k8s.io/client-go/rest"

	"github.com/AljabrIO/koalja-operator/pkg/task"
	taskclient "github.com/AljabrIO/koalja-operator/pkg/task/client"
	"github.com/AljabrIO/koalja-operator/pkg/util"
)

// Config holds the configuration arguments of the service.
type Config struct {
	// SourcePath of the file to split
	SourcePath string
	// Name of the task output we're serving
	OutputName string
}

// Service loop of the REST task.
type Service struct {
	Config
	log       zerolog.Logger
	snsClient taskclient.SnapshotServiceClient
	ornClient taskclient.OutputReadyNotifierClient
}

// NewService initializes a new service.
func NewService(cfg Config, log zerolog.Logger, config *rest.Config) (*Service, error) {
	// Check arguments
	if cfg.SourcePath == "" {
		return nil, fmt.Errorf("SourcePath expected")
	}
	if cfg.OutputName == "" {
		return nil, fmt.Errorf("OutputName expected")
	}

	// Create service clients
	snsClient, err := taskclient.CreateSnapshotServiceClient()
	if err != nil {
		log.Error().Err(err).Msg("Failed to create snapshot service client")
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
		snsClient: snsClient,
		ornClient: ornClient,
	}, nil
}

// Run th service until the given context is canceled
func (s *Service) Run(ctx context.Context) error {
	log := s.log.With().Str("path", s.Config.SourcePath).Logger()

	// Open source file
	f, err := os.Open(s.Config.SourcePath)
	if err != nil {
		log.Error().Err(err).Msg("Failed to open file")
		return maskAny(err)
	}
	defer f.Close()

	// Split into lines
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		// Get line
		line := scanner.Text()

		// Publish line
		dataURI := dataurl.New([]byte(line), "text/plain")
		if _, err := s.outputReady(ctx, dataURI.String(), s.Config.OutputName); err != nil {
			return maskAny(err)
		}
	}

	return nil
}

const (
	maxOutputReadyFailures = 10
)

// outputReady is to be called by a service runner for publishing output notifications.
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
