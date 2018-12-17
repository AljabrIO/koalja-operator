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
	"fmt"
	"text/template"

	"github.com/rs/zerolog"
	"github.com/vincent-petithory/dataurl"

	"github.com/AljabrIO/koalja-operator/pkg/annotatedvalue"
	"github.com/AljabrIO/koalja-operator/pkg/fs"
	fsclient "github.com/AljabrIO/koalja-operator/pkg/fs/client"
)

type templateFunctions struct {
	Log        zerolog.Logger
	FileSystem fsclient.FileSystemClient
}

// newTemplateFunctions instantiates & initializes a new templateFunctions service.
func newTemplateFunctions(log zerolog.Logger, fsc fsclient.FileSystemClient) *templateFunctions {
	return &templateFunctions{
		Log:        log,
		FileSystem: fsc,
	}
}

// FuncMap builds a function map for use in templates.
func (tf *templateFunctions) FuncMap() template.FuncMap {
	return template.FuncMap{
		"readURI": tf.ReadURI,
	}
}

// ReadURI reads the content of the given URI and returns it.
func (tf *templateFunctions) ReadURI(uri string) ([]byte, error) {
	scheme := annotatedvalue.GetDataScheme(uri)
	switch scheme {
	case annotatedvalue.SchemeData:
		durl, err := dataurl.DecodeString(uri)
		if err != nil {
			tf.Log.Warn().Err(err).Msg("Failed to parse data URI")
			return nil, maskAny(err)
		}
		return durl.Data, nil
	case annotatedvalue.SchemeFile:
		ctx := context.Background()
		// TODO add call to FS for fetching content only
		resp, err := tf.FileSystem.CreateFileView(ctx, &fs.CreateFileViewRequest{
			URI:     uri,
			Preview: false,
		})
		if err != nil {
			tf.Log.Warn().Err(err).Msg("Failed to create file view")
			return nil, maskAny(err)
		}
		return resp.Content, nil
	default:
		return nil, fmt.Errorf("Unknown scheme '%s'", scheme)
	}
}
