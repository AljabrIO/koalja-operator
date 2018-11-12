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

	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/AljabrIO/koalja-operator/pkg/agent/pipeline/frontend"
	"github.com/AljabrIO/koalja-operator/pkg/annotatedvalue"
	avclient "github.com/AljabrIO/koalja-operator/pkg/annotatedvalue/client"
	koalja "github.com/AljabrIO/koalja-operator/pkg/apis/koalja/v1alpha1"
	"github.com/AljabrIO/koalja-operator/pkg/constants"
	tracking "github.com/AljabrIO/koalja-operator/pkg/tracking"
	"github.com/AljabrIO/koalja-operator/pkg/util/retry"
)

// Service implements the pipeline agent.
type Service struct {
	log zerolog.Logger
	client.Client
	Namespace      string
	avPublisher    annotatedvalue.AnnotatedValuePublisherServer
	agentRegistry  AgentRegistryServer
	statisticsSink tracking.StatisticsSinkServer
	avRegistry     avclient.AnnotatedValueRegistryClient
	frontend       FrontendServer
	frontendHub    *frontend.Hub
}

// APIDependencies provides some dependencies to API builder implementations
type APIDependencies struct {
	// Kubernetes client
	client.Client
	// Namespace in which this link is running
	Namespace string
	// AnnotatedValueRegister client
	AnnotatedValueRegistry avclient.AnnotatedValueRegistryClient
	// The pipeline
	Pipeline *koalja.Pipeline
	// Access to the frontend hub
	FrontendHub FrontendHub
}

// FrontendHub is the API provided by the frontend websocket hub.
type FrontendHub interface {
	// StatisticsChanged sends a message to all clients notifying a statistics change.
	StatisticsChanged()
}

// APIBuilder is an interface provided by an Link implementation
type APIBuilder interface {
	// NewEventPublisher creates an implementation of an AnnotatedValuePublisher used to capture output annotated values.
	NewAnnotatedValuePublisher(deps APIDependencies) (annotatedvalue.AnnotatedValuePublisherServer, error)
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

	avReg, err := avclient.NewAnnotatedValueRegistryClient()
	if err != nil {
		return nil, maskAny(err)
	}
	frontendHub := frontend.NewHub(log)
	deps := APIDependencies{
		Client:                 c,
		Namespace:              ns,
		AnnotatedValueRegistry: avReg,
		Pipeline:               &pipeline,
		FrontendHub:            frontendHub,
	}
	avPublisher, err := builder.NewAnnotatedValuePublisher(deps)
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
	frontendSvr, err := builder.NewFrontend(deps)
	if err != nil {
		return nil, maskAny(err)
	}

	return &Service{
		log:            log,
		Client:         c,
		Namespace:      ns,
		avPublisher:    avPublisher,
		avRegistry:     avReg,
		agentRegistry:  agentRegistry,
		statisticsSink: statisticsSink,
		frontend:       frontendSvr,
		frontendHub:    frontendHub,
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

	g, lctx := errgroup.WithContext(ctx)
	g.Go(func() error { return s.runAPIServer(lctx, port) })
	g.Go(func() error { return s.runFrontendServer(lctx, httpPort, port) })
	g.Go(func() error { s.frontendHub.Run(lctx); return nil })

	// Wait until done
	if err := g.Wait(); err != nil {
		return maskAny(err)
	}

	return nil
}
