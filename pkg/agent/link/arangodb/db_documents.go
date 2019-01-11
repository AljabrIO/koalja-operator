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
	"strconv"
	"time"

	driver "github.com/arangodb/go-driver"

	"github.com/AljabrIO/koalja-operator/pkg/annotatedvalue"
)

type avDocument struct {
	// ID of this document. Auto-incrementing so we can sort on them.
	ID string `json:"_key,omitempty"`
	// Name of the link the AV is queued in
	LinkName string `json:"link_name,omitempty"`
	// ID of the subscription to which this AV was assigned (0==not assigned)
	SubscriptionID int64 `json:"subscription_id"`
	// The annotated value itself
	Value *annotatedvalue.AnnotatedValue `json:"value"`
}

// Remove this document.
func (s *avDocument) Remove(ctx context.Context, queueCol driver.Collection) error {
	if _, err := queueCol.RemoveDocument(ctx, s.ID); err != nil {
		return maskAny(err)
	}
	return nil
}

// subscription of a single client
type subscription struct {
	// ID of the subscription. Auto-incrementing number.
	ID string `json:"_key,omitempty"`
	// Identier of the client (aka Task agent)
	ClientID string `json:"client_id,omitempty"`
	// Name of the link this subscription is part of
	LinkName string `json:"link_name,omitempty"`
	// Expiration time of this subscription
	ExpiresAt time.Time `json:"expired_at"`
}

// IsExpired returns true when this subscription is expired.
func (s *subscription) IsExpired() bool {
	return s.ExpiresAt.Before(time.Now())
}

// getSubscriptionByID loads a subscription with given ID from the DB.
// Returns NotFound error if not found.
func getSubscriptionByID(ctx context.Context, id int64, subscrCol driver.Collection) (*subscription, error) {
	subscr := &subscription{}
	if _, err := subscrCol.ReadDocument(ctx, strconv.FormatInt(id, 10), subscr); err != nil {
		return nil, maskAny(err)
	}
	return subscr, nil
}

// findSubscriptionByClientID loads a subscription with given client-ID from the DB.
// Returns nil if not found.
func findSubscriptionByClientID(ctx context.Context, clientID string, subscrCol driver.Collection) (*subscription, error) {
	q := fmt.Sprintf("FOR doc IN %s FILTER doc.client_id == @client_id RETURN doc", subscrCol.Name())
	cursor, err := subscrCol.Database().Query(ctx, q, map[string]interface{}{
		"client_id": clientID,
	})
	if err != nil {
		return nil, maskAny(err)
	}
	for {
		subscr := &subscription{}
		if _, err := cursor.ReadDocument(ctx, subscr); driver.IsNoMoreDocuments(err) {
			// No such subscription
			return nil, nil
		} else if err != nil {
			return nil, maskAny(err)
		}
		// return the document
		return subscr, nil
	}
}

// getExpiredSubscriptions fetches all expired subscriptions from the DB.
func getExpiredSubscriptions(ctx context.Context, linkName string, subscrCol driver.Collection) ([]*subscription, error) {
	q := fmt.Sprintf("FOR doc IN %s FILTER doc.link_name == @link_name RETURN doc", subscrCol.Name())
	cursor, err := subscrCol.Database().Query(ctx, q, map[string]interface{}{
		"link_name": linkName,
	})
	if err != nil {
		return nil, maskAny(err)
	}
	var result []*subscription
	for {
		subscr := &subscription{}
		if _, err := cursor.ReadDocument(ctx, subscr); driver.IsNoMoreDocuments(err) {
			// No such subscription, we,re done
			return result, nil
		} else if err != nil {
			return nil, maskAny(err)
		}
		if subscr.IsExpired() {
			result = append(result, subscr)
		}
	}
}

// GetID returns the ID of the subscription as an int64.
func (s *subscription) GetID() int64 {
	x, _ := strconv.ParseInt(s.ID, 10, 64)
	return x
}

// AsPB creates a protobuf Subscription from this subscription.
func (s *subscription) AsPB() *annotatedvalue.Subscription {
	return &annotatedvalue.Subscription{
		ID: s.GetID(),
	}
}

// RenewExpiresAt raises the expiration time to now+TTL.
func (s *subscription) RenewExpiresAt(ctx context.Context, subscrCol driver.Collection) error {
	update := subscription{
		ExpiresAt: time.Now().Add(subscriptionTTL),
	}
	if _, err := subscrCol.UpdateDocument(ctx, s.ID, update); err != nil {
		return maskAny(err)
	}
	s.ExpiresAt = update.ExpiresAt
	return nil
}

// UnassignQueueValues un-assigned all annotated values that we're assigned to this subscription.
func (s *subscription) UnassignQueueValues(ctx context.Context, queueCol driver.Collection) error {
	q := fmt.Sprintf("FOR doc IN %s FILTER doc.subscription_id == @subscription_id UPDATE { _key: doc._key, subscription_id: 0 } IN %s", queueCol.Name(), queueCol.Name())
	if _, err := queueCol.Database().Query(ctx, q, map[string]interface{}{
		"subscription_id": s.ID,
	}); err != nil {
		return maskAny(err)
	}
	return nil
}

// Remove this document.
func (s *subscription) Remove(ctx context.Context, subscrCol driver.Collection) error {
	if _, err := subscrCol.RemoveDocument(ctx, s.ID); err != nil {
		return maskAny(err)
	}
	return nil
}

// AssignFirstValue assigns the first unassigned annotated value to this subscription and returns it.
// Returns nil if no such annotated value exists.
func (s *subscription) AssignFirstValue(ctx context.Context, queueCol driver.Collection) (*avDocument, error) {
	q := fmt.Sprintf("FOR doc IN %s FILTER doc.subscription_id == 0 AND doc.link_name == @link_name UPDATE { _key: doc._key, subscription_id: @subscription_id } IN %s RETURN NEW", queueCol.Name(), queueCol.Name())
	cursor, err := queueCol.Database().Query(ctx, q, map[string]interface{}{
		"subscription_id": s.GetID(),
		"link_name":       s.LinkName,
	})
	if err != nil {
		return nil, maskAny(err)
	}
	for {
		var doc avDocument
		if _, err := cursor.ReadDocument(ctx, &doc); driver.IsNoMoreDocuments(err) {
			// Not found
			return nil, nil
		} else if err != nil {
			return nil, maskAny(err)
		}
		// Document found
		return &doc, nil
	}
}

// FindAnnotatedValueByID loads an annotated value document with given annotated-value ID
// for this subscription from the DB.
// Returns nil if not found.
func (s *subscription) FindAnnotatedValueByID(ctx context.Context, annotatedValueID string, queueCol driver.Collection) (*avDocument, error) {
	q := fmt.Sprintf("FOR doc IN %s FILTER doc.subscription_id == @subscription_id AND doc.value.id == @av_id RETURN doc", queueCol.Name())
	cursor, err := queueCol.Database().Query(ctx, q, map[string]interface{}{
		"subscription_id": s.GetID(),
		"av_id":           annotatedValueID,
	})
	if err != nil {
		return nil, maskAny(err)
	}
	for {
		avdoc := &avDocument{}
		if _, err := cursor.ReadDocument(ctx, avdoc); driver.IsNoMoreDocuments(err) {
			// No such document
			return nil, nil
		} else if err != nil {
			return nil, maskAny(err)
		}
		// return the document
		return avdoc, nil
	}
}
