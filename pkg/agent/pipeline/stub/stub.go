/*
Copyright 2018 Aljabr Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package stub

import (
	"sync"

	"github.com/AljabrIO/koalja-operator/pkg/annotatedvalue"
	avclient "github.com/AljabrIO/koalja-operator/pkg/annotatedvalue/client"
	koalja "github.com/AljabrIO/koalja-operator/pkg/apis/koalja/v1alpha1"
	"github.com/AljabrIO/koalja-operator/pkg/tracking"
	"github.com/rs/zerolog"

	"github.com/AljabrIO/koalja-operator/pkg/agent/pipeline"
)

// stub is an in-memory implementation of an annotated value queue.
type stub struct {
	agentRegistry *agentRegistry
	outputStore   *outputStore
	log           zerolog.Logger
	mutex         sync.Mutex
}

// NewStub initializes a new stub API builder
func NewStub(log zerolog.Logger) pipeline.APIBuilder {
	return &stub{
		log: log,
	}
}

// getOrCreateAgentRegistry returns the agent registry, creating one if needed.
func (s *stub) getOrCreateAgentRegistry(hub pipeline.FrontendHub) *agentRegistry {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.agentRegistry == nil {
		s.agentRegistry = newAgentRegistry(s.log, hub)
	}
	return s.agentRegistry
}

// getOrCreateOutputStore returns the output store, creating one if needed.
func (s *stub) getOrCreateOutputStore(r avclient.AnnotatedValueRegistryClient, pipeline *koalja.Pipeline, hub pipeline.FrontendHub) *outputStore {
	agentRegistry := s.getOrCreateAgentRegistry(hub)
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.outputStore == nil {
		s.outputStore = newOutputStore(s.log, r, pipeline, agentRegistry)
	}
	return s.outputStore
}

// NewAnnotatedValuePublisher "builds" a new publisher
func (s *stub) NewAnnotatedValuePublisher(deps pipeline.APIDependencies) (annotatedvalue.AnnotatedValuePublisherServer, error) {
	return s.getOrCreateOutputStore(deps.AnnotatedValueRegistry, deps.Pipeline, deps.FrontendHub), nil
}

// NewAgentRegistry creates an implementation of an AgentRegistry used to main a list of agent instances.
func (s *stub) NewAgentRegistry(deps pipeline.APIDependencies) (pipeline.AgentRegistryServer, error) {
	return s.getOrCreateAgentRegistry(deps.FrontendHub), nil
}

// NewStatisticsSink creates an implementation of an StatisticsSink.
func (s *stub) NewStatisticsSink(deps pipeline.APIDependencies) (tracking.StatisticsSinkServer, error) {
	return s.getOrCreateAgentRegistry(deps.FrontendHub), nil
}

// NewFrontend creates an implementation of an Frontend, used to query results of the pipeline.
func (s *stub) NewFrontend(deps pipeline.APIDependencies) (pipeline.FrontendServer, error) {
	return s.getOrCreateOutputStore(deps.AnnotatedValueRegistry, deps.Pipeline, deps.FrontendHub), nil
}
