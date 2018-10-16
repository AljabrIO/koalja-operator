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
	fmt "fmt"
	"os"

	"github.com/AljabrIO/koalja-operator/pkg/constants"
	"github.com/AljabrIO/koalja-operator/pkg/event"
	grpc "google.golang.org/grpc"
)

// CreateEventRegistryClient creates a client for the event registry, which
// address is found in the environment
func CreateEventRegistryClient() (*grpc.ClientConn, event.EventRegistryClient, error) {
	address := os.Getenv(constants.EnvEventRegistryAddress)
	if address == "" {
		return nil, nil, fmt.Errorf("Environment variable '%s' not set", constants.EnvEventRegistryAddress)
	}

	// Create a connection
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return nil, nil, err
	}

	// Create a client
	c := event.NewEventRegistryClient(conn)

	return conn, c, nil
}
