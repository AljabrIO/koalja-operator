/*
Copyright 2019 Aljabr Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package arangodb

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	driver "github.com/arangodb/go-driver"
	"github.com/arangodb/go-driver/http"
	"github.com/rs/zerolog"

	link "github.com/AljabrIO/koalja-operator/pkg/agent/link"
	"github.com/AljabrIO/koalja-operator/pkg/annotatedvalue"
)

// Config holds database configuration settings, used to connect
// to the ArangoDB database.
type Config struct {
	// Endpoint(s) URL of the database.
	Endpoints []string
	// Username used for authentication
	Username string
	// Password used for authentication
	Password string
	// Database name to store data into
	Database string
}

// dbBuilder is an in-memory implementation of an annotated value queue.
type dbBuilder struct {
	Config
	log zerolog.Logger
}

const (
	subscriptionTTL           = time.Minute
	maxAnnotatedValuesWaiting = 10 // Maximum number of value's we're allowed to queue (TODO make configurable)
)

const (
	// collection name template for link queue (args: namespace, pipeline)
	queueColNameTemplate = "lqueue_%s_%s"
	// collection name template for link subscriptions (args: namespace, pipeline)
	subscrColNameTemplate = "lsubscr_%s_%s"
)

// NewArangoDB initializes a new ArangoDB API builder
func NewArangoDB(log zerolog.Logger, cfg Config) link.APIBuilder {
	return &dbBuilder{
		log:    log,
		Config: cfg,
	}
}

// NewAnnotatedValuePublisher "builds" a new publisher
func (s *dbBuilder) NewAnnotatedValuePublisher(deps link.APIDependencies) (annotatedvalue.AnnotatedValuePublisherServer, error) {
	ctx := context.Background()
	db, err := s.openDatabase(ctx)
	if err != nil {
		return nil, maskAny(err)
	}
	col, err := s.ensureQueueCollection(ctx, db, deps)
	if err != nil {
		return nil, maskAny(err)
	}
	return &dbPub{
		log:        s.log,
		db:         db,
		queueCol:   col,
		uri:        deps.URI,
		registry:   deps.AnnotatedValueRegistry,
		statistics: deps.Statistics,
	}, nil
}

// NewAnnotatedValueSource "builds" a new source
func (s *dbBuilder) NewAnnotatedValueSource(deps link.APIDependencies) (annotatedvalue.AnnotatedValueSourceServer, error) {
	ctx := context.Background()
	db, err := s.openDatabase(ctx)
	if err != nil {
		return nil, maskAny(err)
	}
	queueCol, err := s.ensureQueueCollection(ctx, db, deps)
	if err != nil {
		return nil, maskAny(err)
	}
	subscrCol, err := s.ensureSubscriptionCollection(ctx, db, deps)
	if err != nil {
		return nil, maskAny(err)
	}
	return &dbSource{
		log:        s.log,
		db:         db,
		queueCol:   queueCol,
		subscrCol:  subscrCol,
		linkName:   deps.LinkName,
		uri:        deps.URI,
		statistics: deps.Statistics,
	}, nil
}

// openDatabase opens a database connection and creates a client
// for it.
func (s *dbBuilder) openDatabase(ctx context.Context) (driver.Database, error) {
	log := s.log.With().
		Strs("endpoints", s.Endpoints).
		Str("username", s.Username).
		Str("database", s.Database).
		Logger()

	// Check config
	if len(s.Endpoints) == 0 {
		return nil, maskAny(fmt.Errorf("Missing Endpoints"))
	}
	if len(s.Database) == 0 {
		return nil, maskAny(fmt.Errorf("Missing Database"))
	}

	// Create db connection
	conn, err := http.NewConnection(http.ConnectionConfig{
		Endpoints: s.Endpoints,
		TLSConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to connect to database")
		return nil, maskAny(err)
	}
	// Create db client
	clientCfg := driver.ClientConfig{
		Connection: conn,
	}
	if s.Username != "" {
		clientCfg.Authentication = driver.BasicAuthentication(s.Username, s.Password)
	}
	client, err := driver.NewClient(clientCfg)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create database client")
		return nil, maskAny(err)
	}
	// Ensure database exists
	db, err := client.Database(ctx, s.Database)
	if driver.IsNotFound(err) {
		// Database has to be created
		db, err = client.CreateDatabase(ctx, s.Database, nil)
		if err != nil {
			log.Error().Err(err).Msg("Failed to create database")
			return nil, maskAny(err)
		}
	} else if err != nil {
		log.Error().Err(err).Msg("Failed to lookup database")
		return nil, maskAny(err)
	}
	return db, nil
}

// ensureQueueCollection opens the queue collection collection in a given database,
// creating it if needed.
func (s *dbBuilder) ensureQueueCollection(ctx context.Context, db driver.Database, deps link.APIDependencies) (driver.Collection, error) {
	colName := fmt.Sprintf(queueColNameTemplate, deps.Namespace, deps.PipelineName)
	col, err := s.ensureCollection(ctx, db, colName, true)
	if err != nil {
		return nil, maskAny(err)
	}
	return col, nil
}

// ensureSubscriptionCollection opens the subscription collection collection in a given database,
// creating it if needed.
func (s *dbBuilder) ensureSubscriptionCollection(ctx context.Context, db driver.Database, deps link.APIDependencies) (driver.Collection, error) {
	colName := fmt.Sprintf(subscrColNameTemplate, deps.Namespace, deps.PipelineName)
	col, err := s.ensureCollection(ctx, db, colName, true)
	if err != nil {
		return nil, maskAny(err)
	}
	if _, _, err := col.EnsureHashIndex(ctx, []string{"client_id"}, &driver.EnsureHashIndexOptions{
		Unique: true,
		Sparse: false,
	}); err != nil {
		return nil, maskAny(err)
	}
	return col, nil
}

// ensureCollection opens a collection in a given database,
// creating it if needed.
func (s *dbBuilder) ensureCollection(ctx context.Context, db driver.Database, colName string, autoIncrement bool) (driver.Collection, error) {
	// Ensure collection exists
	col, err := db.Collection(ctx, colName)
	if driver.IsNotFound(err) {
		// Collection has to be created
		opts := &driver.CreateCollectionOptions{}
		if autoIncrement {
			opts.KeyOptions = &driver.CollectionKeyOptions{
				Type:      driver.KeyGeneratorAutoIncrement,
				Increment: 1,
				Offset:    1,
			}
		}
		col, err = db.CreateCollection(ctx, colName, opts)
		if err != nil {
			s.log.Error().Err(err).Msg("Failed to create collection")
			return nil, maskAny(err)
		}
	} else if err != nil {
		s.log.Error().Err(err).Msg("Failed to lookup collection")
		return nil, maskAny(err)
	}
	return col, nil
}
