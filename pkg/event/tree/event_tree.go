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

package tree

import (
	"context"

	"github.com/AljabrIO/koalja-operator/pkg/event"
)

// EventTree is a tree of an event with all is inputs (and inputs of those...)
type EventTree struct {
	Event  event.Event
	Inputs map[string]*EventTree
}

// Build an event tree for the given event.
func Build(ctx context.Context, evt event.Event, r event.EventRegistryClient) (*EventTree, error) {
	tree := &EventTree{
		Event: evt,
	}
	if len(evt.GetSourceInputs()) > 0 {
		tree.Inputs = make(map[string]*EventTree)
		for _, inp := range evt.GetSourceInputs() {
			inpName := inp.GetInputName()
			resp, err := r.GetEventByID(ctx, &event.GetEventByIDRequest{ID: inp.GetID()})
			if err != nil {
				return nil, maskAny(err)
			}
			evt := resp.GetEvent()
			childTree, err := Build(ctx, *evt, r)
			if err != nil {
				return nil, maskAny(err)
			}
			tree.Inputs[inpName] = childTree
		}
	}
	return tree, nil
}

// ContainsID returns true if this tree contains an event with given ID.
func (t *EventTree) ContainsID(eventID string) bool {
	if t.Event.ID == eventID {
		return true
	}
	for _, child := range t.Inputs {
		if child.ContainsID(eventID) {
			return true
		}
	}
	return false
}
