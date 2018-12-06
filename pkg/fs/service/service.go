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
	"fmt"
	"net"

	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/AljabrIO/koalja-operator/pkg/constants"
	"github.com/AljabrIO/koalja-operator/pkg/fs"
	"github.com/AljabrIO/koalja-operator/pkg/util/retry"
)

// Service implements the filesystem service.
type Service struct {
	log             zerolog.Logger
	port            int
	uri             string
	fs              FileSystemServer
	c               client.Client
	scheme          *runtime.Scheme
	ns              string
	volumePluginDir string
}

// FileSystemServer API
type FileSystemServer interface {
	// Full FileSystemServer API
	fs.FileSystemServer

	// Register any additional GRPC APIs
	Register(*grpc.Server)

	// Run the FileSystemServer until the given context is canceled
	Run(context.Context) error
}

// APIDependencies provides some dependencies to API builder implementations
type APIDependencies struct {
	// Kubernetes client
	client.Client
	// Namespace in which this fs is running
	Namespace string
	// API scheme
	Scheme *runtime.Scheme
}

// APIBuilder is an interface provided by a FileSystem implementation
type APIBuilder interface {
	NewFileSystem(ctx context.Context, deps APIDependencies) (FileSystemServer, error)
}

// NewService creates a new Service instance.
func NewService(log zerolog.Logger, config *rest.Config, scheme *runtime.Scheme, builder APIBuilder, volumePluginDir string) (*Service, error) {
	var c client.Client
	ctx := context.Background()
	if err := retry.Do(ctx, func(ctx context.Context) error {
		var err error
		c, err = client.New(config, client.Options{})
		return err
	}, retry.Timeout(constants.TimeoutK8sClient)); err != nil {
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
		Scheme:    scheme,
	}
	fs, err := builder.NewFileSystem(ctx, deps)
	if err != nil {
		return nil, err
	}
	return &Service{
		log:             log,
		port:            port,
		fs:              fs,
		c:               c,
		scheme:          scheme,
		ns:              ns,
		volumePluginDir: volumePluginDir,
	}, nil
}

// Run the pipeline agent until the given context is canceled.
func (s *Service) Run(ctx context.Context) error {
	if err := s.createFlexDriverInstallDaemonsets(ctx); err != nil {
		return maskAny(err)
	}

	g, lctx := errgroup.WithContext(ctx)
	g.Go(func() error { return s.runAPI(lctx) })
	g.Go(func() error { return s.fs.Run(lctx) })
	if err := g.Wait(); err != nil {
		return maskAny(err)
	}
	return nil
}

// runAPI runs the GRPC server until the given context is canceled.
func (s *Service) runAPI(ctx context.Context) error {
	addr := fmt.Sprintf("0.0.0.0:%d", s.port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		s.log.Fatal().Err(err).Msg("failed to listen")
	}
	svr := grpc.NewServer()
	fs.RegisterFileSystemServer(svr, s.fs)
	s.fs.Register(svr)
	// Register reflection service on gRPC server.
	reflection.Register(svr)
	go func() {
		if err := svr.Serve(lis); err != nil {
			s.log.Fatal().Err(err).Msg("failed to serve")
		}
	}()
	s.log.Info().Msgf("Started filesystem, listening on %s", addr)
	<-ctx.Done()
	svr.GracefulStop()
	return nil
}
