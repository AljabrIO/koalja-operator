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

	"github.com/AljabrIO/koalja-operator/pkg/constants"
	"github.com/AljabrIO/koalja-operator/pkg/tracking"
)

// StatisticsSinkClient is a closable client interface for StatisticsSinkClient.
type StatisticsSinkClient interface {
	tracking.StatisticsSinkClient
	CloseConnection() error
}

// CreateStatisticsSinkClient creates a client for the statistics sink configured in the environment.
func CreateStatisticsSinkClient() (StatisticsSinkClient, error) {
	addr := os.Getenv(constants.EnvStatisticsSinkAddress)
	if addr == "" {
		return nil, fmt.Errorf("Missing environment variable '%s'", constants.EnvStatisticsSinkAddress)
	}
	// Create a connection
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		return nil, maskAny(err)
	}

	// Create a client
	c := tracking.NewStatisticsSinkClient(conn)

	return &statisticsSinkClient{c: c, conn: conn}, nil
}

type statisticsSinkClient struct {
	c    tracking.StatisticsSinkClient
	conn *grpc.ClientConn
}

// Provide statistics of a link (called by the link)
func (c *statisticsSinkClient) PublishLinkStatistics(ctx context.Context, in *tracking.LinkStatistics, opts ...grpc.CallOption) (*empty.Empty, error) {
	return c.c.PublishLinkStatistics(ctx, in, opts...)
}

// Provide statistics of a task (called by the task)
func (c *statisticsSinkClient) PublishTaskStatistics(ctx context.Context, in *tracking.TaskStatistics, opts ...grpc.CallOption) (*empty.Empty, error) {
	return c.c.PublishTaskStatistics(ctx, in, opts...)
}

// Close the connection
func (c *statisticsSinkClient) CloseConnection() error {
	return c.conn.Close()
}
