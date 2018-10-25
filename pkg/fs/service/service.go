/*
Copyright 2018 Aljabr Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package service

import (
	"context"
	"time"

	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	fmt "fmt"
	"log"
	"net"

	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/AljabrIO/koalja-operator/pkg/constants"
	"github.com/AljabrIO/koalja-operator/pkg/fs"
	"github.com/AljabrIO/koalja-operator/pkg/util"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Service implements the filesystem service.
type Service struct {
	port int
	uri  string
	fs   fs.FileSystemServer
}

// APIDependencies provides some dependencies to API builder implementations
type APIDependencies struct {
	// Kubernetes client
	client.Client
	// Namespace in which this fs is running
	Namespace string
}

// APIBuilder is an interface provided by a FileSystem implementation
type APIBuilder interface {
	NewFileSystem(deps APIDependencies) (fs.FileSystemServer, error)
}

// NewService creates a new Service instance.
func NewService(config *rest.Config, builder APIBuilder) (*Service, error) {
	var c client.Client
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if err := util.Retry(ctx, func(ctx context.Context) error {
		var err error
		c, err = client.New(config, client.Options{})
		return err
	}); err != nil {
		return nil, err
	}
	ns, err := constants.GetNamespace()
	if err != nil {
		return nil, err
	}
	port, err := constants.GetAPIPort()
	if err != nil {
		return nil, err
	}
	deps := APIDependencies{
		Client:    c,
		Namespace: ns,
	}
	fs, err := builder.NewFileSystem(deps)
	if err != nil {
		return nil, err
	}
	return &Service{
		port: port,
		fs:   fs,
	}, nil
}

// Run the pipeline agent until the given context is canceled.
func (s *Service) Run(ctx context.Context) error {
	addr := fmt.Sprintf("0.0.0.0:%d", s.port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	svr := grpc.NewServer()
	fs.RegisterFileSystemServer(svr, s.fs)
	// Register reflection service on gRPC server.
	reflection.Register(svr)
	go func() {
		if err := svr.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()
	log.Printf("Started filesystem, listening on %s", addr)
	<-ctx.Done()
	svr.GracefulStop()
	return nil
}
