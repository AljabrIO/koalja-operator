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
	"time"

	driver "github.com/arangodb/go-driver"
	ptypes "github.com/gogo/protobuf/types"
	google_protobuf1 "github.com/golang/protobuf/ptypes/empty"
	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/AljabrIO/koalja-operator/pkg/annotatedvalue"
	"github.com/AljabrIO/koalja-operator/pkg/tracking"
)

// dbSource is an ArangoDB implementation of an annotated value queue source.
type dbSource struct {
	log        zerolog.Logger
	db         driver.Database
	queueCol   driver.Collection
	subscrCol  driver.Collection
	linkName   string
	uri        string
	statistics *tracking.LinkStatistics
}

const (
	nextRetryPeriod = time.Millisecond * 100
)

// Run this server until the given context is canceled.
func (s *dbSource) Run(ctx context.Context) error {
	if err := s.runSubscriptionGC(ctx); err != nil {
		return maskAny(err)
	}
	return nil
}

// runSubscriptionGC garbage collects expired subscriptions.
func (s *dbSource) runSubscriptionGC(ctx context.Context) error {
	for {
		// Collect expired subscriptions
		list, err := getExpiredSubscriptions(ctx, s.linkName, s.subscrCol)
		if err != nil {
			s.log.Warn().Err(err).Msg("Failed to get expired subscriptions")
		} else {
			for _, subscr := range list {
				log := s.log.With().Int64("subscription", subscr.GetID()).Logger()
				// Un-assign queue values
				if err := subscr.UnassignQueueValues(ctx, s.queueCol); err != nil {
					log.Warn().Err(err).Msg("Failed to unassign annotated values from expired subscription")
					continue
				}
				// Remove subscription
				if err := subscr.Remove(ctx, s.subscrCol); err != nil && !driver.IsNotFound(err) {
					log.Warn().Err(err).Msg("Failed to remove expired subscription")
					continue
				}
				log.Debug().Msg("Expired subscription")
			}
		}

		// Wait a bit
		select {
		case <-time.After(time.Second * 10):
			// Continue
		case <-ctx.Done():
			// Context canceled
			return ctx.Err()
		}
	}
}

// Subscribe to annotated values
func (s *dbSource) Subscribe(ctx context.Context, req *annotatedvalue.SubscribeRequest) (*annotatedvalue.SubscribeResponse, error) {
	s.log.Debug().
		Str("clientID", req.GetClientID()).
		Msg("Subscribe request")

	// Find existing subscription
	subscr, err := findSubscriptionByClientID(ctx, req.GetClientID(), s.subscrCol)
	if err != nil {
		return nil, arangoErrorToGRPC(err)
	} else if subscr != nil {
		// We found an existing subscription for this client id.
		// Renew expiration
		if err := subscr.RenewExpiresAt(ctx, s.subscrCol); err != nil {
			return nil, arangoErrorToGRPC(err)
		}
		return &annotatedvalue.SubscribeResponse{
			Subscription: subscr.AsPB(),
			TTL:          ptypes.DurationProto(subscriptionTTL),
		}, nil
	}

	// Subscription now found,
	// add new subscription
	subscr = &subscription{
		ClientID:  req.GetClientID(),
		LinkName:  s.linkName,
		ExpiresAt: time.Now().Add(subscriptionTTL),
	}
	meta, err := s.subscrCol.CreateDocument(ctx, subscr)
	if err != nil {
		return nil, arangoErrorToGRPC(err)
	}
	subscr.ID = meta.Key
	id := subscr.GetID()

	s.log.Debug().
		Str("clientID", req.ClientID).
		Int64("id", id).
		Msg("Subscribe response")
	return &annotatedvalue.SubscribeResponse{
		Subscription: &annotatedvalue.Subscription{
			ID: id,
		},
	}, nil
}

// Ping keeps a subscription alive
func (s *dbSource) Ping(ctx context.Context, req *annotatedvalue.PingRequest) (*google_protobuf1.Empty, error) {
	id := req.GetSubscription().GetID()
	s.log.Debug().Int64("id", id).Msg("Ping request")

	// Load existing subscription
	subscr, err := getSubscriptionByID(ctx, id, s.subscrCol)
	if driver.IsNotFound(err) {
		return nil, status.Errorf(codes.NotFound, "Subscription %d not found", id)
	} else if err != nil {
		return nil, arangoErrorToGRPC(err)
	}
	// Renew expiration
	if err := subscr.RenewExpiresAt(ctx, s.subscrCol); err != nil {
		return nil, arangoErrorToGRPC(err)
	}
	return &google_protobuf1.Empty{}, nil
}

