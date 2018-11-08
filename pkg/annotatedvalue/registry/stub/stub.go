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
	"fmt"
	"sync"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/rs/zerolog"

	"github.com/AljabrIO/koalja-operator/pkg/annotatedvalue"
	"github.com/AljabrIO/koalja-operator/pkg/annotatedvalue/registry"
)

// stub is an in-memory implementation of an annotatedvalue registry.
type stub struct {
	log           zerolog.Logger
	registry      map[string]*annotatedvalue.AnnotatedValue
	registryMutex sync.Mutex
}

// NewStub initializes a new stub
func NewStub(log zerolog.Logger) registry.APIBuilder {
	return &stub{
		log:      log,
		registry: make(map[string]*annotatedvalue.AnnotatedValue),
	}
}

// NewRegistry "builds" a new registry
func (s *stub) NewRegistry(deps registry.APIDependencies) (annotatedvalue.AnnotatedValueRegistryServer, error) {
	return s, nil
}

// Record the given annotatedvalue in the registry
func (s *stub) Record(ctx context.Context, e *annotatedvalue.AnnotatedValue) (*empty.Empty, error) {
	s.log.Debug().Str("annotatedvalue", e.GetID()).Msg("Record request")
	s.registryMutex.Lock()
	defer s.registryMutex.Unlock()

	s.registry[e.GetID()] = e
	return &empty.Empty{}, nil
}

// GetByID returns the event with given ID.
func (s *stub) GetByID(ctx context.Context, req *annotatedvalue.GetByIDRequest) (*annotatedvalue.GetResponse, error) {
	s.log.Debug().Str("annotatedvalue", req.GetID()).Msg("GetByID request")
	s.registryMutex.Lock()
	defer s.registryMutex.Unlock()

	if av, found := s.registry[req.GetID()]; found {
		return &annotatedvalue.GetResponse{AnnotatedValue: av}, nil
	}
	return nil, fmt.Errorf("AnnotatedValue '%s' not found", req.GetID())
}

// GetByTaskAndData returns the event with given ID.
func (s *stub) GetByTaskAndData(ctx context.Context, req *annotatedvalue.GetByTaskAndDataRequest) (*annotatedvalue.GetResponse, error) {
	s.log.Debug().
		Str("task", req.GetSourceTask()).
		Str("data", req.GetData()).
		Msg("GetByTaskAndData request")
	s.registryMutex.Lock()
	defer s.registryMutex.Unlock()

	for _, av := range s.registry {
		if av.GetSourceTask() == req.GetSourceTask() && av.GetData() == req.GetData() {
			return &annotatedvalue.GetResponse{AnnotatedValue: av}, nil
		}
	}
	return nil, fmt.Errorf("No such annotated value")
}
