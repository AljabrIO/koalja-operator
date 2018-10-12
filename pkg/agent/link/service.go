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

//go:generate protoc -I .:../../../vendor --go_out=plugins=grpc:. agent_api.proto

import (
	"context"
	fmt "fmt"
	"log"
	"net"

	"github.com/AljabrIO/koalja-operator/pkg/agent"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"k8s.io/client-go/rest"
)

// Service implements the link agent.
type Service struct {
}

// NewService creates a new Service instance.
func NewService(config *rest.Config) (*Service, error) {
	return &Service{}, nil
}

// Run the pipeline agent until the given context is canceled.
func (s *Service) Run(ctx context.Context) error {
	port, err := agent.GetAgentPort()
	if err != nil {
		return err
	}
	lis, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	svr := grpc.NewServer()
	RegisterAgentServer(svr, s)
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

// CreateSourceSidecar returns a filled out Container that is injected
// into a Task Execution Pod to serve as link endpoint for the source
// of this link.
func (s *Service) CreateSourceSidecar(context.Context, *SidecarRequest) (*SidecarResponse, error) {
	return &SidecarResponse{}, nil
}

// CreateDestinationSidecar returns a filled out Container that is injected
// into a Task Execution Pod to serve as link endpoint for the destination
// of this link.
func (s *Service) CreateDestinationSidecar(context.Context, *SidecarRequest) (*SidecarResponse, error) {
	return &SidecarResponse{}, nil
}
