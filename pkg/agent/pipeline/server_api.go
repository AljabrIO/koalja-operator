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

import (
	"context"
	fmt "fmt"
	"net"

	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/AljabrIO/koalja-operator/pkg/event"
	tracking "github.com/AljabrIO/koalja-operator/pkg/tracking"
)

// runAPIServer runs the pipeline agent GRPC server until the given context is canceled.
func (s *Service) runAPIServer(ctx context.Context, port int) error {
	// Server GRPC api
	addr := fmt.Sprintf("0.0.0.0:%d", port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		s.log.Error().Err(err).Msg("Failed to listen")
		return maskAny(err)
	}
	svr := grpc.NewServer()
	defer svr.GracefulStop()
	event.RegisterEventPublisherServer(svr, s.eventPublisher)
	RegisterAgentRegistryServer(svr, s.agentRegistry)
	tracking.RegisterStatisticsSinkServer(svr, s.statisticsSink)
	RegisterFrontendServer(svr, s.frontend)
	// Register reflection service on gRPC server.
	reflection.Register(svr)
	go func() {
		if err := svr.Serve(lis); err != nil {
			s.log.Fatal().Err(err).Msg("Failed to serve")
		}
	}()
	s.log.Info().Msgf("Started pipeline agent, listening on %s", addr)

	// Wait until context canceled
	<-ctx.Done()
	return nil
}
