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

	koalja "github.com/AljabrIO/koalja-operator/pkg/apis/koalja/v1alpha1"
	"github.com/AljabrIO/koalja-operator/pkg/event"
	"github.com/AljabrIO/koalja-operator/pkg/event/registry"
	"github.com/rs/zerolog"

	"github.com/AljabrIO/koalja-operator/pkg/agent/pipeline"
)

// stub is an in-memory implementation of an event queue.
type stub struct {
	outputStore *outputStore
	log         zerolog.Logger
	mutex       sync.Mutex
}

// NewStub initializes a new stub API builder
func NewStub(log zerolog.Logger) pipeline.APIBuilder {
	return &stub{
		log: log,
	}
}

// getOrCreateOutputStore returns the output store, creating one if needed.
func (s *stub) getOrCreateOutputStore(r registry.EventRegistryClient, pipeline *koalja.Pipeline) *outputStore {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.outputStore == nil {
		s.outputStore = newOutputStore(s.log, r, pipeline)
	}
	return s.outputStore
}

// NewEventPublisher "builds" a new publisher
func (s *stub) NewEventPublisher(deps pipeline.APIDependencies) (event.EventPublisherServer, error) {
	return s.getOrCreateOutputStore(deps.EventRegistry, deps.Pipeline), nil
}

// NewAgentRegistry creates an implementation of an AgentRegistry used to main a list of agent instances.
func (s *stub) NewAgentRegistry(deps pipeline.APIDependencies) (pipeline.AgentRegistryServer, error) {
	return newAgentRegistry(s.log), nil
}

// NewFrontend creates an implementation of an Frontend, used to query results of the pipeline.
func (s *stub) NewFrontend(deps pipeline.APIDependencies) (pipeline.FrontendServer, error) {
	return s.getOrCreateOutputStore(deps.EventRegistry, deps.Pipeline), nil
}
