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

	"github.com/AljabrIO/koalja-operator/pkg/constants"
	"github.com/AljabrIO/koalja-operator/pkg/event"
	"github.com/golang/protobuf/ptypes/empty"
	grpc "google.golang.org/grpc"
)

// EventRegistryClient is a closable client interface.
type EventRegistryClient interface {
	event.EventRegistryClient
	Close() error
}

// CreateEventRegistryClient creates a client for the event registry, which
// address is found in the environment
func CreateEventRegistryClient() (EventRegistryClient, error) {
	address := os.Getenv(constants.EnvEventRegistryAddress)
	if address == "" {
		return nil, fmt.Errorf("Environment variable '%s' not set", constants.EnvEventRegistryAddress)
	}

	// Create a connection
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	// Create a client
	c := event.NewEventRegistryClient(conn)

	return &eventRegistryClient{c: c, conn: conn}, nil
}

type eventRegistryClient struct {
	c    event.EventRegistryClient
	conn *grpc.ClientConn
}

// Record the given event in the registry
func (c *eventRegistryClient) RecordEvent(ctx context.Context, in *event.Event, opts ...grpc.CallOption) (*empty.Empty, error) {
	return c.c.RecordEvent(ctx, in, opts...)
}

// GetEventByID returns the event with given ID.
func (c *eventRegistryClient) GetEventByID(ctx context.Context, in *event.GetEventByIDRequest, opts ...grpc.CallOption) (*event.GetEventResponse, error) {
	return c.c.GetEventByID(ctx, in, opts...)
}

// GetEventByID returns the event with given ID.
func (c *eventRegistryClient) GetEventByTaskAndData(ctx context.Context, in *event.GetEventByTaskAndDataRequest, opts ...grpc.CallOption) (*event.GetEventResponse, error) {
	return c.c.GetEventByTaskAndData(ctx, in, opts...)
}

// Close the connection
func (c *eventRegistryClient) Close() error {
	return c.conn.Close()
}
