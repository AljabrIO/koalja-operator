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

package task

import (
	"context"

	grpc "google.golang.org/grpc"

	"github.com/AljabrIO/koalja-operator/pkg/annotatedvalue"
)

// AnnotatedValuePublisherClient is a closable client interface for AnnotatedValuePublisher.
type AnnotatedValuePublisherClient interface {
	annotatedvalue.AnnotatedValuePublisherClient
	CloseConnection() error
}

// CreateAnnotatedValuePublisherClient creates a client for the given annotated value source with the given address.
func CreateAnnotatedValuePublisherClient(address string) (AnnotatedValuePublisherClient, error) {
	// Create a connection
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return nil, maskAny(err)
	}

	// Create a client
	c := annotatedvalue.NewAnnotatedValuePublisherClient(conn)

	return &annotatedValuePublisherClient{c: c, conn: conn}, nil
}

type annotatedValuePublisherClient struct {
	c    annotatedvalue.AnnotatedValuePublisherClient
	conn *grpc.ClientConn
}

// Publish an annotated value
func (c *annotatedValuePublisherClient) Publish(ctx context.Context, in *annotatedvalue.PublishRequest, opts ...grpc.CallOption) (*annotatedvalue.PublishResponse, error) {
	return c.c.Publish(ctx, in, opts...)
}

// Close the connection
func (c *annotatedValuePublisherClient) CloseConnection() error {
	return c.conn.Close()
}
