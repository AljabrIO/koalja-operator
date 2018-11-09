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
	"sync/atomic"

	"github.com/AljabrIO/koalja-operator/pkg/annotatedvalue"
	"github.com/AljabrIO/koalja-operator/pkg/tracking"
)

// InputSnapshot holds a single annotated value for every input of a task.
type InputSnapshot struct {
	members map[string]*inputPair
}

type inputPair struct {
	sequence  []avActPair
	stats     *tracking.TaskInputStatistics
	minSeqLen int
}

type avActPair struct {
	av  *annotatedvalue.AnnotatedValue
	ack func(context.Context, *annotatedvalue.AnnotatedValue) error
}

// IsReadyForExecution returns true when the length of the sequence
// of annotated values is equal to or above the minimum.
func (ip *inputPair) IsReadyForExecution() bool {
	return len(ip.sequence) >= ip.minSeqLen
}

// GetSequence returns the annotated value sequence.
func (ip *inputPair) GetSequence() []*annotatedvalue.AnnotatedValue {
	result := make([]*annotatedvalue.AnnotatedValue, len(ip.sequence))
	for i, x := range ip.sequence {
		result[i] = x.av
	}
	return result
}

// Clone the given pair
func (ip *inputPair) Clone() *inputPair {
	return &inputPair{
		sequence: append([]avActPair{}, ip.sequence...),
		stats:    ip.stats,
	}
}

// Add the annotated value to the given sequence.
// This will acknowledge any existing annotated value when the sequence
// overflows.
func (ip *inputPair) Add(ctx context.Context, av *annotatedvalue.AnnotatedValue, maxLen int, stats *tracking.TaskInputStatistics, ack func(context.Context, *annotatedvalue.AnnotatedValue) error) error {
	// Remove existing entry if sequence is full.
	for {
		if l := len(ip.sequence); l == 0 || l < maxLen {
			break
		}
		// Remove first entry of existing sequence
		first := ip.sequence[0]
		if first.ack != nil {
			if err := first.ack(ctx, first.av); err != nil {
				// Return error
				return err
			}
		}
		// Remove first from sequence
		ip.sequence = ip.sequence[1:]
		// Update statistics reflecting that we're skipping an annotated value
		atomic.AddInt64(&stats.AnnotatedValuesSkipped, 1)
	}

	// Add annotated value to sequence (if not nil)
	if av != nil {
		ip.sequence = append(ip.sequence, avActPair{av, ack})
	}
	return nil
}

// AckAll acknowledges all annotated values that need it.
func (ip *inputPair) AckAll(ctx context.Context) error {
	for _, v := range ip.sequence {
		if v.ack != nil {
			if err := v.ack(ctx, v.av); err != nil {
				return err
			}
			// Acknowledge no longer needed
			v.ack = nil
		}
	}
	return nil
}

// RemoveAck removes the acknowledge callback for all annotated values in the sequence.
func (ip *inputPair) RemoveAck() {
	for i := range ip.sequence {
		ip.sequence[i].ack = nil
	}
}

// IsReadyForExecution returns true when the number of inputs is equal
// to the expected number of inputs and all inputs are ready for execution.
func (s *InputSnapshot) IsReadyForExecution(expectedInputs int) bool {
	if len(s.members) != expectedInputs {
		return false
	}
	for _, ip := range s.members {
		if !ip.IsReadyForExecution() {
			return false
		}
	}
	return true
}

// GetSequenceLengthForInput returns the length of the sequence for the given input
// or 0 if no such input is found.
func (s *InputSnapshot) GetSequenceLengthForInput(inputName string) int {
	if ip, found := s.members[inputName]; found {
		return len(ip.sequence)
	}
	return 0
}

// GetSequence returns the annotated value sequence for the input with given name.
func (s *InputSnapshot) GetSequence(inputName string) []*annotatedvalue.AnnotatedValue {
	if ip, found := s.members[inputName]; found {
		return ip.GetSequence()
	}
	return nil
}

// Clone the snapshot
func (s *InputSnapshot) Clone() *InputSnapshot {
	result := &InputSnapshot{}
	for k, v := range s.members {
		result.m()[k] = v.Clone()
	}
	return result
}

// m returns the members map, initializing it when needed.
// Requires a locked mutex.
func (s *InputSnapshot) m() map[string]*inputPair {
	if s.members == nil {
		s.members = make(map[string]*inputPair)
	}
	return s.members
}

// Set the annotated value at the given input name.
// This will acknowledge any existing annotated value.
func (s *InputSnapshot) Set(ctx context.Context, inputName string, av *annotatedvalue.AnnotatedValue, minSeqLen, maxSeqLen int, stats *tracking.TaskInputStatistics, ack func(context.Context, *annotatedvalue.AnnotatedValue) error) error {
	m := s.m()
	ip, found := m[inputName]
	if !found {
		ip = &inputPair{
			stats:     stats,
			minSeqLen: minSeqLen,
		}
		m[inputName] = ip
	}

	// Add to inputPair
	if err := ip.Add(ctx, av, maxSeqLen, stats, ack); err != nil {
		return err
	}
	return nil
}

// Delete the annotated value at the given input name.
// Does not acnowledge the annotated value!
func (s *InputSnapshot) Delete(inputName string) {
	delete(s.m(), inputName)
}

// RemoveAck removes the acknowledge callback for the annotated value at the given input name.
func (s *InputSnapshot) RemoveAck(inputName string) {
	if x, found := s.m()[inputName]; found {
		x.RemoveAck()
	}
}

// AckAll acknowledges all annotated values that need it.
func (s *InputSnapshot) AckAll(ctx context.Context) error {
	for _, v := range s.members {
		if err := v.AckAll(ctx); err != nil {
			return err
		}
	}
	return nil
}

// AddInProgressStatistics adds the given delta to all AnnotatedValuesInProgress statistics
// of the annotated values in this snapshot.
func (s *InputSnapshot) AddInProgressStatistics(delta int64) {
	for _, v := range s.members {
		atomic.AddInt64(&v.stats.AnnotatedValuesInProgress, delta*int64(len(v.sequence)))
	}
}

// AddProcessedStatistics adds the given delta to all AnnotatedValuesProcessed statistics
// of the annotated values in this snapshot.
func (s *InputSnapshot) AddProcessedStatistics(delta int64) {
	for _, v := range s.members {
		atomic.AddInt64(&v.stats.AnnotatedValuesProcessed, delta*int64(len(v.sequence)))
	}
}
