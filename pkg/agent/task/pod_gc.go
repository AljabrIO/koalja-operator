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

package task

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	PodCollectionDelay     = 5 * time.Minute // Wait 5 min before collecting a pod
	PodCollectionQueueSize = 5               // After this many pods, the first pods are automatically collected
)

// PodGarbageCollector describes the API implemented by the process that removes obsolete pods
// created by the executor.
type PodGarbageCollector interface {
	// Add a pod to the GC list
	Add(podName, namespace string)
	// Run the collector until the given context is canceled.
	Run(ctx context.Context) error
}

// NewPodGarbageCollector initializes a new pod GC.
func NewPodGarbageCollector(log zerolog.Logger, client client.Client) (PodGarbageCollector, error) {
	return &podGC{
		log:    log,
		client: client,
	}, nil
}

type podGC struct {
	log     zerolog.Logger
	client  client.Client
	mutex   sync.Mutex
	entries []podGCEntry
}

type podGCEntry struct {
	podName      string
	namespace    string
	collectAfter time.Time
}

// Run the executor until the given context is canceled.
func (c *podGC) Run(ctx context.Context) error {
	for {
		// Collect once
		c.collect(ctx)

		select {
		case <-time.After(time.Second * 5):
			// Continue
		case <-ctx.Done():
			// Context canceled
			return nil
		}
	}
}

// Add a pod to the GC list
func (c *podGC) Add(podName, namespace string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.entries = append(c.entries, podGCEntry{
		podName:      podName,
		namespace:    namespace,
		collectAfter: time.Now().Add(PodCollectionDelay),
	})
	c.log.Debug().
		Str("pod", podName).
		Str("namespace", namespace).
		Msg("Added pod to collection queue")
}

// collect once
func (c *podGC) collect(ctx context.Context) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for {
		if len(c.entries) > PodCollectionQueueSize {
			// Queue full, collect first entry
		} else if len(c.entries) > 0 && c.entries[0].collectAfter.Before(time.Now()) {
			// First entry can be collected
		} else {
			// Nothing more to collect
			return
		}
		// Collect first entry
		first := c.entries[0]
		if err := first.collect(ctx, c.client); err != nil {
			c.log.Error().Err(err).Msg("Failed to collect entry")
		} else {
			c.log.Debug().
				Str("pod", first.podName).
				Str("namespace", first.namespace).
				Msg("Collected pod")
		}
		c.entries = c.entries[1:]
	}
}

// collect this entry
func (e podGCEntry) collect(ctx context.Context, c client.Client) error {
	key := client.ObjectKey{Name: e.podName, Namespace: e.namespace}
	var pod corev1.Pod
	if err := c.Get(ctx, key, &pod); err == nil {
		if err := c.Delete(ctx, &pod); err != nil {
			return maskAny(err)
		}
	}
	return nil
}
