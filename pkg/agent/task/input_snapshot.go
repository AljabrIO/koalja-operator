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
	"sync"
	"sync/atomic"

	"github.com/AljabrIO/koalja-operator/pkg/agent/pipeline"
	"github.com/AljabrIO/koalja-operator/pkg/event"
)

// InputSnapshot holds a single event for every input of a task.
type InputSnapshot struct {
	members map[string]eventActPair
	mutex   sync.Mutex
}

type eventActPair struct {
	e     *event.Event
	stats *pipeline.TaskInputStatistics
	ack   func(context.Context, *event.Event) error
}

// CreateTuple copies all events into a new tuple, if the number of events
// is equal to the given expected number.
// Returns nil if number of events is different from given expected number.
func (s *InputSnapshot) CreateTuple(expectedEvents int) *event.EventTuple {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if len(s.members) != expectedEvents {
		return nil
	}
	result := &event.EventTuple{}
	result.Members = make(map[string]*event.Event)
	for k, v := range s.members {
		result.Members[k] = v.e
	}
	return result
}

// HasEvent returns true if the snapshot has an event for the input with given name.
func (s *InputSnapshot) HasEvent(inputName string) bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	_, found := s.members[inputName]
	return found
}

// Get returns the event for the input with given name.
func (s *InputSnapshot) Get(inputName string) *event.Event {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.members[inputName].e
}

// Clone the snapshot
func (s *InputSnapshot) Clone() *InputSnapshot {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	result := &InputSnapshot{}
	for k, v := range s.members {
		result.m()[k] = v
	}
	return result
}

// m returns the members map, initializing it when needed.
// Requires a locked mutex.
func (s *InputSnapshot) m() map[string]eventActPair {
	if s.members == nil {
		s.members = make(map[string]eventActPair)
	}
	return s.members
}

// Set the event at the given input name.
// This will acknowledge any existing event.
func (s *InputSnapshot) Set(ctx context.Context, inputName string, e *event.Event, stats *pipeline.TaskInputStatistics, ack func(context.Context, *event.Event) error) error {
	s.mutex.Lock()
	previous, found := s.m()[inputName]
	delete(s.m(), inputName)
	s.mutex.Unlock()
	if found {
		// Ack previous (if needed)
		if previous.ack != nil {
			if err := previous.ack(ctx, previous.e); err != nil {
				// Restore previous
				s.mutex.Lock()
				s.m()[inputName] = previous
				s.mutex.Unlock()
				// Return error
				return err
			}
		}
		// Update statistics reflecting that we're skipping an event
		atomic.AddInt64(&previous.stats.EventsSkipped, 1)
	}

	// Set new entry if not nil
	if e != nil {
		s.mutex.Lock()
		s.m()[inputName] = eventActPair{e, stats, ack}
		s.mutex.Unlock()
	}
	return nil
}

// Delete the event at the given input name.
// Does not acnowledge the event!
func (s *InputSnapshot) Delete(inputName string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	delete(s.m(), inputName)
}

// RemoveAck removes the acknowledge callback for the event at the given input name.
func (s *InputSnapshot) RemoveAck(inputName string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if x, found := s.m()[inputName]; found {
		s.members[inputName] = eventActPair{e: x.e, stats: x.stats, ack: nil}
	}
}

// AckAll acknowledges all events that need it.
func (s *InputSnapshot) AckAll(ctx context.Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for _, v := range s.members {
		if v.ack != nil {
			if err := v.ack(ctx, v.e); err != nil {
				return err
			}
		}
	}
	return nil
}

// AddInProgressStatistics adds the given delta to all EventsInProgress statistics
// of the events in this snapshot.
func (s *InputSnapshot) AddInProgressStatistics(delta int64) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for _, v := range s.members {
		atomic.AddInt64(&v.stats.EventsInProgress, delta)
	}
}

// AddProcessedStatistics adds the given delta to all EventsProcessed statistics
// of the events in this snapshot.
func (s *InputSnapshot) AddProcessedStatistics(delta int64) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for _, v := range s.members {
		atomic.AddInt64(&v.stats.EventsProcessed, delta)
	}
}
