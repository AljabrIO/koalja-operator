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
	"log"
	"net"

	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/AljabrIO/koalja-operator/pkg/constants"
	"github.com/AljabrIO/koalja-operator/pkg/event"
)

// Service implements the link agent.
type Service struct {
	port           int
	eventPublisher event.EventPublisherServer
	eventSource    event.EventSourceServer
}

// APIDependencies provides some dependencies to API builder implementations
type APIDependencies struct {
	client.Client
	Namespace string
}

// APIBuilder is an interface provided by an Link implementation
type APIBuilder interface {
	NewEventPublisher(deps APIDependencies) (event.EventPublisherServer, error)
	NewEventSource(deps APIDependencies) (event.EventSourceServer, error)
}

// NewService creates a new Service instance.
func NewService(config *rest.Config, builder APIBuilder) (*Service, error) {
	client, err := client.New(config, client.Options{})
	if err != nil {
		return nil, err
	}
	ns, err := constants.GetNamespace()
	if err != nil {
		return nil, err
	}
	port, err := constants.GetAPIPort()
	if err != nil {
		return nil, err
	}
	deps := APIDependencies{Client: client, Namespace: ns}
	eventPublisher, err := builder.NewEventPublisher(deps)
	if err != nil {
		return nil, err
	}
	eventSource, err := builder.NewEventSource(deps)
	if err != nil {
		return nil, err
	}
	return &Service{
		port:           port,
		eventPublisher: eventPublisher,
		eventSource:    eventSource,
	}, nil
}

// Run the pipeline agent until the given context is canceled.
func (s *Service) Run(ctx context.Context) error {
	addr := fmt.Sprintf("0.0.0.0:%d", s.port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	svr := grpc.NewServer()
	event.RegisterEventPublisherServer(svr, s.eventPublisher)
	event.RegisterEventSourceServer(svr, s.eventSource)
	// Register reflection service on gRPC server.
	reflection.Register(svr)
	go func() {
		if err := svr.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()
	<-ctx.Done()
	svr.GracefulStop()
	return nil
}
