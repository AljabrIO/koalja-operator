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

	"github.com/AljabrIO/koalja-operator/pkg/annotatedvalue"
	"github.com/AljabrIO/koalja-operator/pkg/tracking"
)

// InputSnapshot holds a single annotated value for every input of a task.
type InputSnapshot struct {
	members map[string]avActPair
	mutex   sync.Mutex
}

type avActPair struct {
	av    *annotatedvalue.AnnotatedValue
	stats *tracking.TaskInputStatistics
	ack   func(context.Context, *annotatedvalue.AnnotatedValue) error
}

// CreateTuple copies all annotated values into a new tuple, if the number of annotated values
// is equal to the given expected number.
// Returns nil if number of annotated values is different from given expected number.
func (s *InputSnapshot) CreateTuple(expectedAnnotatedEvents int) *annotatedvalue.AnnotatedValueTuple {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if len(s.members) != expectedAnnotatedEvents {
		return nil
	}
	result := &annotatedvalue.AnnotatedValueTuple{}
	result.Members = make(map[string]*annotatedvalue.AnnotatedValue)
	for k, v := range s.members {
		result.Members[k] = v.av
	}
	return result
}

// HasAnnotatedValue returns true if the snapshot has an annotated value for the input with given name.
func (s *InputSnapshot) HasAnnotatedValue(inputName string) bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	_, found := s.members[inputName]
	return found
}

// Get returns the annotated value for the input with given name.
func (s *InputSnapshot) Get(inputName string) *annotatedvalue.AnnotatedValue {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.members[inputName].av
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
func (s *InputSnapshot) m() map[string]avActPair {
	if s.members == nil {
		s.members = make(map[string]avActPair)
	}
	return s.members
}

// Set the annotated value at the given input name.
// This will acknowledge any existing annotated value.
func (s *InputSnapshot) Set(ctx context.Context, inputName string, av *annotatedvalue.AnnotatedValue, stats *tracking.TaskInputStatistics, ack func(context.Context, *annotatedvalue.AnnotatedValue) error) error {
	s.mutex.Lock()
	previous, found := s.m()[inputName]
	delete(s.m(), inputName)
	s.mutex.Unlock()
	if found {
		// Ack previous (if needed)
		if previous.ack != nil {
			if err := previous.ack(ctx, previous.av); err != nil {
				// Restore previous
				s.mutex.Lock()
				s.m()[inputName] = previous
				s.mutex.Unlock()
				// Return error
				return err
			}
		}
		// Update statistics reflecting that we're skipping an annotated value
		atomic.AddInt64(&previous.stats.AnnotatedValuesSkipped, 1)
	}

	// Set new entry if not nil
	if av != nil {
		s.mutex.Lock()
		s.m()[inputName] = avActPair{av, stats, ack}
		s.mutex.Unlock()
	}
	return nil
}

// Delete the annotated value at the given input name.
// Does not acnowledge the annotated value!
func (s *InputSnapshot) Delete(inputName string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	delete(s.m(), inputName)
}

// RemoveAck removes the acknowledge callback for the annotated value at the given input name.
func (s *InputSnapshot) RemoveAck(inputName string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if x, found := s.m()[inputName]; found {
		s.members[inputName] = avActPair{av: x.av, stats: x.stats, ack: nil}
	}
}

// AckAll acknowledges all annotated values that need it.
func (s *InputSnapshot) AckAll(ctx context.Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for _, v := range s.members {
		if v.ack != nil {
			if err := v.ack(ctx, v.av); err != nil {
				return err
			}
		}
	}
	return nil
}

// AddInProgressStatistics adds the given delta to all AnnotatedValuesInProgress statistics
// of the annotated values in this snapshot.
func (s *InputSnapshot) AddInProgressStatistics(delta int64) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for _, v := range s.members {
		atomic.AddInt64(&v.stats.AnnotatedValuesInProgress, delta)
	}
}

// AddProcessedStatistics adds the given delta to all AnnotatedValuesProcessed statistics
// of the annotated values in this snapshot.
func (s *InputSnapshot) AddProcessedStatistics(delta int64) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for _, v := range s.members {
		atomic.AddInt64(&v.stats.AnnotatedValuesProcessed, delta)
	}
}
