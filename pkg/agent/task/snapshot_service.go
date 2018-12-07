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
	"context"
	"time"

	"github.com/golang/protobuf/ptypes"

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
	log  zerolog.Logger
	next chan *ptask.Snapshot
	ack  chan string
}

// NewSnapshotService creates a new SnapshotService.
func NewSnapshotService(log zerolog.Logger) (SnapshotService, error) {
	return &snapshotService{
		log:  log,
		next: make(chan *ptask.Snapshot),
		ack:  make(chan string),
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
	snapshot := &ptask.Snapshot{}

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
