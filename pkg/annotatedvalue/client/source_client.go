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
	fmt "fmt"
	"time"

	"github.com/AljabrIO/koalja-operator/pkg/annotatedvalue"
	"github.com/golang/protobuf/ptypes/empty"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// AnnotatedValueSourceClient is a closable client interface for AnnotatedValueSource.
type AnnotatedValueSourceClient interface {
	annotatedvalue.AnnotatedValueSourceClient
	CloseConnection() error
}

// NewAnnotatedValueSourceClient creates a client for the given annotated value source with the given address.
func NewAnnotatedValueSourceClient(address string) (AnnotatedValueSourceClient, error) {
	// Create a connection
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return nil, maskAny(err)
	}

	// Create a client
	c := annotatedvalue.NewAnnotatedValueSourceClient(conn)

	return &annotatedValueSourceClient{c: c, conn: conn}, nil
}

type annotatedValueSourceClient struct {
	c    annotatedvalue.AnnotatedValueSourceClient
	conn *grpc.ClientConn
}

// Subscribe to annotated values
func (c *annotatedValueSourceClient) Subscribe(ctx context.Context, in *annotatedvalue.SubscribeRequest, opts ...grpc.CallOption) (*annotatedvalue.SubscribeResponse, error) {
	return c.c.Subscribe(ctx, in, opts...)
}

// Ping keeps a subscription alive
func (c *annotatedValueSourceClient) Ping(ctx context.Context, in *annotatedvalue.PingRequest, opts ...grpc.CallOption) (*empty.Empty, error) {
	return c.c.Ping(ctx, in, opts...)
}

// Close a subscription
func (c *annotatedValueSourceClient) Close(ctx context.Context, in *annotatedvalue.CloseRequest, opts ...grpc.CallOption) (*empty.Empty, error) {
	return c.c.Close(ctx, in, opts...)
}

// Ask for the next annotated value on a subscription
func (c *annotatedValueSourceClient) Next(ctx context.Context, in *annotatedvalue.NextRequest, opts ...grpc.CallOption) (*annotatedvalue.NextResponse, error) {
	if deadline, ok := ctx.Deadline(); ok {
		ms := int64(time.Until(deadline).Seconds() * 1000.0 * 1.5)
		ctx = metadata.NewOutgoingContext(ctx, metadata.New(map[string]string{
			"x-envoy-upstream-rq-timeout-ms":         fmt.Sprintf("%d", ms),
			"x-envoy-upstream-rq-per-try-timeout-ms": fmt.Sprintf("%d", ms),
			"x-envoy-expected-rq-timeout-ms":         fmt.Sprintf("%d", ms),
		}))
		ctx = metadata.NewIncomingContext(ctx, metadata.New(map[string]string{
			"x-envoy-upstream-rq-timeout-ms":         fmt.Sprintf("%d", ms),
			"x-envoy-upstream-rq-per-try-timeout-ms": fmt.Sprintf("%d", ms),
			"x-envoy-expected-rq-timeout-ms":         fmt.Sprintf("%d", ms),
		}))
	}
	return c.c.Next(ctx, in, opts...)
}

// Acknowledge the processing of an annotated value
func (c *annotatedValueSourceClient) Ack(ctx context.Context, in *annotatedvalue.AckRequest, opts ...grpc.CallOption) (*empty.Empty, error) {
	return c.c.Ack(ctx, in, opts...)
}

// Close the connection
func (c *annotatedValueSourceClient) CloseConnection() error {
	return c.conn.Close()
}
