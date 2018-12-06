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
	"google.golang.org/grpc"
)

// OutputFileSystemServiceClient is a closable client interface.
type OutputFileSystemServiceClient interface {
	task.OutputFileSystemServiceClient
	Close() error
}

// CreateOutputFileSystemServiceClient creates a client for the output FS service
// which the address is found in the environment
func CreateOutputFileSystemServiceClient() (OutputFileSystemServiceClient, error) {
	address := os.Getenv(constants.EnvOutputFileSystemServiceAddress)
	if address == "" {
		return nil, fmt.Errorf("Environment variable '%s' not set", constants.EnvOutputFileSystemServiceAddress)
	}

	// Create a connection
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	// Create a client
	c := task.NewOutputFileSystemServiceClient(conn)

	return &outputFileSystemServiceClient{c: c, conn: conn}, nil
}

type outputFileSystemServiceClient struct {
	c    task.OutputFileSystemServiceClient
	conn *grpc.ClientConn
}

// CreateFileURI creates a URI for the given file/dir
func (c *outputFileSystemServiceClient) CreateFileURI(ctx context.Context, in *task.CreateFileURIRequest, opts ...grpc.CallOption) (*task.CreateFileURIResponse, error) {
	return c.c.CreateFileURI(ctx, in, opts...)
}

// Close the connection
func (c *outputFileSystemServiceClient) Close() error {
	return c.conn.Close()
}
