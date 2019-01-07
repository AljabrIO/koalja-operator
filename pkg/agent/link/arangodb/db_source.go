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
	"fmt"
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
	uri        string
	statistics *tracking.LinkStatistics
}

// Subscribe to annotated values
func (s *dbSource) Subscribe(ctx context.Context, req *annotatedvalue.SubscribeRequest) (*annotatedvalue.SubscribeResponse, error) {
	s.log.Debug().Str("clientID", req.ClientID).Msg("Subscribe request")

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
	s.lastSubscriptionID++
	id := s.lastSubscriptionID
	s.subscriptions[id] = &subscription{
		id:        id,
		clientID:  req.ClientID,
		expiresAt: time.Now().Add(subscriptionTTL),
	}

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
	s.log.Debug().Int64("id", id).Msg("Ping request")

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

	err := func() error {
		s.subscriptionsMutex.Lock()
		defer s.subscriptionsMutex.Unlock()

		// Lookup subscription
		sub, found := s.subscriptions[id]
		if !found {
			// Subscription not found
			return fmt.Errorf("Subscription %d not found", id)
		}

		// Renew subscription
		sub.RenewExpiresAt()
		return nil
	}()
	if err != nil {
		// Subscription not found
		return nil, maskAny(err)
	}

	// No annotated value inflight, get one out of the queue(s)
	waitTimeout, err := ptypes.DurationFromProto(req.GetWaitTimeout())
	if err != nil {
		return nil, maskAny(err)
	}
	setInflight := func(av *annotatedvalue.AnnotatedValue) (*annotatedvalue.NextResponse, error) {
		s.subscriptionsMutex.Lock()
		defer s.subscriptionsMutex.Unlock()

		// Lookup subscription
		if sub, found := s.subscriptions[id]; !found {
			// Subscription not found
			return nil, fmt.Errorf("Subscription %d no longer found", id)
		} else {
			sub.RenewExpiresAt()
			sub.inflight = append(sub.inflight, av)
			atomic.AddInt64(&s.statistics.AnnotatedValuesInProgress, 1)
			atomic.AddInt64(&s.statistics.AnnotatedValuesWaiting, -1)
			return &annotatedvalue.NextResponse{
				AnnotatedValue:      av,
				NoAnnotatedValueYet: false,
			}, nil
		}
	}
	select {
	case e := <-s.retryQueue:
		// Found annotated value in retry queue
		return setInflight(e)
	case e := <-s.queue:
		// Found annotated value in normal queue
		return setInflight(e)
	case <-time.After(waitTimeout):
		// No annotated value in time
		return &annotatedvalue.NextResponse{
			NoAnnotatedValueYet: true,
		}, nil
	case <-ctx.Done():
		return nil, maskAny(ctx.Err())
	}
}

// Acknowledge the processing of an annotated value
func (s *dbSource) Ack(ctx context.Context, req *annotatedvalue.AckRequest) (*google_protobuf1.Empty, error) {
	s.log.Debug().Str("annotatedvalue", req.GetAnnotatedValueID()).Msg("Ack request")
	s.subscriptionsMutex.Lock()
	defer s.subscriptionsMutex.Unlock()

	// Lookup subscription
	id := req.GetSubscription().GetID()
	if sub, found := s.subscriptions[id]; !found {
		// Subscription not found
		return nil, fmt.Errorf("Subscription %d not found", id)
	} else {
		sub.RenewExpiresAt()
		for i, av := range sub.inflight {
			if av.GetID() == req.GetAnnotatedValueID() {
				// Found inflight annotated value; Remove it.
				sub.inflight = append(sub.inflight[:i], sub.inflight[i+1:]...)
				// Update statistics
				atomic.AddInt64(&s.statistics.AnnotatedValuesInProgress, -1)
				atomic.AddInt64(&s.statistics.AnnotatedValuesAcknowledged, 1)
				return &google_protobuf1.Empty{}, nil
			}
		}
		return nil, fmt.Errorf("Subscription %d does not have inflight annotated value with ID '%s'", id, req.GetAnnotatedValueID())
	}
}
