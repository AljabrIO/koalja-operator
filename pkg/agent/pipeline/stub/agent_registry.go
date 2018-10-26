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

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/rs/zerolog"

	"github.com/AljabrIO/koalja-operator/pkg/agent/pipeline"
)

// agentRegistry is an in-memory implementation of an agent registry.
type agentRegistry struct {
	log         zerolog.Logger
	linkAgents  map[string][]string // map[link-name][]uri
	taskAgents  map[string][]string // map[task-name][]uri
	agentsMutex sync.Mutex
}

// newAgentRegistry initializes a new agent registry
func newAgentRegistry(log zerolog.Logger) pipeline.AgentRegistryServer {
	return &agentRegistry{
		log:        log,
		linkAgents: make(map[string][]string),
		taskAgents: make(map[string][]string),
	}
}

// Register an instance of a link agent
func (s *agentRegistry) RegisterLink(ctx context.Context, req *pipeline.RegisterLinkRequest) (*empty.Empty, error) {
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
func (s *agentRegistry) RegisterTask(ctx context.Context, req *pipeline.RegisterTaskRequest) (*empty.Empty, error) {
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
