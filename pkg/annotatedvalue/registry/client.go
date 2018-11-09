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

package registry

import (
	"context"
	fmt "fmt"
	"os"
	"time"

	"github.com/AljabrIO/koalja-operator/pkg/annotatedvalue"
	"github.com/AljabrIO/koalja-operator/pkg/constants"
	"github.com/golang/protobuf/ptypes/empty"
	grpc "google.golang.org/grpc"
)

// AnnotatedValueRegistryClient is a closable client interface.
type AnnotatedValueRegistryClient interface {
	annotatedvalue.AnnotatedValueRegistryClient
	Close() error
}

// CreateAnnotatedValueRegistryClient creates a client for the annotated value registry, which
// address is found in the environment
func CreateAnnotatedValueRegistryClient() (AnnotatedValueRegistryClient, error) {
	address := os.Getenv(constants.EnvAnnotatedValueRegistryAddress)
	if address == "" {
		return nil, fmt.Errorf("Environment variable '%s' not set", constants.EnvAnnotatedValueRegistryAddress)
	}

	// Create a connection
	conn, err := grpc.Dial(address, grpc.WithInsecure(), grpc.WithBackoffMaxDelay(time.Second*5))
	if err != nil {
		return nil, err
	}

	// Create a client
	c := annotatedvalue.NewAnnotatedValueRegistryClient(conn)

	return &annotatedValueRegistryClient{c: c, conn: conn}, nil
}

type annotatedValueRegistryClient struct {
	c    annotatedvalue.AnnotatedValueRegistryClient
	conn *grpc.ClientConn
}

// Record the given annotated value in the registry
func (c *annotatedValueRegistryClient) Record(ctx context.Context, in *annotatedvalue.AnnotatedValue, opts ...grpc.CallOption) (*empty.Empty, error) {
	return c.c.Record(ctx, in, opts...)
}

// GetByID returns the annotated value with given ID.
func (c *annotatedValueRegistryClient) GetByID(ctx context.Context, in *annotatedvalue.GetByIDRequest, opts ...grpc.CallOption) (*annotatedvalue.GetResponse, error) {
	return c.c.GetByID(ctx, in, opts...)
}

// GetByTaskAndData returns the annotated value with given task and data.
func (c *annotatedValueRegistryClient) GetByTaskAndData(ctx context.Context, in *annotatedvalue.GetByTaskAndDataRequest, opts ...grpc.CallOption) (*annotatedvalue.GetResponse, error) {
	return c.c.GetByTaskAndData(ctx, in, opts...)
}

// Close the connection
func (c *annotatedValueRegistryClient) Close() error {
	return c.conn.Close()
}
