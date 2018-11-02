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

package pipeline

import (
	"context"
	fmt "fmt"
	"net"
	"net/http"
	"strconv"
	"strings"

	gwruntime "github.com/grpc-ecosystem/grpc-gateway/runtime"
	assets "github.com/jessevdk/go-assets"
	grpc "google.golang.org/grpc"

	"github.com/AljabrIO/koalja-operator/frontend"
)

// runFrontendServer runs the HTTP frontend server until the given context is canceled.
func (s *Service) runFrontendServer(ctx context.Context, httpPort, grpcPort int) error {
	// Server HTTP API
	mux := gwruntime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithInsecure()}
	if err := RegisterFrontendHandlerFromEndpoint(ctx, mux, net.JoinHostPort("localhost", strconv.Itoa(grpcPort)), opts); err != nil {
		s.log.Error().Err(err).Msg("Failed to register HTTP gateway")
		return maskAny(err)
	}
	// Frontend
	mux.Handle("GET", parsePattern("/"), createAssetFileHandler(frontend.Assets.Files["index.html"]))
	mux.Handle("GET", parsePattern("/v1/updates"), s.frontendHub.CreateHandler())
	for path, file := range frontend.Assets.Files {
		localPath := "/" + strings.TrimPrefix(path, "/")
		mux.Handle("GET", parsePattern(localPath), createAssetFileHandler(file))
	}

	httpAddr := fmt.Sprintf("0.0.0.0:%d", httpPort)
	go func() {
		if err := http.ListenAndServe(httpAddr, mux); err != nil {
			s.log.Fatal().Err(err).Msg("Failed to serve HTTP gateway")
		}
	}()
	s.log.Info().Msgf("Started pipeline agent HTTP gateway, listening on %s", httpAddr)

	// Wait until context canceled
	<-ctx.Done()
	return nil
}

// createAssetFileHandler creates a gin handler to serve the content
// of the given asset file.
func createAssetFileHandler(file *assets.File) func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
	return func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
		http.ServeContent(w, r, file.Name(), file.ModTime(), file)
	}
}

// parsePattern parses a given local path into a pattern understood by GRPC gateway.
func parsePattern(localPath string) gwruntime.Pattern {
	localPath = strings.TrimSuffix(strings.TrimPrefix(localPath, "/"), "/")
	if localPath == "" {
		return gwruntime.MustPattern(gwruntime.NewPattern(1, []int{}, []string{}, ""))
	}
	parts := strings.Split(localPath, "/")
	ops := make([]int, 0, len(parts)*2)
	for i := range parts {
		ops = append(ops, 2, i)
	}
	return gwruntime.MustPattern(gwruntime.NewPattern(1, ops, parts, ""))
}
