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

package filedrop

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/AljabrIO/koalja-operator/pkg/fs"
	"github.com/AljabrIO/koalja-operator/pkg/task"
	"github.com/AljabrIO/koalja-operator/pkg/util"

	"github.com/dchest/uniuri"
	"golang.org/x/sync/errgroup"
)

// runServer runs a webserver until the given context is canceled
func (s *Service) runServer(ctx context.Context) error {
	mux := &http.ServeMux{}
	mux.HandleFunc("/upload", s.uploadHandler)
	svr := &http.Server{
		Addr:    "0.0.0.0:8080",
		Handler: mux,
	}
	g, lctx := errgroup.WithContext(ctx)
	g.Go(svr.ListenAndServe)
	g.Go(func() error {
		select {
		case <-lctx.Done():
		case <-ctx.Done():
			svr.Shutdown(context.Background())
		}
		return nil
	})
	if err := g.Wait(); err != nil {
		return maskAny(err)
	}
	return nil
}

// uploadHandler is responsible for handling upload requests.
func (s *Service) uploadHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := s.log.With().Str("remote-address", r.RemoteAddr).Logger()
	// Allocate a file
	localPath := filepath.Join(s.DropFolder, fmt.Sprintf("upload-%s", uniuri.NewLen(6)))
	os.MkdirAll(s.DropFolder, 0755)
	log = log.With().Str("localPath", localPath).Logger()
	log.Debug().Msg("creating upload file")
	f, err := os.Create(localPath)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create local file")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer f.Close()
	// Copy content into file
	body := r.Body
	defer body.Close()
	if _, err := io.Copy(f, body); err != nil {
		log.Error().Err(err).Msg("Failed to copy request body to local file")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// Create URI
	relLocalPath, err := filepath.Rel(s.MountPath, localPath)
	if err != nil {
		log.Error().Err(err).Msg("Failed to make relative local path")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	resp, err := s.fsClient.CreateFileURI(ctx, &fs.CreateFileURIRequest{
		Scheme:     s.fsScheme,
		VolumeName: s.VolumeName,
		NodeName:   s.NodeName,
		LocalPath:  relLocalPath,
		IsDir:      false,
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to create file URI")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// Signal task with file
	log.Debug().Str("uri", resp.GetURI()).Msg("signaling output ready")
	result := make(map[string]interface{})
	delay := time.Millisecond * 100
	for {
		if resp, err := s.ornClient.OutputReady(ctx, &task.OutputReadyRequest{
			AnnotatedValueData: resp.GetURI(),
			OutputName:         s.OutputName,
		}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		} else if resp.Accepted {
			// Output was accepted
			result["annotatedvalue-id"] = resp.GetAnnotatedValueID()
			break
		} else {
			// Output was not accepted, try again soon.
			select {
			case <-time.After(delay):
				// Try again
				delay = util.Backoff(delay, 1.5, time.Minute)
			case <-ctx.Done():
				// Context canceled
				log.Warn().Err(ctx.Err()).Msg("Context canceled")
				return
			}
		}
	}
	encodedResult, _ := json.Marshal(result)
	w.WriteHeader(http.StatusOK)
	w.Write(encodedResult)
}
