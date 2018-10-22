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

package task

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	koalja "github.com/AljabrIO/koalja-operator/pkg/apis/koalja/v1alpha1"
	"github.com/AljabrIO/koalja-operator/pkg/constants"
	fs "github.com/AljabrIO/koalja-operator/pkg/fs/client"
)

// Service implements the task agent.
type Service struct {
	log       zerolog.Logger
	inputLoop *inputLoop
	executor  Executor
	cache     cache.Cache
}

// NewService creates a new Service instance.
func NewService(log zerolog.Logger, config *rest.Config, scheme *runtime.Scheme) (*Service, error) {
	// Load pipeline
	pipelineName, err := constants.GetPipelineName()
	if err != nil {
		return nil, maskAny(err)
	}
	taskName, err := constants.GetTaskName()
	if err != nil {
		return nil, maskAny(err)
	}
	ns, err := constants.GetNamespace()
	if err != nil {
		return nil, maskAny(err)
	}
	podName, err := constants.GetPodName()
	if err != nil {
		return nil, maskAny(err)
	}
	c, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		return nil, maskAny(err)
	}
	cache, err := cache.New(config, cache.Options{Scheme: scheme, Namespace: ns})
	if err != nil {
		return nil, maskAny(err)
	}
	fileSystem, err := fs.CreateFileSystemClient()
	if err != nil {
		return nil, maskAny(err)
	}

	var pipeline koalja.Pipeline
	ctx := context.Background()
	if err := c.Get(ctx, client.ObjectKey{Name: pipelineName, Namespace: ns}, &pipeline); err != nil {
		return nil, maskAny(err)
	}
	taskSpec, found := pipeline.Spec.TaskByName(taskName)
	if !found {
		return nil, maskAny(fmt.Errorf("Task '%s' not found in pipeline '%s'", taskName, pipelineName))
	}

	// Load pod
	var pod core.Pod
	if err := c.Get(ctx, client.ObjectKey{Name: podName, Namespace: ns}, &pod); err != nil {
		return nil, maskAny(err)
	}

	// Create executor
	executor, err := NewExecutor(log.With().Str("component", "executor").Logger(), c, cache, fileSystem, &pipeline, &taskSpec, &pod)
	if err != nil {
		return nil, maskAny(err)
	}
	il, err := newInputLoop(log.With().Str("component", "inputLoop").Logger(), &taskSpec, &pod, executor)
	if err != nil {
		return nil, maskAny(err)
	}
	return &Service{
		inputLoop: il,
		executor:  executor,
		cache:     cache,
	}, nil
}

// Run the task agent until the given context is canceled.
func (s *Service) Run(ctx context.Context) error {
	s.log.Info().Msg("Service starting...")
	g, lctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		if err := s.cache.Start(lctx.Done()); err != nil {
			s.log.Error().Err(err).Msg("Cache failed to start")
			return maskAny(err)
		}
		return nil
	})
	g.Go(func() error {
		if err := s.inputLoop.Run(lctx); err != nil {
			s.log.Error().Err(err).Msg("InputLoop failed to start")
			return maskAny(err)
		}
		return nil
	})
	g.Go(func() error {
		if err := s.executor.Run(lctx); err != nil {
			s.log.Error().Err(err).Msg("Executor failed to start")
			return maskAny(err)
		}
		return nil
	})
	if err := g.Wait(); err != nil {
		return maskAny(err)
	}
	return nil
}
