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
	"sort"
	"sync"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/rs/zerolog"

	"github.com/AljabrIO/koalja-operator/pkg/agent/pipeline"
	"github.com/AljabrIO/koalja-operator/pkg/tracking"
)

// agentRegistry is an in-memory implementation of an agent registry.
type agentRegistry struct {
	log         zerolog.Logger
	linkAgents  map[string][]*linkAgent // map[link-name][]linkAgent
	taskAgents  map[string][]*taskAgent // map[task-name][]taskAgent
	agentsMutex sync.Mutex
	hub         pipeline.FrontendHub
}

type linkAgent struct {
	URI        string
	Statistics struct {
		Timestamp time.Time
		Data      tracking.LinkStatistics
	}
}

type taskAgent struct {
	URI        string
	Statistics struct {
		Timestamp time.Time
		Data      tracking.TaskStatistics
	}
}

// newAgentRegistry initializes a new agent registry
func newAgentRegistry(log zerolog.Logger, hub pipeline.FrontendHub) *agentRegistry {
	return &agentRegistry{
		log:        log,
		hub:        hub,
		linkAgents: make(map[string][]*linkAgent),
		taskAgents: make(map[string][]*taskAgent),
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
	uriFound := false
	for _, x := range current {
		if x.URI == req.GetURI() {
			uriFound = true
			break
		}
	}
	if !uriFound {
		s.linkAgents[req.GetLinkName()] = append(current, &linkAgent{URI: req.GetURI()})
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
	uriFound := false
	for _, x := range current {
		if x.URI == req.GetURI() {
			uriFound = true
			break
		}
	}
	if !uriFound {
		s.taskAgents[req.GetTaskName()] = append(current, &taskAgent{URI: req.GetURI()})
	}

	return &empty.Empty{}, nil
}

// Provide statistics of a link (called by the link)
func (s *agentRegistry) PublishLinkStatistics(ctx context.Context, req *tracking.LinkStatistics) (*empty.Empty, error) {
	s.log.Debug().
		Str("link", req.GetName()).
		Str("uri", req.GetURI()).
		Msg("Publish link statistics")

	s.agentsMutex.Lock()
	{
		current := s.linkAgents[req.GetName()]
		var linkAgentRef *linkAgent
		for _, x := range current {
			if x.URI == req.GetURI() {
				linkAgentRef = x
				break
			}
		}
		if linkAgentRef == nil {
			linkAgentRef = &linkAgent{URI: req.GetURI()}
			current = append(current, linkAgentRef)
			s.linkAgents[req.GetName()] = current
		}
		linkAgentRef.Statistics.Data.Reset()
		linkAgentRef.Statistics.Data.Add(*req)
		linkAgentRef.Statistics.Timestamp = time.Now()
		sort.Slice(current, func(i, j int) bool { return current[i].Statistics.Timestamp.Before(current[j].Statistics.Timestamp) })
	}
	s.agentsMutex.Unlock()

	// Notify frontend clients
	s.hub.StatisticsChanged()

	return &empty.Empty{}, nil
}

// Provide statistics of a task (called by the task)
func (s *agentRegistry) PublishTaskStatistics(ctx context.Context, req *tracking.TaskStatistics) (*empty.Empty, error) {
	s.log.Debug().
		Str("task", req.GetName()).
		Str("uri", req.GetURI()).
		Msg("Publish task statistics")

	s.agentsMutex.Lock()
	{
		current := s.taskAgents[req.GetName()]
		var taskAgentRef *taskAgent
		for _, x := range current {
			if x.URI == req.GetURI() {
				taskAgentRef = x
				break
			}
		}
		if taskAgentRef == nil {
			taskAgentRef = &taskAgent{URI: req.GetURI()}
			current = append(current, taskAgentRef)
			s.taskAgents[req.GetName()] = current
		}
		taskAgentRef.Statistics.Data.Reset()
		taskAgentRef.Statistics.Data.Add(*req)
		taskAgentRef.Statistics.Timestamp = time.Now()
		sort.Slice(current, func(i, j int) bool { return current[i].Statistics.Timestamp.Before(current[j].Statistics.Timestamp) })
	}
	s.agentsMutex.Unlock()

	// Notify frontend clients
	s.hub.StatisticsChanged()

	return &empty.Empty{}, nil
}

// GetLinkStatistics returns statistics for selected (or all) links.
func (s *agentRegistry) GetLinkStatistics(ctx context.Context, in *pipeline.GetLinkStatisticsRequest) (*pipeline.GetLinkStatisticsResponse, error) {
	s.agentsMutex.Lock()
	defer s.agentsMutex.Unlock()

	isLinkRequested := func(linkName string) bool {
		if len(in.GetLinkNames()) == 0 {
			return true
		}
		for _, x := range in.GetLinkNames() {
			if x == linkName {
				return true
			}
		}
		return false
	}

	result := &pipeline.GetLinkStatisticsResponse{}
	for name, list := range s.linkAgents {
		if isLinkRequested(name) {
			stat := tracking.LinkStatistics{
				Name: name,
			}
			for _, x := range list {
				stat.Add(x.Statistics.Data)
			}
			result.Statistics = append(result.Statistics, &stat)
		}
	}

	return result, nil
}

// GetTaskStatistics returns statistics for selected (or all) tasks.
func (s *agentRegistry) GetTaskStatistics(ctx context.Context, in *pipeline.GetTaskStatisticsRequest) (*pipeline.GetTaskStatisticsResponse, error) {
	s.agentsMutex.Lock()
	defer s.agentsMutex.Unlock()

	isTaskRequested := func(taskName string) bool {
		if len(in.GetTaskNames()) == 0 {
			return true
		}
		for _, x := range in.GetTaskNames() {
			if x == taskName {
				return true
			}
		}
		return false
	}

	result := &pipeline.GetTaskStatisticsResponse{}
	for name, list := range s.taskAgents {
		if isTaskRequested(name) {
			stat := tracking.TaskStatistics{
				Name: name,
			}
			for _, x := range list {
				stat.Add(x.Statistics.Data)
			}
			result.Statistics = append(result.Statistics, &stat)
		}
	}

	return result, nil
}
