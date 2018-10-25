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

package link

import (
	"context"
	fmt "fmt"
	"time"

	"github.com/AljabrIO/koalja-operator/pkg/event"
	"github.com/golang/protobuf/ptypes/empty"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// EventSourceClient is a closable client interface for EventSource.
type EventSourceClient interface {
	event.EventSourceClient
	CloseConnection() error
}

// CreateEventSourceClient creates a client for the given event source with the given address.
func CreateEventSourceClient(address string) (EventSourceClient, error) {
	// Create a connection
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return nil, maskAny(err)
	}

	// Create a client
	c := event.NewEventSourceClient(conn)

	return &eventSourceClient{c: c, conn: conn}, nil
}

type eventSourceClient struct {
	c    event.EventSourceClient
	conn *grpc.ClientConn
}

// Subscribe to events
func (c *eventSourceClient) Subscribe(ctx context.Context, in *event.SubscribeRequest, opts ...grpc.CallOption) (*event.SubscribeResponse, error) {
	return c.c.Subscribe(ctx, in, opts...)
}

// Ping keeps a subscription alive
func (c *eventSourceClient) Ping(ctx context.Context, in *event.PingRequest, opts ...grpc.CallOption) (*empty.Empty, error) {
	return c.c.Ping(ctx, in, opts...)
}

// Close a subscription
func (c *eventSourceClient) Close(ctx context.Context, in *event.CloseRequest, opts ...grpc.CallOption) (*empty.Empty, error) {
	return c.c.Close(ctx, in, opts...)
}

// Ask for the next event on a subscription
func (c *eventSourceClient) NextEvent(ctx context.Context, in *event.NextEventRequest, opts ...grpc.CallOption) (*event.NextEventResponse, error) {
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
	return c.c.NextEvent(ctx, in, opts...)
}

// Acknowledge the processing of an event
func (c *eventSourceClient) AckEvent(ctx context.Context, in *event.AckEventRequest, opts ...grpc.CallOption) (*empty.Empty, error) {
	return c.c.AckEvent(ctx, in, opts...)
}

// Close the connection
func (c *eventSourceClient) CloseConnection() error {
	return c.conn.Close()
}
