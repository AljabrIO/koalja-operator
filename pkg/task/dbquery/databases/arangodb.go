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
	"crypto/tls"
	"encoding/json"
	"time"

	driver "github.com/arangodb/go-driver"
	"github.com/arangodb/go-driver/http"
	"github.com/rs/zerolog"
	"github.com/vincent-petithory/dataurl"

	"github.com/AljabrIO/koalja-operator/pkg/task/dbquery"
)

type arangodb struct{}

func init() {
	dbquery.RegisterDatabase(dbquery.DatabaseTypeArangoDB, &arangodb{})
}

const (
	// If the query takes less than this amount of time, we wait
	// a bit, such that the we query at most once in this time period.
	minQueryTime = time.Second * 5
)

// Query the database provided in the given config until the given context
// is canceled.
func (a *arangodb) Query(ctx context.Context, cfg dbquery.Config, dbCfg dbquery.DatabaseConfig, deps dbquery.QueryDependencies) error {
	// Prepare logger
	log := deps.Log.With().
		Str("address", dbCfg.Address).
		Str("database", dbCfg.Database).
		Logger()

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
		annValID, err := deps.OutputReady(ctx, dataURI.String(), cfg.OutputName)
		if err != nil {
			log.Error().Err(err).Msg("Failed to notify output readiness")
			return maskAny(err)
		}
		log.Debug().Str("id", annValID).Msg("Published document")
		return nil
	}

	// Apply authentication (if needed)
	var auth driver.Authentication
	if s := deps.AuthenticationSecret; s != nil {
		username := string(s.Data["username"])
		password := string(s.Data["password"])
		auth = driver.BasicAuthentication(username, password)
		log.Debug().Str("username", username).Msg("Using authentication")
	}

	// Connect to the database an keep running the query
	lastKey := ""
	if err := a.connectAndQuery(ctx, log, cfg, &lastKey, dbCfg, auth, publish); err != nil {
		log.Debug().Err(err).Msg("connectAndQuery failed")
		return maskAny(err)
	}

	return nil
}

// connectAndQuery opens the database and keeps running the query until the given context is canceled.
func (a *arangodb) connectAndQuery(ctx context.Context, log zerolog.Logger, cfg dbquery.Config, lastKey *string, dbCfg dbquery.DatabaseConfig, auth driver.Authentication, publish func(ctx context.Context, doc interface{}) error) error {
	// Create an HTTP connection to the database
	conn, err := http.NewConnection(http.ConnectionConfig{
		Endpoints: []string{dbCfg.Address},
		TLSConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to create HTTP connection to ArangoDB database")
		return maskAny(err)
	}

	// Create a client
	c, err := driver.NewClient(driver.ClientConfig{
		Connection:     conn,
		Authentication: auth,
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

	// Keep running the query
	for {
		start := time.Now()
		if err := a.queryOnce(ctx, log, cfg, lastKey, db, publish); err != nil {
			log.Debug().Err(err).Msg("queryOnce failed")
			return maskAny(err)
		}

		// If the query quicklyterminated, we should wait a while before trying again
		// so we do not consume too much resources
		if duration := time.Since(start); duration < minQueryTime {
			select {
			case <-time.After(minQueryTime - duration):
				// Continue
			case <-ctx.Done():
				// Context canceled
				return ctx.Err()
			}
		}
	}
}

// connectAndQuery opens the database and keeps running the query until the given context is canceled.
func (a *arangodb) queryOnce(ctx context.Context, log zerolog.Logger, cfg dbquery.Config, lastKey *string, db driver.Database, publish func(ctx context.Context, doc interface{}) error) error {
	// Run query
	qctx := driver.WithQueryStream(ctx, true)
	cursor, err := db.Query(qctx, cfg.Query, map[string]interface{}{
		"lastKey": *lastKey,
	})
	if err != nil {
		log.Error().Err(err).
			Str("query", cfg.Query).
			Str("lastKey", *lastKey).
			Msg("Failed to execute query")
		return maskAny(err)
	}

	// Read documents
	for {
		// Read next document
		var doc interface{}
		if meta, err := cursor.ReadDocument(qctx, &doc); driver.IsNoMoreDocuments(err) {
			// There are no more documents
			return nil
		} else if err != nil {
			// Some other error occurred
			log.Error().Err(err).Msg("ReadDocument failed")
		} else {
			// We have a document, publish it
			if err := publish(ctx, doc); err != nil {
				log.Error().Err(err).Msg("Failed to publish document")
				return maskAny(err)
			}
			// Publication succeeded
			*lastKey = meta.Key
		}
	}

	//return nil
}
