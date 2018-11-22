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

package local

import (
	"context"
	"sync"
	"time"

	empty "github.com/golang/protobuf/ptypes/empty"
	"github.com/rs/zerolog"
	grpc "google.golang.org/grpc"
)

type nodeRegistry struct {
	log   zerolog.Logger
	nodes map[string]*nodeEntry // name -> entry
	mutex sync.Mutex
}

var _ NodeRegistryServer = &nodeRegistry{}

type nodeEntry struct {
	Address   string
	ExpiresAt time.Time
	Client    NodeClient
}

const (
	nodeExpirationTimeout = time.Minute * 2
)

// newNodeRegistry creates a new registry for nodes.
func newNodeRegistry(log zerolog.Logger) *nodeRegistry {
	return &nodeRegistry{
		log:   log,
		nodes: make(map[string]*nodeEntry),
	}
}

// CreateFileView returns a view on the given file identified by the given URI.
func (r *nodeRegistry) RegisterNode(ctx context.Context, req *RegisterNodeRequest) (*empty.Empty, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	nodeName := req.GetName()
	nodeAddress := req.GetNodeAddress()
	log := r.log.With().
		Str("node", nodeName).
		Str("address", nodeAddress).
		Logger()

	if e, found := r.nodes[nodeName]; found {
		if e.Address != nodeAddress {
			e.Address = nodeAddress
			e.Client = nil
		}
		e.ExpiresAt = time.Now().Add(nodeExpirationTimeout)
		log.Debug().Msg("Updated existing node registration")
	} else {
		r.nodes[req.GetName()] = &nodeEntry{
			Address:   nodeAddress,
			ExpiresAt: time.Now().Add(nodeExpirationTimeout),
		}
		log.Info().Msg("Registered new node")
	}

	return &empty.Empty{}, nil
}

// GetNodeClient returns a client for the given node or nil of no such node is found.
func (r *nodeRegistry) GetNodeClient(ctx context.Context, nodeName string) (NodeClient, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	e, found := r.nodes[nodeName]
	if !found {
		return nil, nil
	}

	// Create client if needed
	if e.Client == nil {
		// Create a connection
		conn, err := grpc.Dial(e.Address, grpc.WithInsecure())
		if err != nil {
			return nil, err
		}
		e.Client = NewNodeClient(conn)
	}

	return e.Client, nil
}
