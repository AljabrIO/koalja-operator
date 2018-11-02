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

package client

import (
	"context"
	"fmt"
	"os"

	"github.com/golang/protobuf/ptypes/empty"
	grpc "google.golang.org/grpc"

	"github.com/AljabrIO/koalja-operator/pkg/agent/pipeline"
	"github.com/AljabrIO/koalja-operator/pkg/constants"
)

// AgentRegistryClient is a closable client interface for AgentRegistryClient.
type AgentRegistryClient interface {
	pipeline.AgentRegistryClient
	CloseConnection() error
}

// CreateAgentRegistryClient creates a client for the given agent registry with the given address.
func CreateAgentRegistryClient() (AgentRegistryClient, error) {
	addr := os.Getenv(constants.EnvAgentRegistryAddress)
	if addr == "" {
		return nil, fmt.Errorf("Missing environment variable '%s'", constants.EnvAgentRegistryAddress)
	}
	// Create a connection
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		return nil, maskAny(err)
	}

	// Create a client
	c := pipeline.NewAgentRegistryClient(conn)

	return &agentRegistryClient{c: c, conn: conn}, nil
}

type agentRegistryClient struct {
	c    pipeline.AgentRegistryClient
	conn *grpc.ClientConn
}

// Register an instance of a link agent
func (c *agentRegistryClient) RegisterLink(ctx context.Context, in *pipeline.RegisterLinkRequest, opts ...grpc.CallOption) (*empty.Empty, error) {
	return c.c.RegisterLink(ctx, in, opts...)
}

// Register an instance of a task agent
func (c *agentRegistryClient) RegisterTask(ctx context.Context, in *pipeline.RegisterTaskRequest, opts ...grpc.CallOption) (*empty.Empty, error) {
	return c.c.RegisterTask(ctx, in, opts...)
}

// Provide statistics of a link (called by the link)
func (c *agentRegistryClient) PublishLinkStatistics(ctx context.Context, in *pipeline.LinkStatistics, opts ...grpc.CallOption) (*empty.Empty, error) {
	return c.c.PublishLinkStatistics(ctx, in, opts...)
}

// Provide statistics of a task (called by the task)
func (c *agentRegistryClient) PublishTaskStatistics(ctx context.Context, in *pipeline.TaskStatistics, opts ...grpc.CallOption) (*empty.Empty, error) {
	return c.c.PublishTaskStatistics(ctx, in, opts...)
}

// Close the connection
func (c *agentRegistryClient) CloseConnection() error {
	return c.conn.Close()
}
