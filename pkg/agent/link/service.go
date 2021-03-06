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

package link

//go:generate protoc -I .:../../../vendor:../../.. --go_out=plugins=grpc:. agent_api.proto

import (
	"context"
	fmt "fmt"
	"net"
	"time"

	"golang.org/x/sync/errgroup"

	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/AljabrIO/koalja-operator/pkg/agent/pipeline"
	pipelinecl "github.com/AljabrIO/koalja-operator/pkg/agent/pipeline/client"
	"github.com/AljabrIO/koalja-operator/pkg/annotatedvalue"
	avclient "github.com/AljabrIO/koalja-operator/pkg/annotatedvalue/client"
	"github.com/AljabrIO/koalja-operator/pkg/constants"
	"github.com/AljabrIO/koalja-operator/pkg/tracking"
	trackingcl "github.com/AljabrIO/koalja-operator/pkg/tracking/client"
	"github.com/AljabrIO/koalja-operator/pkg/util"
	"github.com/AljabrIO/koalja-operator/pkg/util/retry"
	"github.com/rs/zerolog"
)

// Service implements the link agent.
type Service struct {
	log            zerolog.Logger
	port           int
	linkName       string
	pipelineName   string
	uri            string
	statistics     *tracking.LinkStatistics
	avPublisher    AnnotatedValuePublisherServer
	avSource       AnnotatedValueSourceServer
	avRegistry     avclient.AnnotatedValueRegistryClient
	agentRegistry  pipeline.AgentRegistryClient
	statisticsSink tracking.StatisticsSinkClient
}

// APIDependencies provides some dependencies to API builder implementations
type APIDependencies struct {
	// Kubernetes client
	client.Client
	// Name of the pipeline
	PipelineName string
	// Name of the link
	LinkName string
	// Namespace in which this link is running
	Namespace string
	// URI of this link
	URI string
	// EventRegister client
	AnnotatedValueRegistry avclient.AnnotatedValueRegistryClient
	// AgentRegistry client
	AgentRegistry pipeline.AgentRegistryClient
	// Statistics
	Statistics *tracking.LinkStatistics
}

// AnnotatedValuePublisherServer extends the annotated-value AnnotatedValuePublisherServer
// with internal Run method.
type AnnotatedValuePublisherServer interface {
	annotatedvalue.AnnotatedValuePublisherServer
	// Run this server until the given context is canceled.
	Run(ctx context.Context) error
}

// AnnotatedValueSourceServer extends the annotated-value AnnotatedValueSourceServer
// with internal Run method.
type AnnotatedValueSourceServer interface {
	annotatedvalue.AnnotatedValueSourceServer
	// Run this server until the given context is canceled.
	Run(ctx context.Context) error
}

// APIBuilder is an interface provided by an Link implementation
type APIBuilder interface {
	NewAnnotatedValuePublisher(deps APIDependencies) (AnnotatedValuePublisherServer, error)
	NewAnnotatedValueSource(deps APIDependencies) (AnnotatedValueSourceServer, error)
}

// NewService creates a new Service instance.
func NewService(log zerolog.Logger, config *rest.Config, builder APIBuilder) (*Service, error) {
	var c client.Client
	ctx := context.Background()
	if err := retry.Do(ctx, func(ctx context.Context) error {
		var err error
		c, err = client.New(config, client.Options{})
		return err
	}, retry.Timeout(constants.TimeoutK8sClient)); err != nil {
		return nil, err
	}
	ns, err := constants.GetNamespace()
	if err != nil {
		return nil, maskAny(err)
	}
	podName, err := constants.GetPodName()
	if err != nil {
		return nil, maskAny(err)
	}
	pipelineName, err := constants.GetPipelineName()
	if err != nil {
		return nil, maskAny(err)
	}
	linkName, err := constants.GetLinkName()
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
	var p corev1.Pod
	podKey := client.ObjectKey{
		Name:      podName,
		Namespace: ns,
	}
	if err := retry.Do(ctx, func(ctx context.Context) error {
		return c.Get(ctx, podKey, &p)
	}, retry.Timeout(constants.TimeoutAPIServer)); err != nil {
		log.Error().Err(err).Msg("Failed to get own pod")
		return nil, maskAny(err)
	}
	avReg, err := avclient.NewAnnotatedValueRegistryClient()
	if err != nil {
		log.Error().Err(err).Msg("Failed to create annotated value registry client")
		return nil, maskAny(err)
	}
	agentReg, err := pipelinecl.CreateAgentRegistryClient()
	if err != nil {
		log.Error().Err(err).Msg("Failed to create agent registry client")
		return nil, maskAny(err)
	}
	statsSink, err := trackingcl.CreateStatisticsSinkClient()
	if err != nil {
		log.Error().Err(err).Msg("Failed to create statistics sink client")
		return nil, maskAny(err)
	}
	uri := newLinkURI(dnsName, port, &p)
	statistics := &tracking.LinkStatistics{
		Name: linkName,
		URI:  uri,
	}
	deps := APIDependencies{
		Client:                 c,
		PipelineName:           pipelineName,
		LinkName:               linkName,
		Namespace:              ns,
		URI:                    uri,
		AnnotatedValueRegistry: avReg,
		AgentRegistry:          agentReg,
		Statistics:             statistics,
	}
	avPublisher, err := builder.NewAnnotatedValuePublisher(deps)
	if err != nil {
		return nil, maskAny(err)
	}
	avSource, err := builder.NewAnnotatedValueSource(deps)
	if err != nil {
		return nil, maskAny(err)
	}
	return &Service{
		log:            log,
		port:           port,
		linkName:       linkName,
		uri:            uri,
		statistics:     statistics,
		avPublisher:    avPublisher,
		avSource:       avSource,
		avRegistry:     avReg,
		agentRegistry:  agentReg,
		statisticsSink: statsSink,
	}, nil
}

