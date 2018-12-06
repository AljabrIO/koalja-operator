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

// OutputReadyNotifierClient is a closable client interface.
type OutputReadyNotifierClient interface {
	task.OutputReadyNotifierClient
	Close() error
}

// CreateOutputReadyNotifierClient creates a client for the output ready notifier
// which the address is found in the environment
func CreateOutputReadyNotifierClient() (OutputReadyNotifierClient, error) {
	address := os.Getenv(constants.EnvOutputReadyNotifierAddress)
	if address == "" {
		return nil, fmt.Errorf("Environment variable '%s' not set", constants.EnvOutputReadyNotifierAddress)
	}

	// Create a connection
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	// Create a client
	c := task.NewOutputReadyNotifierClient(conn)

	return &outputReadyNotifierClient{c: c, conn: conn}, nil
}

type outputReadyNotifierClient struct {
	c    task.OutputReadyNotifierClient
	conn *grpc.ClientConn
}

func (c *outputReadyNotifierClient) OutputReady(ctx context.Context, in *task.OutputReadyRequest, opts ...grpc.CallOption) (*task.OutputReadyResponse, error) {
	return c.c.OutputReady(ctx, in, opts...)
}

// Close the connection
func (c *outputReadyNotifierClient) Close() error {
	return c.conn.Close()
}
