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

package node

import (
	"context"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"golang.org/x/sync/errgroup"

	fmt "fmt"
	"log"
	"net"

	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/AljabrIO/koalja-operator/pkg/constants"
	"github.com/AljabrIO/koalja-operator/pkg/fs"
	"github.com/AljabrIO/koalja-operator/pkg/fs/service/local"
	"github.com/AljabrIO/koalja-operator/pkg/util"
	"github.com/rs/zerolog"
)

// Service implements the filesystem node daemon.
type Service struct {
	Config
	log          zerolog.Logger
	nodeRegistry local.NodeRegistryClient
	myIP         string
}

type Config struct {
	// Port to listen on
	Port int
	// Address of the node registry
	RegistryAddress string
	// Name of this node
	NodeName string
}

// NewService creates a new Service instance.
func NewService(log zerolog.Logger, config Config) (*Service, error) {
	// Find my own IP address
	myIP, err := constants.GetPodIP()
	if err != nil {
		log.Debug().Err(err).Msg("Failed to dial node registry")
		return nil, maskAny(err)
	}

	// Connect to node registry
	conn, err := grpc.Dial(config.RegistryAddress, grpc.WithInsecure())
	if err != nil {
		log.Debug().Err(err).Msg("Failed to dial node registry")
		return nil, maskAny(err)
	}
	nodeRegistry := local.NewNodeRegistryClient(conn)
	return &Service{
		Config:       config,
		log:          log,
		nodeRegistry: nodeRegistry,
		myIP:         myIP,
	}, nil
}

// Run the pipeline agent until the given context is canceled.
func (s *Service) Run(ctx context.Context) error {
	g, lctx := errgroup.WithContext(ctx)
	g.Go(func() error { return s.runServer(lctx) })
	g.Go(func() error { s.runNodeRegistration(lctx); return nil })
	if err := g.Wait(); err != nil {
		return maskAny(err)
	}
	return nil
}

// runServer runs the GRPC server.
func (s *Service) runServer(ctx context.Context) error {
	addr := fmt.Sprintf("0.0.0.0:%d", s.Config.Port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		s.log.Fatal().Err(err).Msg("failed to listen")
	}
	svr := grpc.NewServer()
	local.RegisterNodeServer(svr, s)
	// Register reflection service on gRPC server.
	reflection.Register(svr)
	go func() {
		if err := svr.Serve(lis); err != nil {
			s.log.Fatal().Err(err).Msg("failed to serve")
		}
	}()
	log.Printf("Started node daemon, listening on %s", addr)
	<-ctx.Done()
	svr.GracefulStop()
	return nil
}

// runNodeRegistration keeps registering the node at the registry.
func (s *Service) runNodeRegistration(ctx context.Context) {
	delay := time.Second
	nodeAddress := net.JoinHostPort(s.myIP, strconv.Itoa(s.Config.Port))
	for {
		if _, err := s.nodeRegistry.RegisterNode(ctx, &local.RegisterNodeRequest{
			Name:        s.Config.NodeName,
			NodeAddress: nodeAddress,
		}); err != nil {
			s.log.Error().Err(err).Msg("Failed to register node")
			delay = util.Backoff(delay, 1.5, time.Second*15)
		} else {
			delay = time.Minute
		}
		select {
		case <-time.After(delay):
			// Continue
		case <-ctx.Done():
			// Context canceled
			return
		}
	}
}

// CreateFileView returns a view on the given file identified by the given URI.
func (s *Service) CreateFileView(ctx context.Context, req *local.CreateFileViewRequest) (*fs.CreateFileViewResponse, error) {
	// Fetch args
	localPath := req.GetLocalPath()

	log := s.log.With().
		Str("path", localPath).
		Bool("preview", req.GetPreview()).
		Logger()

	// Read file
	content, err := ioutil.ReadFile(localPath)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to read file")
		return nil, err
	}

	// Detect content type
	contentType := http.DetectContentType(content)

	return &fs.CreateFileViewResponse{
		Content:     content,
		ContentType: contentType,
	}, nil
}
