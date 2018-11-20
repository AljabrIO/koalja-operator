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

	"github.com/AljabrIO/koalja-operator/pkg/fs"
	"github.com/rs/zerolog/log"

	"github.com/AljabrIO/koalja-operator/pkg/agent/pipeline"
	"github.com/AljabrIO/koalja-operator/pkg/annotatedvalue"
)

type fileViewBuilder struct{}

func init() {
	pipeline.RegisterDataViewBuilder(annotatedvalue.SchemeFile, fileViewBuilder{})
}

// Prepare input of a task for an input that uses an URI with data scheme.
func (b fileViewBuilder) CreateView(ctx context.Context, req *pipeline.GetDataViewRequest, deps *pipeline.CreateViewDependencies) (*pipeline.GetDataViewResponse, error) {
	uri := req.GetData()
	deps.Log.Debug().
		Str("uri", uri).
		Msg("Create data view")

	resp, err := deps.FileSystem.CreateFileView(ctx, &fs.CreateFileViewRequest{
		URI:     req.GetData(),
		Preview: req.GetPreview(),
	})
	if err != nil {
		log.Debug().Err(err).Msg("CreateFileView failed")
		return nil, err
	}

	return &pipeline.GetDataViewResponse{
		Content:     resp.GetContent(),
		ContentType: resp.GetContentType(),
	}, nil
}
