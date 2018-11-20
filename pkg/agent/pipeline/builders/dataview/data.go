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

package dataview

import (
	"context"

	"github.com/vincent-petithory/dataurl"

	"github.com/AljabrIO/koalja-operator/pkg/agent/pipeline"
	"github.com/AljabrIO/koalja-operator/pkg/annotatedvalue"
)

type dataViewBuilder struct{}

func init() {
	pipeline.RegisterDataViewBuilder(annotatedvalue.SchemeData, dataViewBuilder{})
}

// Prepare input of a task for an input that uses an URI with data scheme.
func (b dataViewBuilder) CreateView(ctx context.Context, req *pipeline.GetDataViewRequest, deps *pipeline.CreateViewDependencies) (*pipeline.GetDataViewResponse, error) {
	uri := req.GetData()
	deps.Log.Debug().
		Str("uri", uri).
		Msg("Create data view")

	// Decode data
	durl, err := dataurl.DecodeString(uri)
	if err != nil {
		deps.Log.Warn().Err(err).Msg("Failed to parse data URI")
		return nil, maskAny(err)
	}

	return &pipeline.GetDataViewResponse{
		Content:     durl.Data,
		ContentType: durl.ContentType(),
	}, nil
}
