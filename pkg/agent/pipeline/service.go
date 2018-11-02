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
	"net/http"
	"strconv"
	"strings"

	gwruntime "github.com/grpc-ecosystem/grpc-gateway/runtime"
	assets "github.com/jessevdk/go-assets"
	"github.com/rs/zerolog"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/AljabrIO/koalja-operator/frontend"
	koalja "github.com/AljabrIO/koalja-operator/pkg/apis/koalja/v1alpha1"
	"github.com/AljabrIO/koalja-operator/pkg/constants"
	"github.com/AljabrIO/koalja-operator/pkg/event"
	"github.com/AljabrIO/koalja-operator/pkg/event/registry"
	tracking "github.com/AljabrIO/koalja-operator/pkg/tracking"
	"github.com/AljabrIO/koalja-operator/pkg/util/retry"
)

// Service implements the pipeline agent.
type Service struct {
	log zerolog.Logger
	client.Client
	Namespace      string
	eventPublisher event.EventPublisherServer
	agentRegistry  AgentRegistryServer
	statisticsSink tracking.StatisticsSinkServer
	eventRegistry  registry.EventRegistryClient
	frontend       FrontendServer
}

// APIDependencies provides some dependencies to API builder implementations
type APIDependencies struct {
	// Kubernetes client
	client.Client
	// Namespace in which this link is running
	Namespace string
	// EventRegister client
	EventRegistry registry.EventRegistryClient
	// The pipeline
	Pipeline *koalja.Pipeline
}

// APIBuilder is an interface provided by an Link implementation
type APIBuilder interface {
	// NewEventPublisher creates an implementation of an EventPublisher used to capture output events.
	NewEventPublisher(deps APIDependencies) (event.EventPublisherServer, error)
	// NewAgentRegistry creates an implementation of an AgentRegistry used to main a list of agent instances.
	NewAgentRegistry(deps APIDependencies) (AgentRegistryServer, error)
	// NewStatisticsSink creates an implementation of an StatisticsSink.
	NewStatisticsSink(deps APIDependencies) (tracking.StatisticsSinkServer, error)
	// NewFrontend creates an implementation of an FrontendServer, used to query results of the pipeline.
	NewFrontend(deps APIDependencies) (FrontendServer, error)
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
	pipelineName, err := constants.GetPipelineName()
	if err != nil {
		return nil, maskAny(err)
	}
	var pipeline koalja.Pipeline
	if err := retry.Do(ctx, func(ctx context.Context) error {
		return c.Get(ctx, client.ObjectKey{Name: pipelineName, Namespace: ns}, &pipeline)
	}, retry.Timeout(constants.TimeoutAPIServer)); err != nil {
		return nil, maskAny(err)
	}

	evtReg, err := registry.CreateEventRegistryClient()
	if err != nil {
		return nil, maskAny(err)
	}
	deps := APIDependencies{
		Client:        c,
		Namespace:     ns,
		EventRegistry: evtReg,
		Pipeline:      &pipeline,
	}
	eventPublisher, err := builder.NewEventPublisher(deps)
	if err != nil {
		return nil, maskAny(err)
	}
	agentRegistry, err := builder.NewAgentRegistry(deps)
	if err != nil {
		return nil, maskAny(err)
	}
	statisticsSink, err := builder.NewStatisticsSink(deps)
	if err != nil {
		return nil, maskAny(err)
	}
	frontend, err := builder.NewFrontend(deps)
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
		statisticsSink: statisticsSink,
		frontend:       frontend,
	}, nil
}

// Run the pipeline agent until the given context is canceled.
func (s *Service) Run(ctx context.Context) error {
	// Get config
	port, err := constants.GetAPIPort()
	if err != nil {
		return maskAny(err)
	}
	httpPort, err := constants.GetAPIHTTPPort()
	if err != nil {
		return maskAny(err)
	}

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

	// Server HTTP API
	mux := gwruntime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithInsecure()}
	if err := RegisterFrontendHandlerFromEndpoint(ctx, mux, net.JoinHostPort("localhost", strconv.Itoa(port)), opts); err != nil {
		s.log.Error().Err(err).Msg("Failed to register HTTP gateway")
		return maskAny(err)
	}
	// Frontend
	mux.Handle("GET", parsePattern("/"), createAssetFileHandler(frontend.Assets.Files["index.html"]))
	for path, file := range frontend.Assets.Files {
		localPath := "/" + strings.TrimPrefix(path, "/")
		mux.Handle("GET", parsePattern(localPath), createAssetFileHandler(file))
	}

	httpAddr := fmt.Sprintf("0.0.0.0:%d", httpPort)
	go func() {
		if err := http.ListenAndServe(httpAddr, mux); err != nil {
			s.log.Fatal().Err(err).Msg("Failed to serve HTTP gateway")
		}
	}()
	s.log.Info().Msgf("Started pipeline agent HTTP gateway, listening on %s", httpAddr)

	// Wait until context canceled
	<-ctx.Done()
	return nil
}

// createAssetFileHandler creates a gin handler to serve the content
// of the given asset file.
func createAssetFileHandler(file *assets.File) func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
	return func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
		http.ServeContent(w, r, file.Name(), file.ModTime(), file)
	}
}

func parsePattern(localPath string) gwruntime.Pattern {
	localPath = strings.TrimSuffix(strings.TrimPrefix(localPath, "/"), "/")
	if localPath == "" {
		return gwruntime.MustPattern(gwruntime.NewPattern(1, []int{}, []string{}, ""))
	}
	parts := strings.Split(localPath, "/")
	ops := make([]int, 0, len(parts)*2)
	for i := range parts {
		ops = append(ops, 2, i)
	}
	return gwruntime.MustPattern(gwruntime.NewPattern(1, ops, parts, ""))
}
