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

	"github.com/AljabrIO/koalja-operator/pkg/constants"
	"github.com/AljabrIO/koalja-operator/pkg/task"
	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"
)

// SnapshotServiceClient is a closable client interface.
type SnapshotServiceClient interface {
	task.SnapshotServiceClient
	Close() error
}

// CreateSnapshotServiceClient creates a client for the output FS service
// which the address is found in the environment
func CreateSnapshotServiceClient() (SnapshotServiceClient, error) {
	address := os.Getenv(constants.EnvSnapshotServiceAddress)
	if address == "" {
		return nil, fmt.Errorf("Environment variable '%s' not set", constants.EnvSnapshotServiceAddress)
	}

	// Create a connection
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	// Create a client
	c := task.NewSnapshotServiceClient(conn)

	return &snapshotServiceClient{c: c, conn: conn}, nil
}

type snapshotServiceClient struct {
	c    task.SnapshotServiceClient
	conn *grpc.ClientConn
}

// Next pulls the task agent for the next available snapshot.
func (c *snapshotServiceClient) Next(ctx context.Context, in *task.NextRequest, opts ...grpc.CallOption) (*task.NextResponse, error) {
	return c.c.Next(ctx, in, opts...)
}

// Acknowledge the processing of a snapshot
func (c *snapshotServiceClient) Ack(ctx context.Context, in *task.AckRequest, opts ...grpc.CallOption) (*empty.Empty, error) {
	return c.c.Ack(ctx, in, opts...)
}

// Close the connection
func (c *snapshotServiceClient) Close() error {
	return c.conn.Close()
}