// Close a subscription
func (s *dbSource) Close(ctx context.Context, req *annotatedvalue.CloseRequest) (*google_protobuf1.Empty, error) {
	id := req.GetSubscription().GetID()
	s.log.Debug().Int64("id", id).Msg("Close request")

	// Load existing subscription
	subscr, err := getSubscriptionByID(ctx, id, s.subscrCol)
	if driver.IsNotFound(err) {
		return nil, status.Errorf(codes.NotFound, "Subscription %d not found", id)
	} else if err != nil {
		return nil, arangoErrorToGRPC(err)
	}
	// Un-assign queue values
	if err := subscr.UnassignQueueValues(ctx, s.queueCol); err != nil {
		return nil, arangoErrorToGRPC(err)
	}
	// Remove subscription
	if err := subscr.Remove(ctx, s.subscrCol); err != nil {
		return nil, arangoErrorToGRPC(err)
	}
	return &google_protobuf1.Empty{}, nil
}

// Ask for the next annotated value on a subscription
func (s *dbSource) Next(ctx context.Context, req *annotatedvalue.NextRequest) (*annotatedvalue.NextResponse, error) {
	id := req.GetSubscription().GetID()
	waitTimeout, err := ptypes.DurationFromProto(req.GetWaitTimeout())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid wait timeout %v: %s", req.GetWaitTimeout(), err)
	}
	s.log.Debug().
		Dur("timeout", waitTimeout).
		Int64("subscription", id).
		Msg("Next request")

	endTime := time.Now().Add(waitTimeout)
	for {
		// Load existing subscription
		subscr, err := getSubscriptionByID(ctx, id, s.subscrCol)
		if driver.IsNotFound(err) {
			return nil, status.Errorf(codes.NotFound, "Subscription %d not found", id)
		} else if err != nil {
			return nil, arangoErrorToGRPC(err)
		}
		// Renew expiration
		if err := subscr.RenewExpiresAt(ctx, s.subscrCol); err != nil {
			return nil, arangoErrorToGRPC(err)
		}

		// Assign first non-assigned document to this subscription
		avdoc, err := subscr.AssignFirstValue(ctx, s.queueCol)
		if err != nil {
			return nil, arangoErrorToGRPC(err)
		}
		if avdoc != nil {
			// Found first document
			atomic.AddInt64(&s.statistics.AnnotatedValuesInProgress, 1)
			if unassigned, err := collectUnassignedQueueStats(ctx, s.linkName, s.queueCol); err == nil {
				atomic.StoreInt64(&s.statistics.AnnotatedValuesWaiting, unassigned)
			}
			return &annotatedvalue.NextResponse{
				AnnotatedValue:      avdoc.Value,
				NoAnnotatedValueYet: false,
			}, nil
		}

		// Wait a bit
		delay := time.Until(endTime)
		if delay <= 0 {
			// No annotated value in time
			return &annotatedvalue.NextResponse{
				NoAnnotatedValueYet: true,
			}, nil
		}
		if delay > nextRetryPeriod {
			delay = nextRetryPeriod
		}
		select {
		case <-time.After(delay):
			// Retry
		case <-ctx.Done():
			// Context canceled
			return nil, ctx.Err()
		}
	}
}

// Acknowledge the processing of an annotated value
func (s *dbSource) Ack(ctx context.Context, req *annotatedvalue.AckRequest) (*google_protobuf1.Empty, error) {
	avid := req.GetAnnotatedValueID()
	id := req.GetSubscription().GetID()
	s.log.Debug().
		Str("annotatedvalue", req.GetAnnotatedValueID()).
		Int64("subscription", id).
		Msg("Ack request")

	// Load existing subscription
	subscr, err := getSubscriptionByID(ctx, id, s.subscrCol)
	if driver.IsNotFound(err) {
		return nil, status.Errorf(codes.NotFound, "Subscription %d not found", id)
	} else if err != nil {
		return nil, arangoErrorToGRPC(err)
	}
	// Renew expiration
	if err := subscr.RenewExpiresAt(ctx, s.subscrCol); err != nil {
		return nil, arangoErrorToGRPC(err)
	}

	// Lookup annotated value
	avdoc, err := subscr.FindAnnotatedValueByID(ctx, avid, s.queueCol)
	if err != nil {
		return nil, arangoErrorToGRPC(err)
	}
	if avdoc == nil {
		return nil, status.Errorf(codes.NotFound, "Subscription %d does not have inflight annotated value with ID '%s'", id, avid)
	}

	// Found inflight annotated value; Remove it.
	if err := avdoc.Remove(ctx, s.queueCol); err != nil {
		return nil, arangoErrorToGRPC(err)
	}

	// Update statistics
	atomic.AddInt64(&s.statistics.AnnotatedValuesInProgress, -1)
	atomic.AddInt64(&s.statistics.AnnotatedValuesAcknowledged, 1)
	return &google_protobuf1.Empty{}, nil
}