// Run the link agent until the given context is canceled.
func (s *Service) Run(ctx context.Context) error {
	defer s.avRegistry.Close()

	// Start the registry update loop
	agentRegistered := make(chan struct{})
	go s.runUpdateAgentRegistration(ctx, agentRegistered)

	// Wait until agent is first registered
	select {
	case <-agentRegistered:
		// Agent was register. Good to start
	case <-ctx.Done():
		// Context was canceled
		return maskAny(ctx.Err())
	}

	// Serve API
	addr := fmt.Sprintf("0.0.0.0:%d", s.port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to listen")
		return maskAny(err)
	}
	svr := grpc.NewServer()
	defer svr.GracefulStop()
	annotatedvalue.RegisterAnnotatedValuePublisherServer(svr, s.avPublisher)
	annotatedvalue.RegisterAnnotatedValueSourceServer(svr, s.avSource)
	// Register reflection service on gRPC server.
	reflection.Register(svr)
	g, lctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		if err := svr.Serve(lis); err != nil {
			s.log.Fatal().Err(err).Msg("Failed to service")
		}
		return nil
	})
	g.Go(func() error {
		if err := s.avPublisher.Run(lctx); err != nil {
			return maskAny(err)
		}
		return nil
	})
	g.Go(func() error {
		if err := s.avSource.Run(lctx); err != nil {
			return maskAny(err)
		}
		return nil
	})
	s.log.Info().Msgf("Started link %s, listening on %s", s.linkName, addr)

	// Wait until context closed
	if err := g.Wait(); err != nil {
		return maskAny(err)
	}
	return nil
}

// runUpdateAgentRegistration registers the link at the agent registry at regular
// intervals and publishes the statistics frequently.
func (s *Service) runUpdateAgentRegistration(ctx context.Context, agentRegistered chan struct{}) {
	log := s.log.With().
		Str("link", s.linkName).
		Str("uri", s.uri).
		Logger()
	minDelay := time.Second
	maxDelay := time.Second * 15
	delay := minDelay
	var lastAgentRegistration time.Time
	var lastHash string
	for {
		// Register agent (if needed)
		registered := false
		if time.Since(lastAgentRegistration) > time.Minute {
			if err := retry.Do(ctx, func(ctx context.Context) error {
				if _, err := s.agentRegistry.RegisterLink(ctx, &pipeline.RegisterLinkRequest{
					LinkName: s.linkName,
					URI:      s.uri,
				}); err != nil {
					log.Debug().Err(err).Msg("register link agent attempt failed")
					return err
				}
				return nil
			}, retry.Timeout(constants.TimeoutRegisterAgent)); err != nil {
				log.Error().Err(err).Msg("Failed to register link agent")
			} else {
				// Success
				lastAgentRegistration = time.Now()
				registered = true
				if agentRegistered != nil {
					close(agentRegistered)
					agentRegistered = nil
					log.Info().Msg("Registered link")
				} else {
					log.Debug().Msg("Re-registered link")
				}
			}
		}

		if agentRegistered == nil {
			if err := retry.Do(ctx, func(ctx context.Context) error {
				stats := *s.statistics
				if newHash := util.Hash(stats); newHash != lastHash || registered {
					if _, err := s.statisticsSink.PublishLinkStatistics(ctx, &stats); err != nil {
						s.log.Debug().Err(err).Msg("publish link statistics attempt failed")
						return err
					}
					lastHash = newHash
				}
				return nil
			}, retry.Timeout(constants.TimeoutRegisterAgent)); err != nil {
				s.log.Error().Err(err).Msg("Failed to public link statistics")
				delay = time.Duration(float64(delay) * 1.5)
				if delay > maxDelay {
					delay = maxDelay
				}
			} else {
				// Success
				delay = minDelay
			}
		}

		select {
		case <-time.After(delay):
			// Continue
		case <-ctx.Done():
			// Context canceled
			return
		}
	}
}
