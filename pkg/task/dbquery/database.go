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

package dbquery

import (
	"context"
	"fmt"

	taskclient "github.com/AljabrIO/koalja-operator/pkg/task/client"
	"github.com/rs/zerolog"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Database provides the API implemented by various database types.
type Database interface {
	// Query the database provided in the given config until the given context
	// is canceled.
	Query(context.Context, Config, DatabaseConfig, QueryDependencies) error
}

// QueryDependencies holds services that are available for Database implementation
// during a Query call.
type QueryDependencies struct {
	// Logger
	Log zerolog.Logger
	// Client for FS service
	FileSystemClient taskclient.OutputFileSystemServiceClient
	// Scheme to use for FS URI's
	FileSystemScheme string
	// OutputReady is to be called by a database for publishing output notifications.
	// Returns: (annotatedValueID, error)
	OutputReady func(ctx context.Context, annotatedValueData, outputName string) (string, error)
	// Client for accessing Kubernetes resources
	KubernetesClient client.Client
	// Secret containing authentication info
	AuthenticationSecret *corev1.Secret
}

var (
	databases = make(map[DatabaseType]Database)
)

// RegisterDatabase registers a database implementation for given type.
// This function must be called during init phase.
func RegisterDatabase(dbType DatabaseType, db Database) {
	databases[dbType] = db
}

// GetDatabaseByType returns the implementation of Database
// for the given type.
// Returns an error if not found.
func GetDatabaseByType(dbType DatabaseType) (Database, error) {
	if db, found := databases[dbType]; found {
		return db, nil
	}
	return nil, maskAny(fmt.Errorf("Database with type '%s' not found", dbType))
}
