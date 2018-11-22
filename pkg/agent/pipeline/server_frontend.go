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
	"io"
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
func (s *Service) runFrontendServer(ctx context.Context, httpPort, grpcPort int, grpcServer FrontendServer) error {
	// Server HTTP API
	mux := gwruntime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithInsecure()}
	if err := RegisterFrontendHandlerFromEndpoint(ctx, mux, net.JoinHostPort("localhost", strconv.Itoa(grpcPort)), opts); err != nil {
		s.log.Error().Err(err).Msg("Failed to register HTTP gateway")
		return maskAny(err)
	}
	// Frontend
	var rootHandlerFunc gwruntime.HandlerFunc
	mux.Handle("GET", parsePattern("/v1/updates"), s.frontendHub.CreateHandler())
	mux.Handle("POST", parsePattern("/v1/data/view"), createGetDataViewHandler(mux, grpcServer))
	for path, file := range frontend.Assets.Files {
		handler := createAssetFileHandler(file)
		if path == "index.html" {
			rootHandlerFunc = handler
		}
		localPath := "/" + strings.TrimPrefix(path, "/")
		mux.Handle("GET", parsePattern(localPath), handler)
	}

	httpAddr := fmt.Sprintf("0.0.0.0:%d", httpPort)
	handler := createHandler(mux, rootHandlerFunc)
	go func() {
		if err := http.ListenAndServe(httpAddr, handler); err != nil {
			s.log.Fatal().Err(err).Msg("Failed to serve HTTP gateway")
		}
	}()
	s.log.Info().Msgf("Started pipeline agent HTTP gateway, listening on %s", httpAddr)

	// Wait until context canceled
	<-ctx.Done()
	return nil
}

// createHandler builds a HTTP request handler.
func createHandler(mux *gwruntime.ServeMux, rootHandler gwruntime.HandlerFunc) http.Handler {
	if rootHandler == nil {
		panic("no rootHandler found")
	}
	handler := func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			rootHandler(w, r, nil)
		} else if strings.ToLower(path) == "index.html" {
			http.Redirect(w, r, "/", http.StatusFound)
		} else {
			mux.ServeHTTP(w, r)
		}
	}
	return http.HandlerFunc(handler)
}

// createAssetFileHandler creates a gin handler to serve the content
// of the given asset file.
func createAssetFileHandler(file *assets.File) gwruntime.HandlerFunc {
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

// createGetDataViewHandler creates a handler to serve the content
// for Frontend.GetDataView where content is returned as HTTP stream
// instead of JSON encoded HTTP stream.
func createGetDataViewHandler(mux *gwruntime.ServeMux, grpcServer FrontendServer) gwruntime.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
		inboundMarshaler, _ := gwruntime.MarshalerForRequest(mux, r)
		var protoReq GetDataViewRequest

		if err := inboundMarshaler.NewDecoder(r.Body).Decode(&protoReq); err != nil && err != io.EOF {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		resp, err := grpcServer.GetDataView(r.Context(), &protoReq)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Length", strconv.Itoa(len(resp.GetContent())))
		w.Header().Set("Content-Type", resp.GetContentType())
		w.WriteHeader(http.StatusOK)
		w.Write(resp.GetContent())
	}
}
