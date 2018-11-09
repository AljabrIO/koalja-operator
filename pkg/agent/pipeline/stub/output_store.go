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

	koalja "github.com/AljabrIO/koalja-operator/pkg/apis/koalja/v1alpha1"
	"github.com/AljabrIO/koalja-operator/pkg/constants"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/golang/protobuf/ptypes/empty"

	"github.com/AljabrIO/koalja-operator/pkg/agent/pipeline"
	"github.com/AljabrIO/koalja-operator/pkg/annotatedvalue"
	"github.com/AljabrIO/koalja-operator/pkg/annotatedvalue/registry"
	"github.com/AljabrIO/koalja-operator/pkg/annotatedvalue/tree"
	"github.com/AljabrIO/koalja-operator/pkg/util/retry"
	"github.com/rs/zerolog"
)

// outputStore is an in-memory implementation of an annotated value queue.
type outputStore struct {
	log                  zerolog.Logger
	agentRegistry        *agentRegistry
	avRegistry           registry.AnnotatedValueRegistryClient
	pipeline             *koalja.Pipeline
	annotatedValues      []*tree.AnnotatedValueTree
	annotatedValuesMutex sync.Mutex
}

// newOutputStore creates a new output store
func newOutputStore(log zerolog.Logger, r registry.AnnotatedValueRegistryClient, pipeline *koalja.Pipeline, agentRegistry *agentRegistry) *outputStore {
	return &outputStore{
		log:           log,
		avRegistry:    r,
		pipeline:      pipeline,
		agentRegistry: agentRegistry,
	}
}

// Publish an annotated value
func (s *outputStore) Publish(ctx context.Context, req *annotatedvalue.PublishRequest) (*annotatedvalue.PublishResponse, error) {
	s.log.Debug().Interface("annotatedvalue", req.AnnotatedValue).Msg("Publish request")
	// Try to record annotated value in registry
	av := *req.GetAnnotatedValue()
	av.Link = "" // the end
	if err := retry.Do(ctx, func(ctx context.Context) error {
		s.log.Debug().Msg("RecordEvent attempt start")
		if _, err := s.avRegistry.Record(ctx, &av); err != nil {
			s.log.Debug().Err(err).Msg("Record attempt failed")
			return maskAny(err)
		}
		return nil
	}, retry.Timeout(constants.TimeoutRecordAnnotatedValue)); err != nil {
		s.log.Error().Err(err).Msg("Failed to record annotated value")
		return nil, maskAny(err)
	}

	// Build annotated value tree
	avTree, err := tree.Build(ctx, av, s.avRegistry)
	if err != nil {
		return nil, maskAny(err)
	}

	// Now put annotated value in in-memory list
	s.annotatedValuesMutex.Lock()
	defer s.annotatedValuesMutex.Unlock()
	s.annotatedValues = append(s.annotatedValues, avTree)
	return &annotatedvalue.PublishResponse{}, nil
}

// GetOutputAnnotatedValues returns all annotated values (resulting from task outputs that
// are not connected to inputs of other tasks) that match the given filter.
func (s *outputStore) GetOutputAnnotatedValues(ctx context.Context, req *pipeline.OutputAnnotatedValuesRequest) (*pipeline.OutputAnnotatedValues, error) {
	s.log.Debug().Interface("req", req).Msg("GetOutputAnnotatedValues request")
	s.annotatedValuesMutex.Lock()
	defer s.annotatedValuesMutex.Unlock()

	resp := &pipeline.OutputAnnotatedValues{}
	for _, tree := range s.annotatedValues {
		if isMatch(tree, req) {
			av := tree.AnnotatedValue // Create clone
			resp.AnnotatedValues = append(resp.AnnotatedValues, &av)
		}
	}

	return resp, nil
}

// GetPipeline returns the pipeline resource.
func (s *outputStore) GetPipeline(context.Context, *empty.Empty) (*koalja.PipelineSpec, error) {
	return &s.pipeline.Spec, nil
}

// GetLinkStatistics returns statistics for selected (or all) links.
func (s *outputStore) GetLinkStatistics(ctx context.Context, req *pipeline.GetLinkStatisticsRequest) (*pipeline.GetLinkStatisticsResponse, error) {
	result, err := s.agentRegistry.GetLinkStatistics(ctx, req)
	if err != nil {
		return nil, maskAny(err)
	}
	return result, nil
}

// GetTaskStatistics returns statistics for selected (or all) tasks.
func (s *outputStore) GetTaskStatistics(ctx context.Context, req *pipeline.GetTaskStatisticsRequest) (*pipeline.GetTaskStatisticsResponse, error) {
	result, err := s.agentRegistry.GetTaskStatistics(ctx, req)
	if err != nil {
		return nil, maskAny(err)
	}
	return result, nil
}

// isMatch returns true when the given annotated value tree matches the given request.
func isMatch(e *tree.AnnotatedValueTree, req *pipeline.OutputAnnotatedValuesRequest) bool {
	createdAt, _ := ptypes.TimestampFromProto(e.AnnotatedValue.GetCreatedAt())
	if tsPB := req.GetCreatedAfter(); tsPB != nil {
		ts, _ := ptypes.TimestampFromProto(tsPB)
		if createdAt.Before(ts) {
			return false
		}
	}
	if tsPB := req.GetCreatedBefore(); tsPB != nil {
		ts, _ := ptypes.TimestampFromProto(tsPB)
		if createdAt.After(ts) {
			return false
		}
	}
	if taskNames := req.GetTaskNames(); len(taskNames) > 0 {
		found := false
		for _, name := range taskNames {
			if e.AnnotatedValue.GetSourceTask() == name {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	if ids := req.GetAnnotatedValueIDs(); len(ids) > 0 {
		found := false
		for _, id := range ids {
			if e.ContainsID(id) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
