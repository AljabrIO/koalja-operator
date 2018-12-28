/*
Copyright 2018 Aljabr Inc.

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
	"strings"

	driver "github.com/arangodb/go-driver"
	"github.com/arangodb/go-driver/http"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/AljabrIO/koalja-operator/pkg/annotatedvalue"
	"github.com/AljabrIO/koalja-operator/pkg/annotatedvalue/registry"
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

// dbBuilder is an ArangoDB database implementation of an annotatedvalue registry builder.
type dbBuilder struct {
	Config
	log zerolog.Logger
}

const (
	// collection name template for annotated values
	avColNameTemplate = "av_%s"
)

// NewArangoDB initializes a new db struct.
func NewArangoDB(log zerolog.Logger, cfg Config) registry.APIBuilder {
	return &dbBuilder{
		log:    log,
		Config: cfg,
	}
}

// NewRegistry "builds" a new registry
func (s *dbBuilder) NewRegistry(deps registry.APIDependencies) (annotatedvalue.AnnotatedValueRegistryServer, error) {
	colName := fmt.Sprintf(avColNameTemplate, deps.Namespace)
	log := s.log.With().
		Strs("endpoints", s.Endpoints).
		Str("username", s.Username).
		Str("database", s.Database).
		Str("collection", colName).
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
	ctx := context.Background()
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
	// Ensure collection exists
	col, err := db.Collection(ctx, colName)
	if driver.IsNotFound(err) {
		// Collection has to be created
		col, err = db.CreateCollection(ctx, colName, nil)
		if err != nil {
			log.Error().Err(err).Msg("Failed to create collection")
			return nil, maskAny(err)
		}
	} else if err != nil {
		log.Error().Err(err).Msg("Failed to lookup collection")
		return nil, maskAny(err)
	}

	return &dbReg{
		log:    log,
		client: client,
		db:     db,
		col:    col,
	}, nil
}

// db is an ArangoDB database implementation of an annotatedvalue registry.
type dbReg struct {
	log    zerolog.Logger
	client driver.Client
	db     driver.Database
	col    driver.Collection
}

// Record the given annotated value in the registry
func (s *dbReg) Record(ctx context.Context, e *annotatedvalue.AnnotatedValue) (*empty.Empty, error) {
	s.log.Debug().Str("annotatedvalue", e.GetID()).Msg("Record request")

	// Insert annotated value into collection
	if _, err := s.col.CreateDocument(ctx, e); err != nil {
		s.log.Debug().Err(err).Msg("Failed to create AnnotatedValue document")
		return nil, arangoErrorToGRPC(err)
	}

	return &empty.Empty{}, nil
}

// GetByID returns the annotated value with given ID.
func (s *dbReg) GetByID(ctx context.Context, req *annotatedvalue.GetByIDRequest) (*annotatedvalue.GetOneResponse, error) {
	s.log.Debug().Str("annotatedvalue", req.GetID()).Msg("GetByID request")

	q := fmt.Sprintf("FOR x in %s FILTER x.id==@id LIMIT 1 RETURN x", s.col.Name())
	cursor, err := s.db.Query(ctx, q, map[string]interface{}{
		"id": req.GetID(),
	})
	if err != nil {
		s.log.Debug().Err(err).Msg("Failed to query for ID")
		return nil, arangoErrorToGRPC(err)
	}

	// Fetch first document (if any)
	var result annotatedvalue.AnnotatedValue
	if _, err := cursor.ReadDocument(ctx, &result); driver.IsNoMoreDocuments(err) {
		// No such ID found
		return nil, status.Error(codes.NotFound, fmt.Sprintf("AnnotatedValue '%s' not found", req.GetID()))
	} else if err != nil {
		s.log.Debug().Err(err).Msg("Failed to read document")
		return nil, arangoErrorToGRPC(err)
	}

	return &annotatedvalue.GetOneResponse{AnnotatedValue: &result}, nil
}

// GetByTaskAndData returns the annotated value with given task and data.
func (s *dbReg) GetByTaskAndData(ctx context.Context, req *annotatedvalue.GetByTaskAndDataRequest) (*annotatedvalue.GetOneResponse, error) {
	s.log.Debug().
		Str("task", req.GetSourceTask()).
		Str("data", req.GetData()).
		Msg("GetByTaskAndData request")

	q := fmt.Sprintf("FOR x in %s FILTER x.source_task==@sourceTask AND data==@data LIMIT 1 RETURN x", s.col.Name())
	cursor, err := s.db.Query(ctx, q, map[string]interface{}{
		"sourceTask": req.GetSourceTask(),
		"data":       req.GetData(),
	})
	if err != nil {
		s.log.Debug().Err(err).Msg("Failed to query for task and data")
		return nil, arangoErrorToGRPC(err)
	}

	// Fetch first document (if any)
	var result annotatedvalue.AnnotatedValue
	if _, err := cursor.ReadDocument(ctx, &result); driver.IsNoMoreDocuments(err) {
		// No such document found
		return nil, status.Error(codes.NotFound, "No such annotated value")
	} else if err != nil {
		s.log.Debug().Err(err).Msg("Failed to read document")
		return nil, arangoErrorToGRPC(err)
	}

	return &annotatedvalue.GetOneResponse{AnnotatedValue: &result}, nil
}

// Get returns the annotated values that match the given criteria.
func (s *dbReg) Get(ctx context.Context, req *annotatedvalue.GetRequest) (*annotatedvalue.GetResponse, error) {
	s.log.Debug().
		Msg("Get request")

	var filters []string
	bindVars := make(map[string]interface{})
	if len(req.GetIDs()) > 0 {
		filters = append(filters, fmt.Sprintf("x.id IN @ids"))
		bindVars["ids"] = req.GetIDs()
	}
	if len(req.GetSourceTasks()) > 0 {
		filters = append(filters, fmt.Sprintf("x.source_task IN @stasks"))
		bindVars["stasks"] = req.GetSourceTasks()
	}
	if len(req.GetData()) > 0 {
		filters = append(filters, fmt.Sprintf("x.data IN @data"))
		bindVars["data"] = req.GetData()
	}
	var filter string
	if len(filters) > 0 {
		filter = "FILTER " + strings.Join(filters, " AND ")
	}

	q := fmt.Sprintf("FOR x in %s %s RETURN x", s.col.Name(), filter)
	cursor, err := s.db.Query(ctx, q, bindVars)
	if err != nil {
		s.log.Debug().Err(err).Msg("Failed to query for request")
		return nil, arangoErrorToGRPC(err)
	}

	// Fetch documents
	resp := &annotatedvalue.GetResponse{}
	for {
		doc := &annotatedvalue.AnnotatedValue{}
		if _, err := cursor.ReadDocument(ctx, doc); driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			s.log.Debug().Err(err).Msg("Failed to read document")
			return nil, arangoErrorToGRPC(err)
		}
		if doc.IsMatch(req) {
			resp.AnnotatedValues = append(resp.AnnotatedValues, doc)
		}
	}

	return resp, nil
}
