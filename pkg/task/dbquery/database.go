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
)

// Database provides the API implemented by various database types.
type Database interface {
	// Query the database provided in the given config until the given context
	// is canceled.
	Query(context.Context, Config, DatabaseConfig) error
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
