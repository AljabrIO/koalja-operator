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

package tracking

import "sort"

// Add the values of the source statistic to the target statistic.
func (target *LinkStatistics) Add(source LinkStatistics) {
	if target.Name == "" {
		target.Name = source.Name
	}
	if target.URI == "" {
		target.URI = source.URI
	}
	target.EventsWaiting += source.EventsWaiting
	target.EventsInProgress += source.EventsInProgress
	target.EventsAcknowledged += source.EventsAcknowledged
}

// InputByName returns the statistic for the input with given name,
// creating one if needed.
func (target *TaskStatistics) InputByName(name string) *TaskInputStatistics {
	for _, x := range target.Inputs {
		if x.Name == name {
			return x
		}
	}
	s := &TaskInputStatistics{Name: name}
	target.Inputs = append(target.Inputs, s)
	sort.Slice(target.Inputs, func(i, j int) bool { return target.Inputs[i].Name < target.Inputs[j].Name })
	return s
}

// OutputByName returns the statistic for the output with given name,
// creating one if needed.
func (target *TaskStatistics) OutputByName(name string) *TaskOutputStatistics {
	for _, x := range target.Outputs {
		if x.Name == name {
			return x
		}
	}
	s := &TaskOutputStatistics{Name: name}
	target.Outputs = append(target.Outputs, s)
	sort.Slice(target.Outputs, func(i, j int) bool { return target.Outputs[i].Name < target.Outputs[j].Name })
	return s
}

// Add the values of the source statistic to the target statistic.
func (target *TaskStatistics) Add(source TaskStatistics) {
	if target.Name == "" {
		target.Name = source.Name
	}
	if target.URI == "" {
		target.URI = source.URI
	}
	target.SnapshotsWaiting += source.SnapshotsWaiting
	target.SnapshotsInProgress += source.SnapshotsInProgress
	target.SnapshotsSucceeded += source.SnapshotsSucceeded
	target.SnapshotsFailed += source.SnapshotsFailed

	for _, s := range source.Inputs {
		t := target.InputByName(s.Name)
		t.Add(*s)
	}

	for _, s := range source.Outputs {
		t := target.OutputByName(s.Name)
		t.Add(*s)
	}
}

// Add the values of the source statistic to the target statistic.
func (target *TaskInputStatistics) Add(source TaskInputStatistics) {
	if target.Name == "" {
		target.Name = source.Name
	}
	target.EventsReceived += source.EventsReceived
	target.EventsInProgress += source.EventsInProgress
	target.EventsProcessed += source.EventsProcessed
	target.EventsSkipped += source.EventsSkipped
}

// Add the values of the source statistic to the target statistic.
func (target *TaskOutputStatistics) Add(source TaskOutputStatistics) {
	if target.Name == "" {
		target.Name = source.Name
	}
	target.EventsPublished += source.EventsPublished
}
