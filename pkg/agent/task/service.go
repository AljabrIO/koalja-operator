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
	"net"

	"github.com/AljabrIO/koalja-operator/pkg/util/retry"

	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/AljabrIO/koalja-operator/pkg/agent/pipeline"
	pipelinecl "github.com/AljabrIO/koalja-operator/pkg/agent/pipeline/client"
	koalja "github.com/AljabrIO/koalja-operator/pkg/apis/koalja/v1alpha1"
	"github.com/AljabrIO/koalja-operator/pkg/constants"
	fs "github.com/AljabrIO/koalja-operator/pkg/fs/client"
	ptask "github.com/AljabrIO/koalja-operator/pkg/task"
)

// Service implements the task agent.
type Service struct {
	log             zerolog.Logger
	inputLoop       *inputLoop
	executor        Executor
	outputPublisher *outputPublisher
	cache           cache.Cache
	port            int
	taskName        string
	uri             string
}

// NewService creates a new Service instance.
func NewService(log zerolog.Logger, config *rest.Config, scheme *runtime.Scheme) (*Service, error) {
	// Load pipeline
	pipelineName, err := constants.GetPipelineName()
	if err != nil {
		return nil, maskAny(err)
	}
	log = log.With().Str("pipeline", pipelineName).Logger()
	taskName, err := constants.GetTaskName()
	if err != nil {
		return nil, maskAny(err)
	}
	log = log.With().Str("task", taskName).Logger()
	ns, err := constants.GetNamespace()
	if err != nil {
		return nil, maskAny(err)
	}
	podName, err := constants.GetPodName()
	if err != nil {
		return nil, maskAny(err)
	}
	port, err := constants.GetAPIPort()
	if err != nil {
		return nil, maskAny(err)
	}
	dnsName, err := constants.GetDNSName()
	if err != nil {
		return nil, maskAny(err)
	}
	var c client.Client
	ctx := context.Background()
	if err := retry.Do(ctx, func(ctx context.Context) error {
		var err error
		c, err = client.New(config, client.Options{Scheme: scheme})
		return err
	}, retry.Timeout(constants.TimeoutK8sClient)); err != nil {
		return nil, err
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
	if err := retry.Do(ctx, func(ctx context.Context) error {
		return c.Get(ctx, client.ObjectKey{Name: pipelineName, Namespace: ns}, &pipeline)
	}, retry.Timeout(constants.TimeoutAPIServer)); err != nil {
		return nil, maskAny(err)
	}
	taskSpec, found := pipeline.Spec.TaskByName(taskName)
	if !found {
		return nil, maskAny(fmt.Errorf("Task '%s' not found in pipeline '%s'", taskName, pipelineName))
	}

	// Load pod
	var pod core.Pod
	if err := retry.Do(ctx, func(ctx context.Context) error {
		return c.Get(ctx, client.ObjectKey{Name: podName, Namespace: ns}, &pod)
	}, retry.Timeout(constants.TimeoutAPIServer)); err != nil {
		return nil, maskAny(err)
	}

	// Create executor
	uri := newTaskURI(dnsName, port, &pod)
	op, err := newOutputPublisher(log.With().Str("component", "outputPublisher").Logger(), &taskSpec, &pod)
	if err != nil {
		return nil, maskAny(err)
	}
	executor, err := NewExecutor(log.With().Str("component", "executor").Logger(), c, cache, fileSystem, &pipeline, &taskSpec, &pod, port, op)
	if err != nil {
		return nil, maskAny(err)
	}
	il, err := newInputLoop(log.With().Str("component", "inputLoop").Logger(), &taskSpec, &pod, executor)
	if err != nil {
		return nil, maskAny(err)
	}
	return &Service{
		inputLoop:       il,
		executor:        executor,
		outputPublisher: op,
		cache:           cache,
		port:            port,
		taskName:        taskName,
		uri:             uri,
	}, nil
}

// Run the task agent until the given context is canceled.
func (s *Service) Run(ctx context.Context) error {
	s.log.Info().Msg("Service starting...")

	// Register agent
	agentReg, err := pipelinecl.CreateAgentRegistryClient()
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to create agent registry client")
		return maskAny(err)
	}
	if err := retry.Do(ctx, func(ctx context.Context) error {
		if _, err := agentReg.RegisterTask(ctx, &pipeline.RegisterTaskRequest{
			TaskName: s.taskName,
			URI:      s.uri,
		}); err != nil {
			s.log.Debug().Err(err).Msg("register task agent attempt failed")
			return err
		}
		return nil
	}, retry.Timeout(constants.TimeoutRegisterAgent)); err != nil {
		s.log.Error().Err(err).Msg("Failed to register task agent")
		return maskAny(err)
	}
	s.log.Info().Msgf("Registered task %s as %s", s.taskName, s.uri)

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
	g.Go(func() error {
		if err := s.outputPublisher.Run(lctx); err != nil {
			s.log.Error().Err(err).Msg("OutputPublisher failed to start")
			return maskAny(err)
		}
		return nil
	})
	g.Go(func() error {
		if err := s.runServer(lctx); err != nil {
			s.log.Error().Err(err).Msg("Server failed to start")
			return maskAny(err)
		}
		return nil
	})
	if err := g.Wait(); err != nil {
		return maskAny(err)
	}
	return nil
}

// runServer runs the GRPC server of the task agent until the given context is canceled.
func (s *Service) runServer(ctx context.Context) error {
	// Serve API
	addr := fmt.Sprintf("0.0.0.0:%d", s.port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		s.log.Fatal().Err(err).Msg("Failed to listen")
	}
	svr := grpc.NewServer()
	ptask.RegisterOutputReadyNotifierServer(svr, s.outputPublisher)
	// Register reflection service on gRPC server.
	reflection.Register(svr)
	go func() {
		if err := svr.Serve(lis); err != nil {
			s.log.Fatal().Err(err).Msg("Failed to service")
		}
	}()
	s.log.Info().Msgf("Started output ready notifier, listening on %s", addr)
	<-ctx.Done()
	svr.GracefulStop()
	return nil
}
