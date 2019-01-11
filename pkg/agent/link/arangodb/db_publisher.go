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
	"sync/atomic"

	"github.com/AljabrIO/koalja-operator/pkg/annotatedvalue"
	avclient "github.com/AljabrIO/koalja-operator/pkg/annotatedvalue/client"
	"github.com/AljabrIO/koalja-operator/pkg/tracking"
	driver "github.com/arangodb/go-driver"
	"github.com/rs/zerolog"
)

// dbPub is an ArangoDB implementation of an annotated value queue publisher.
type dbPub struct {
	log        zerolog.Logger
	db         driver.Database
	queueCol   driver.Collection
	registry   avclient.AnnotatedValueRegistryClient
	linkName   string
	uri        string
	statistics *tracking.LinkStatistics
}

// Run this server until the given context is canceled.
func (s *dbPub) Run(ctx context.Context) error {
	<-ctx.Done()
	return nil
}

// CanPublish returns true if publishing annotated values is allowed.
func (s *dbPub) CanPublish(ctx context.Context, req *annotatedvalue.CanPublishRequest) (*annotatedvalue.CanPublishResponse, error) {
	return &annotatedvalue.CanPublishResponse{
		Allowed: atomic.LoadInt64(&s.statistics.AnnotatedValuesWaiting) < maxAnnotatedValuesWaiting,
	}, nil
}

// Publish an annotated value
func (s *dbPub) Publish(ctx context.Context, req *annotatedvalue.PublishRequest) (*annotatedvalue.PublishResponse, error) {
	s.log.Debug().Interface("annotatedvalue", req.AnnotatedValue).Msg("Publish request")
	// Try to record annotated value in registry
	av := *req.GetAnnotatedValue()
	av.Link = s.uri
	if _, err := s.registry.Record(ctx, &av); err != nil {
		return nil, maskAny(err)
	}

	// Add to DB
	doc := avDocument{
		Value:          &av,
		LinkName:       s.linkName,
		SubscriptionID: 0,
	}
	if _, err := s.queueCol.CreateDocument(ctx, doc); err != nil {
		return nil, arangoErrorToGRPC(err)
	}

	// Update statistic
	if unassigned, err := collectUnassignedQueueStats(ctx, s.linkName, s.queueCol); err == nil {
		atomic.StoreInt64(&s.statistics.AnnotatedValuesWaiting, unassigned)
	}

	// We're done
	return &annotatedvalue.PublishResponse{}, nil
}
