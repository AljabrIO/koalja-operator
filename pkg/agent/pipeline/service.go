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

package pipeline

//go:generate protoc -I .:../../../vendor --go_out=plugins=grpc:. agent_api.proto

import (
	"context"
	fmt "fmt"
	"net"

	"github.com/AljabrIO/koalja-operator/pkg/constants"
	"github.com/AljabrIO/koalja-operator/pkg/event"
	"github.com/AljabrIO/koalja-operator/pkg/event/registry"
	"github.com/AljabrIO/koalja-operator/pkg/util/retry"
	"github.com/rs/zerolog"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Service implements the pipeline agent.
type Service struct {
	log zerolog.Logger
	client.Client
	Namespace      string
	eventPublisher event.EventPublisherServer
	agentRegistry  AgentRegistryServer
	eventRegistry  registry.EventRegistryClient
}

// APIDependencies provides some dependencies to API builder implementations
type APIDependencies struct {
	// Kubernetes client
	client.Client
	// Namespace in which this link is running
	Namespace string
	// EventRegister client
	EventRegistry registry.EventRegistryClient
}

// APIBuilder is an interface provided by an Link implementation
type APIBuilder interface {
	// NewEventPublisher creates an implementation of an EventPublisher used to capture output events.
	NewEventPublisher(deps APIDependencies) (event.EventPublisherServer, error)
	// NewAgentRegistry creates an implementation of an AgentRegistry used to main a list of agent instances.
	NewAgentRegistry(deps APIDependencies) (AgentRegistryServer, error)
}

// NewService creates a new Service instance.
func NewService(log zerolog.Logger, config *rest.Config, scheme *runtime.Scheme, builder APIBuilder) (*Service, error) {
	var c client.Client
	ctx := context.Background()
	if err := retry.Do(ctx, func(ctx context.Context) error {
		var err error
		c, err = client.New(config, client.Options{Scheme: scheme})
		return err
	}, retry.Timeout(constants.TimeoutK8sClient)); err != nil {
		return nil, err
	}
	ns, err := constants.GetNamespace()
	if err != nil {
		return nil, err
	}
	evtReg, err := registry.CreateEventRegistryClient()
	if err != nil {
		return nil, maskAny(err)
	}
	deps := APIDependencies{
		Client:        c,
		Namespace:     ns,
		EventRegistry: evtReg,
	}
	eventPublisher, err := builder.NewEventPublisher(deps)
	if err != nil {
		return nil, maskAny(err)
	}
	agentRegistry, err := builder.NewAgentRegistry(deps)
	if err != nil {
		return nil, maskAny(err)
	}

	return &Service{
		log:            log,
		Client:         c,
		Namespace:      ns,
		eventPublisher: eventPublisher,
		eventRegistry:  evtReg,
		agentRegistry:  agentRegistry,
	}, nil
}

// Run the pipeline agent until the given context is canceled.
func (s *Service) Run(ctx context.Context) error {
	port, err := constants.GetAPIPort()
	if err != nil {
		return err
	}
	addr := fmt.Sprintf("0.0.0.0:%d", port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		s.log.Fatal().Err(err).Msg("Failed to listen")
	}
	svr := grpc.NewServer()
	event.RegisterEventPublisherServer(svr, s.eventPublisher)
	RegisterAgentRegistryServer(svr, s.agentRegistry)
	// Register reflection service on gRPC server.
	reflection.Register(svr)
	go func() {
		if err := svr.Serve(lis); err != nil {
			s.log.Fatal().Err(err).Msg("Failed to serve")
		}
	}()
	s.log.Info().Msgf("Started pipeline agent, listening on %s", addr)
	<-ctx.Done()
	svr.GracefulStop()
	return nil
}
