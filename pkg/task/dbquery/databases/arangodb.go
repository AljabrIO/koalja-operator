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

package databases

import (
	"context"
	"encoding/json"

	driver "github.com/arangodb/go-driver"
	"github.com/arangodb/go-driver/http"
	"github.com/vincent-petithory/dataurl"

	"github.com/AljabrIO/koalja-operator/pkg/task"
	"github.com/AljabrIO/koalja-operator/pkg/task/dbquery"
)

type arangodb struct{}

func init() {
	dbquery.RegisterDatabase(dbquery.DatabaseTypeArangoDB, &arangodb{})
}

// Query the database provided in the given config until the given context
// is canceled.
func (a *arangodb) Query(ctx context.Context, cfg dbquery.Config, dbCfg dbquery.DatabaseConfig, deps dbquery.QueryDependencies) error {
	// Prepare logger
	log := deps.Log.With().
		Str("address", dbCfg.Address).
		Str("database", dbCfg.Database).
		Logger()

	// Create an HTTP connection to the database
	conn, err := http.NewConnection(http.ConnectionConfig{
		Endpoints: []string{dbCfg.Address},
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to create HTTP connection to ArangoDB database")
		return maskAny(err)
	}

	// Create a client
	c, err := driver.NewClient(driver.ClientConfig{
		Connection: conn,
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to create ArangoDB client")
		return maskAny(err)
	}

	// Open database
	db, err := c.Database(ctx, dbCfg.Database)
	if err != nil {
		log.Error().Err(err).Msg("Failed to open ArangoDB database")
		return maskAny(err)
	}

	// Run query
	qctx := driver.WithQueryStream(ctx, true)
	cursor, err := db.Query(qctx, cfg.Query, nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to execute query")
		return maskAny(err)
	}

	// Prepare publish function
	publish := func(ctx context.Context, doc interface{}) error {
		// Encode document as JSON
		encoded, err := json.Marshal(doc)
		if err != nil {
			log.Error().Err(err).Msg("Failed to marshal document into JSON")
			return maskAny(err)
		}
		// Prepare data URI
		dataURI := dataurl.New(encoded, "text/json")
		resp, err := deps.OutputReadyNotifierClient.OutputReady(ctx, &task.OutputReadyRequest{
			AnnotatedValueData: dataURI.String(),
			OutputName:         cfg.OutputName,
		})
		if err != nil {
			log.Error().Err(err).Msg("Failed to notify output readiness")
			return maskAny(err)
		}
		log.Debug().Str("id", resp.GetAnnotatedValueID()).Msg("Published document")
		return nil
	}

	// Read documents
	for {
		// Read next document
		var doc interface{}
		if _, err := cursor.ReadDocument(qctx, &doc); driver.IsNoMoreDocuments(err) {
			// There are no more documents
			return nil
		} else if err != nil {
			// Some other error occurred
			log.Error().Err(err).Msg("ReadDocument failed")
		} else {
			// We have a document, publish it
			if err := publish(ctx, doc); err != nil {
				log.Error().Err(err).Msg("Failed to publish document")
			}
		}
	}

	//return nil
}
