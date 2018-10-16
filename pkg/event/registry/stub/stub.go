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

package main

import (
	"context"
	"fmt"
	"sync"

	"github.com/AljabrIO/koalja-operator/pkg/event"
	"github.com/AljabrIO/koalja-operator/pkg/event/registry"
	"github.com/golang/protobuf/ptypes/empty"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

// stub is an in-memory implementation of an event registry.
type stub struct {
	registry      map[string]*event.Event
	registryMutex sync.Mutex
}

// newStub initializes a new stub
func newStub() *stub {
	return &stub{
		registry: make(map[string]*event.Event),
	}
}

// NewEventRegistry "builds" a new registry
func (s *stub) NewEventRegistry(deps registry.APIDependencies) (event.EventRegistryServer, error) {
	return s, nil
}

// Record the given event in the registry
func (s *stub) RecordEvent(ctx context.Context, e *event.Event) (*empty.Empty, error) {
	s.registryMutex.Lock()
	defer s.registryMutex.Unlock()

	s.registry[e.GetID()] = e
	return &empty.Empty{}, nil
}

// GetEventByID returns the event with given ID.
func (s *stub) GetEventByID(ctx context.Context, req *event.GetEventByIDRequest) (*event.GetEventResponse, error) {
	s.registryMutex.Lock()
	defer s.registryMutex.Unlock()

	if e, found := s.registry[req.GetID()]; found {
		return &event.GetEventResponse{Event: e}, nil
	}
	return nil, fmt.Errorf("Event '%s' not found", req.GetID())
}

// GetEventByID returns the event with given ID.
func (s *stub) GetEventByTaskAndData(ctx context.Context, req *event.GetEventByTaskAndDataRequest) (*event.GetEventResponse, error) {
	s.registryMutex.Lock()
	defer s.registryMutex.Unlock()

	for _, e := range s.registry {
		if e.GetSourceTask() == req.GetSourceTask() && e.GetData() == req.GetData() {
			return &event.GetEventResponse{Event: e}, nil
		}
	}
	return nil, fmt.Errorf("No such event")
}
