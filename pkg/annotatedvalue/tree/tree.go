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

	"github.com/AljabrIO/koalja-operator/pkg/annotatedvalue"
)

// AnnotatedValueTree is a tree of an annotated value with all is inputs (and inputs of those...)
type AnnotatedValueTree struct {
	AnnotatedValue annotatedvalue.AnnotatedValue
	Inputs         map[string]*AnnotatedValueTree
}

// Build an annotated value tree for the given annotated value.
func Build(ctx context.Context, av annotatedvalue.AnnotatedValue, r annotatedvalue.AnnotatedValueRegistryClient) (*AnnotatedValueTree, error) {
	tree := &AnnotatedValueTree{
		AnnotatedValue: av,
	}
	if len(av.GetSourceInputs()) > 0 {
		tree.Inputs = make(map[string]*AnnotatedValueTree)
		for _, inp := range av.GetSourceInputs() {
			inpName := inp.GetInputName()
			resp, err := r.GetByID(ctx, &annotatedvalue.GetByIDRequest{ID: inp.GetID()})
			if err != nil {
				return nil, maskAny(err)
			}
			av := resp.GetAnnotatedValue()
			childTree, err := Build(ctx, *av, r)
			if err != nil {
				return nil, maskAny(err)
			}
			tree.Inputs[inpName] = childTree
		}
	}
	return tree, nil
}

// ContainsID returns true if this tree contains an annotated value with given ID.
func (t *AnnotatedValueTree) ContainsID(id string) bool {
	if t.AnnotatedValue.ID == id {
		return true
	}
	for _, child := range t.Inputs {
		if child.ContainsID(id) {
			return true
		}
	}
	return false
}
