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

package stub

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/AljabrIO/koalja-operator/pkg/annotatedvalue"
	"github.com/AljabrIO/koalja-operator/pkg/annotatedvalue/registry"
	"github.com/AljabrIO/koalja-operator/pkg/tracking"
	ptypes "github.com/gogo/protobuf/types"
	google_protobuf1 "github.com/golang/protobuf/ptypes/empty"
	"github.com/rs/zerolog"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	link "github.com/AljabrIO/koalja-operator/pkg/agent/link"
	"github.com/AljabrIO/koalja-operator/pkg/agent/pipeline"
)

// subscription of a single client
type subscription struct {
	id        int64
	clientID  string
	expiresAt time.Time
	inflight  []*annotatedvalue.AnnotatedValue
}

// AsPB creates a protobuf Subscription from this subscription.
func (s *subscription) AsPB() *annotatedvalue.Subscription {
	return &annotatedvalue.Subscription{
		ID: s.id,
	}
}

// RenewExpiresAt raises the expiration time to now+TTL.
func (s *subscription) RenewExpiresAt() {
	s.expiresAt = time.Now().Add(subscriptionTTL)
}

// stub is an in-memory implementation of an annotated value queue.
type stub struct {
	log                zerolog.Logger
	registry           registry.AnnotatedValueRegistryClient
	agentRegistry      pipeline.AgentRegistryClient
	uri                string
	queue              chan *annotatedvalue.AnnotatedValue
	retryQueue         chan *annotatedvalue.AnnotatedValue
	subscriptions      map[int64]*subscription
	subscriptionsMutex sync.Mutex
	lastSubscriptionID int64
	statistics         *tracking.LinkStatistics
}

const (
	queueSize       = 1024
	retryQueueSize  = 32
	subscriptionTTL = time.Minute
)

// NewStub initializes a new stub API builder
func NewStub(log zerolog.Logger) link.APIBuilder {
	return &stub{
		log:           log,
		queue:         make(chan *annotatedvalue.AnnotatedValue, queueSize),
		retryQueue:    make(chan *annotatedvalue.AnnotatedValue, retryQueueSize),
		subscriptions: make(map[int64]*subscription),
	}
}

// NewAnnotatedValuePublisher "builds" a new publisher
func (s *stub) NewAnnotatedValuePublisher(deps link.APIDependencies) (annotatedvalue.AnnotatedValuePublisherServer, error) {
	s.uri = deps.URI
	s.registry = deps.AnnotatedValueRegistry
	s.agentRegistry = deps.AgentRegistry
	s.statistics = deps.Statistics
	return s, nil
}

// NewAnnotatedValueSource "builds" a new source
func (s *stub) NewAnnotatedValueSource(deps link.APIDependencies) (annotatedvalue.AnnotatedValueSourceServer, error) {
	return s, nil
}

// Publish an event
func (s *stub) Publish(ctx context.Context, req *annotatedvalue.PublishRequest) (*annotatedvalue.PublishResponse, error) {
	s.log.Debug().Interface("annotatedvalue", req.AnnotatedValue).Msg("Publish request")
	// Try to record event in registry
	av := *req.GetAnnotatedValue()
	av.Link = s.uri
	if _, err := s.registry.Record(ctx, &av); err != nil {
		return nil, maskAny(err)
	}
	atomic.AddInt64(&s.statistics.AnnotatedValuesWaiting, 1)

	// Now put event in in-memory queue
	select {
	case s.queue <- &av:
		return &annotatedvalue.PublishResponse{}, nil
	case <-ctx.Done():
		return nil, maskAny(ctx.Err())
	}
}

// Subscribe to events
func (s *stub) Subscribe(ctx context.Context, req *annotatedvalue.SubscribeRequest) (*annotatedvalue.SubscribeResponse, error) {
	s.log.Debug().Str("clientID", req.ClientID).Msg("Subscribe request")
	s.subscriptionsMutex.Lock()
	defer s.subscriptionsMutex.Unlock()

	// Find existing subscription
	for _, sub := range s.subscriptions {
		if sub.clientID == req.ClientID {
			sub.RenewExpiresAt()
			return &annotatedvalue.SubscribeResponse{
				Subscription: sub.AsPB(),
				TTL:          ptypes.DurationProto(subscriptionTTL),
			}, nil
		}
	}

	// Add new subscription
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
func (s *stub) Ping(ctx context.Context, req *annotatedvalue.PingRequest) (*google_protobuf1.Empty, error) {
	s.log.Debug().Int64("id", req.GetSubscription().GetID()).Msg("Ping request")
	s.subscriptionsMutex.Lock()
	defer s.subscriptionsMutex.Unlock()

	id := req.GetSubscription().GetID()
	if sub, found := s.subscriptions[id]; found {
		sub.RenewExpiresAt()
	} else {
		return nil, fmt.Errorf("Subscription %d not found", id)
	}
	return &google_protobuf1.Empty{}, nil
}

// Close a subscription
func (s *stub) Close(ctx context.Context, req *annotatedvalue.CloseRequest) (*google_protobuf1.Empty, error) {
	s.log.Debug().Int64("id", req.GetSubscription().GetID()).Msg("Ping request")
	s.subscriptionsMutex.Lock()
	defer s.subscriptionsMutex.Unlock()

	id := req.GetSubscription().GetID()
	if sub, found := s.subscriptions[id]; found {
		if len(sub.inflight) != 0 {
			// Put inflight events back into queue
			inflight := sub.inflight
			for _, av := range inflight {
				select {
				case s.retryQueue <- av:
					sub.inflight = sub.inflight[1:] // Remove inflight annotated value
					// Update statistics
					atomic.AddInt64(&s.statistics.AnnotatedValuesInProgress, -1)
					atomic.AddInt64(&s.statistics.AnnotatedValuesWaiting, 1)
					// OK
				case <-ctx.Done():
					// Context expired
					return nil, maskAny(ctx.Err())
				default:
					// retryQueue full
					// Since we have a mutex locked, we cannot wait here until retry queue has space.
					return nil, maskAny(fmt.Errorf("Retry queue full"))
				}
			}
		}
		delete(s.subscriptions, id)
	} else {
		return nil, fmt.Errorf("Subscription %d not found", id)
	}
	return &google_protobuf1.Empty{}, nil
}

// Ask for the next annotated value on a subscription
func (s *stub) Next(ctx context.Context, req *annotatedvalue.NextRequest) (*annotatedvalue.NextResponse, error) {
	id := req.GetSubscription().GetID()
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

	// No event inflight, get one out of the queue(s)
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
func (s *stub) Ack(ctx context.Context, req *annotatedvalue.AckRequest) (*google_protobuf1.Empty, error) {
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
