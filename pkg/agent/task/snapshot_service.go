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

package task

import (
	"bytes"
	"context"
	"text/template"
	"time"

	"github.com/dchest/uniuri"
	"github.com/golang/protobuf/ptypes"

	"github.com/AljabrIO/koalja-operator/pkg/annotatedvalue"
	koalja "github.com/AljabrIO/koalja-operator/pkg/apis/koalja/v1alpha1"
	ptask "github.com/AljabrIO/koalja-operator/pkg/task"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/rs/zerolog"
)

type SnapshotService interface {
	// Provide external API
	ptask.SnapshotServiceServer

	// Run the service until the given context is canceled
	Run(context.Context) error
	// "Execute" the executor for the given snapshot.
	// That means providing the snapshot to the executor
	// and waiting until it has acknowledged it.
	Execute(context.Context, *InputSnapshot) error
}

// snapshotService implements the SnapshotService provided
// to task executors for tasks with a custom launch policy.
type snapshotService struct {
	log      zerolog.Logger
	taskSpec *koalja.TaskSpec
	next     chan *ptask.Snapshot
	ack      chan string
}

// NewSnapshotService creates a new SnapshotService.
func NewSnapshotService(log zerolog.Logger, taskSpec *koalja.TaskSpec) (SnapshotService, error) {
	return &snapshotService{
		log:      log,
		taskSpec: taskSpec,
		next:     make(chan *ptask.Snapshot),
		ack:      make(chan string),
	}, nil
}

// Run the service until the given context is canceled
func (s *snapshotService) Run(ctx context.Context) error {
	<-ctx.Done()
	return nil
}

// "Execute" the executor for the given snapshot.
// That means providing the snapshot to the executor
// and waiting until it has acknowledged it.
func (s *snapshotService) Execute(ctx context.Context, inputSnapshot *InputSnapshot) error {
	// Build API compatible snapshot
	snapshot := s.createAPISnapshot(inputSnapshot)

	select {
	case s.next <- snapshot:
		// Ok, some Next request was waiting for this
	case <-ctx.Done():
		// Context canceled before a Next request was waiting for it
		return ctx.Err()
	}

	// Wait until it is acknowledged
	for {
		select {
		case id := <-s.ack:
			if id == snapshot.GetID() {
				// We're done
				return nil
			}
			// Oops, we got an unexpected ID
			s.log.Warn().
				Str("expected-id", snapshot.ID).
				Str("actual-id", id).
				Msg("Got unexpected snapshot ID in acknowledgment")
		case <-ctx.Done():
			// Context canceled
			return ctx.Err()
		}
	}
}

// Next pulls the task agent for the next available snapshot.
func (s *snapshotService) Next(ctx context.Context, req *ptask.NextRequest) (*ptask.NextResponse, error) {
	timeout, err := ptypes.Duration(req.GetWaitTimeout())
	if err != nil {
		return nil, maskAny(err)
	}
	select {
	case x := <-s.next:
		// Ok, we got a snapshot
		return &ptask.NextResponse{
			Snapshot:      x,
			NoSnapshotYet: false,
		}, nil
	case <-ctx.Done():
		// Context canceled
		return nil, maskAny(ctx.Err())
	case <-time.After(timeout):
		// Timeout expired
		return &ptask.NextResponse{
			Snapshot:      nil,
			NoSnapshotYet: true,
		}, nil
	}
}

// Acknowledge the processing of a snapshot
func (s *snapshotService) Ack(ctx context.Context, req *ptask.AckRequest) (*empty.Empty, error) {
	select {
	case s.ack <- req.GetSnapshotID():
		// Ok, we acknowledged it
		return &empty.Empty{}, nil
	case <-ctx.Done():
		// Context canceled
		return nil, maskAny(ctx.Err())
	}
}

// ExecuteTemplate is invoked to parse & execute a template
// with given snapshot.
func (s *snapshotService) ExecuteTemplate(ctx context.Context, req *ptask.ExecuteTemplateRequest) (*ptask.ExecuteTemplateResponse, error) {
	log := s.log.With().
		Str("template", req.GetTemplate()).
		Logger()

	// Build data
	data := s.buildDataMap(req.GetSnapshot())

	// Parse template
	t, err := template.New("x").Parse(req.GetTemplate())
	if err != nil {
		log.Debug().Err(err).Msg("Failed to parse template")
		return nil, maskAny(err)
	}
	w := &bytes.Buffer{}
	if err := t.Execute(w, data); err != nil {
		log.Debug().Err(err).Msg("Failed to execute template")
		return nil, maskAny(err)
	}
	return &ptask.ExecuteTemplateResponse{
		Result: w.Bytes(),
	}, nil
}

// buildDataMap builds a (template) data structure for the given snapshot.
func (s *snapshotService) buildDataMap(snapshot *ptask.Snapshot) map[string]interface{} {
	dataSnapshot := make(map[string]interface{})
	dataSnapshot["id"] = snapshot.GetID()

	avBuilder := func(inp *annotatedvalue.AnnotatedValue) map[string]interface{} {
		d := map[string]interface{}{
			"id":  inp.GetID(),
			"uri": inp.GetID(),
		}
		return d
	}

	for _, entry := range snapshot.GetInputs() {
		if taskInput, found := s.taskSpec.InputByName(entry.GetInputName()); found {
			if taskInput.GetMaxSequenceLength() > 1 {
				// Put annotated values in list
				var list []map[string]interface{}
				for _, av := range entry.GetAnnotatedValues() {
					list = append(list, avBuilder(av))
				}
				dataSnapshot[entry.GetInputName()] = list
			} else {
				// Put annotated value as single entry
				if len(entry.GetAnnotatedValues()) > 0 {
					dataSnapshot[entry.GetInputName()] = avBuilder(entry.GetAnnotatedValues()[0])
				}
			}
		}
	}

	data := map[string]interface{}{
		"snapshot": dataSnapshot,
		// TODO add other task info
	}
	return data
}

// createAPISnapshot creates a SnapshotService API compatible Snapshot from
// the given task agent snapshot.
func (s *snapshotService) createAPISnapshot(inputSnapshot *InputSnapshot) *ptask.Snapshot {
	result := &ptask.Snapshot{
		ID: uniuri.New(),
	}
	for k, v := range inputSnapshot.members {
		result.Inputs = append(result.Inputs, &ptask.SnapshotInputPair{
			InputName:       k,
			AnnotatedValues: v.GetSequence(),
		})
	}
	return result
}
