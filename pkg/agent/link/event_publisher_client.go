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

	"github.com/AljabrIO/koalja-operator/pkg/event"
	grpc "google.golang.org/grpc"
)

// EventPublisherClient is a closable client interface for EventPublisher.
type EventPublisherClient interface {
	event.EventPublisherClient
	CloseConnection() error
}

// CreateEventPublisherClient creates a client for the given event source with the given address.
func CreateEventPublisherClient(address string) (EventPublisherClient, error) {
	// Create a connection
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	// Create a client
	c := event.NewEventPublisherClient(conn)

	return &eventPublisherClient{c: c, conn: conn}, nil
}

type eventPublisherClient struct {
	c    event.EventPublisherClient
	conn *grpc.ClientConn
}

// Subscribe to events
func (c *eventPublisherClient) Publish(ctx context.Context, in *event.PublishRequest, opts ...grpc.CallOption) (*event.PublishResponse, error) {
	return c.c.Publish(ctx, in, opts...)
}

// Close the connection
func (c *eventPublisherClient) CloseConnection() error {
	return c.conn.Close()
}
