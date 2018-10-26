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
	"context"
	"sync"

	"github.com/AljabrIO/koalja-operator/pkg/event"
	"github.com/AljabrIO/koalja-operator/pkg/event/registry"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/rs/zerolog"

	"github.com/AljabrIO/koalja-operator/pkg/agent/pipeline"
)

// stub is an in-memory implementation of an event queue.
type stub struct {
	log         zerolog.Logger
	registry    registry.EventRegistryClient
	events      []event.Event
	eventsMutex sync.Mutex
	linkAgents  map[string][]string // map[link-name][]uri
	taskAgents  map[string][]string // map[task-name][]uri
	agentsMutex sync.Mutex
}

// NewStub initializes a new stub API builder
func NewStub(log zerolog.Logger) pipeline.APIBuilder {
	return &stub{
		log:        log,
		linkAgents: make(map[string][]string),
		taskAgents: make(map[string][]string),
	}
}

// NewEventPublisher "builds" a new publisher
func (s *stub) NewEventPublisher(deps pipeline.APIDependencies) (event.EventPublisherServer, error) {
	s.registry = deps.EventRegistry
	return s, nil
}

// NewAgentRegistry creates an implementation of an AgentRegistry used to main a list of agent instances.
func (s *stub) NewAgentRegistry(deps pipeline.APIDependencies) (pipeline.AgentRegistryServer, error) {
	return s, nil
}

// Publish an event
func (s *stub) Publish(ctx context.Context, req *event.PublishRequest) (*event.PublishResponse, error) {
	s.log.Debug().Interface("event", req.Event).Msg("Publish request")
	// Try to record event in registry
	e := *req.GetEvent()
	e.Link = "" // the end
	if _, err := s.registry.RecordEvent(ctx, &e); err != nil {
		return nil, maskAny(err)
	}

	// Now put event in in-memory list
	s.eventsMutex.Lock()
	defer s.eventsMutex.Unlock()
	s.events = append(s.events, e)
	return &event.PublishResponse{}, nil
}

// Register an instance of a link agent
func (s *stub) RegisterLink(ctx context.Context, req *pipeline.RegisterLinkRequest) (*empty.Empty, error) {
	s.log.Debug().
		Str("link", req.GetLinkName()).
		Str("uri", req.GetURI()).
		Msg("Register link")
	s.agentsMutex.Lock()
	defer s.agentsMutex.Unlock()

	current := s.linkAgents[req.GetLinkName()]
	if !contains(current, req.GetURI()) {
		s.linkAgents[req.GetLinkName()] = append(current, req.GetURI())
	}

	return &empty.Empty{}, nil
}

// Register an instance of a task agent
func (s *stub) RegisterTask(ctx context.Context, req *pipeline.RegisterTaskRequest) (*empty.Empty, error) {
	s.log.Debug().
		Str("task", req.GetTaskName()).
		Str("uri", req.GetURI()).
		Msg("Register task")
	s.agentsMutex.Lock()
	defer s.agentsMutex.Unlock()

	current := s.taskAgents[req.GetTaskName()]
	if !contains(current, req.GetURI()) {
		s.taskAgents[req.GetTaskName()] = append(current, req.GetURI())
	}

	return &empty.Empty{}, nil
}

func contains(list []string, value string) bool {
	for _, x := range list {
		if x == value {
			return true
		}
	}
	return false
}
