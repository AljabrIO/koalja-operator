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

package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/AljabrIO/koalja-operator/pkg/event"
	"github.com/AljabrIO/koalja-operator/pkg/event/registry"
	"github.com/golang/protobuf/ptypes"
	google_protobuf1 "github.com/golang/protobuf/ptypes/empty"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	link "github.com/AljabrIO/koalja-operator/pkg/agent/link"
)

// subscription of a single client
type subscription struct {
	id        int64
	clientID  string
	expiresAt time.Time
	inflight  *event.Event
}

// AsPB creates a protobuf Subscription from this subscription.
func (s *subscription) AsPB() *event.Subscription {
	return &event.Subscription{
		ID: s.id,
	}
}

// RenewExpiresAt raises the expiration time to now+TTL.
func (s *subscription) RenewExpiresAt() {
	s.expiresAt = time.Now().Add(subscriptionTTL)
}

// stub is an in-memory implementation of an event queue.
type stub struct {
	registry           registry.EventRegistryClient
	uri                string
	queue              chan *event.Event
	retryQueue         chan *event.Event
	subscriptions      map[int64]*subscription
	subscriptionsMutex sync.Mutex
	lastSubscriptionID int64
}

const (
	queueSize       = 1024
	retryQueueSize  = 8
	subscriptionTTL = time.Minute
)

// newStub initializes a new stub
func newStub() *stub {
	return &stub{
		queue:         make(chan *event.Event, queueSize),
		retryQueue:    make(chan *event.Event, retryQueueSize),
		subscriptions: make(map[int64]*subscription),
	}
}

// NewEventPublisher "builds" a new publisher
func (s *stub) NewEventPublisher(deps link.APIDependencies) (event.EventPublisherServer, error) {
	s.uri = deps.URI
	s.registry = deps.EventRegistry
	return s, nil
}

// NewEventSource "builds" a new source
func (s *stub) NewEventSource(deps link.APIDependencies) (event.EventSourceServer, error) {
	return s, nil
}

// Publish an event
func (s *stub) Publish(ctx context.Context, req *event.PublishRequest) (*event.PublishResponse, error) {
	// Try to record event in registry
	e := *req.GetEvent()
	e.Link = s.uri
	if _, err := s.registry.RecordEvent(ctx, &e); err != nil {
		return nil, err
	}

	// Now put event in in-memory queue
	select {
	case s.queue <- &e:
		return &event.PublishResponse{}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Subscribe to events
func (s *stub) Subscribe(ctx context.Context, req *event.SubscribeRequest) (*event.SubscribeResponse, error) {
	s.subscriptionsMutex.Lock()
	defer s.subscriptionsMutex.Unlock()

	// Find existing subscription
	for _, sub := range s.subscriptions {
		if sub.clientID == req.ClientID {
			sub.RenewExpiresAt()
			return &event.SubscribeResponse{
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

	return &event.SubscribeResponse{
		Subscription: &event.Subscription{
			ID: id,
		},
	}, nil
}

// Ping keeps a subscription alive
func (s *stub) Ping(ctx context.Context, req *event.PingRequest) (*google_protobuf1.Empty, error) {
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
func (s *stub) Close(ctx context.Context, req *event.CloseRequest) (*google_protobuf1.Empty, error) {
	s.subscriptionsMutex.Lock()
	defer s.subscriptionsMutex.Unlock()

	id := req.GetSubscription().GetID()
	if sub, found := s.subscriptions[id]; found {
		if sub.inflight != nil {
			// Put inflight event back into queue
			select {
			case s.retryQueue <- sub.inflight:
				// OK
			case <-ctx.Done():
				// Context expired
				return nil, ctx.Err()
			}
		}
		delete(s.subscriptions, id)
	} else {
		return nil, fmt.Errorf("Subscription %d not found", id)
	}
	return &google_protobuf1.Empty{}, nil
}

// Ask for the next event on a subscription
func (s *stub) NextEvent(ctx context.Context, req *event.NextEventRequest) (*event.NextEventResponse, error) {
	id := req.GetSubscription().GetID()
	inflight, err := func() (*event.Event, error) {
		s.subscriptionsMutex.Lock()
		defer s.subscriptionsMutex.Unlock()

		// Lookup subscription
		sub, found := s.subscriptions[id]
		if !found {
			// Subscription not found
			return nil, fmt.Errorf("Subscription %d not found", id)
		}

		// Renew subscription
		sub.RenewExpiresAt()

		// Return (optional) inflight event
		return sub.inflight, nil
	}()
	if err != nil {
		// Subscription not found
		return nil, err
	}
	if inflight != nil {
		// Subscription has event in flight, return that
		return &event.NextEventResponse{
			Event:      inflight,
			NoEventYet: false,
		}, nil
	}

	// No event inflight, get one out of the queue(s)
	waitTimeout, err := ptypes.Duration(req.GetWaitTimeout())
	if err != nil {
		return nil, err
	}
	setEventInflight := func(e *event.Event) (*event.NextEventResponse, error) {
		s.subscriptionsMutex.Lock()
		defer s.subscriptionsMutex.Unlock()

		// Lookup subscription
		if sub, found := s.subscriptions[id]; !found {
			// Subscription not found
			return nil, fmt.Errorf("Subscription %d no longer found", id)
		} else {
			sub.RenewExpiresAt()
			sub.inflight = e
			return &event.NextEventResponse{
				Event:      e,
				NoEventYet: false,
			}, nil
		}
	}
	select {
	case e := <-s.retryQueue:
		// Found event in retry queue
		return setEventInflight(e)
	case e := <-s.queue:
		// Found event in normal queue
		return setEventInflight(e)
	case <-time.After(waitTimeout):
		// No event in time
		return &event.NextEventResponse{
			NoEventYet: true,
		}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Acknowledge the processing of an event
func (s *stub) AckEvent(ctx context.Context, req *event.AckEventRequest) (*google_protobuf1.Empty, error) {
	s.subscriptionsMutex.Lock()
	defer s.subscriptionsMutex.Unlock()

	// Lookup subscription
	id := req.GetSubscription().GetID()
	if sub, found := s.subscriptions[id]; !found {
		// Subscription not found
		return nil, fmt.Errorf("Subscription %d not found", id)
	} else {
		sub.RenewExpiresAt()
		if sub.inflight == nil {
			return nil, fmt.Errorf("Subscription %d has no event to acknowledge", id)
		} else if sub.inflight.GetID() != req.GetEventID() {
			return nil, fmt.Errorf("Subscription %d has different event to acknowledge", id)
		}
		// Got correct request
		sub.inflight = nil
		return &google_protobuf1.Empty{}, nil
	}
}
