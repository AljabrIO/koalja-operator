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
	"sort"
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
	// Set the selector of Pods to garbage collect when terminated.
	Set(podLabels map[string]string, namespace string)
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
	log    zerolog.Logger
	client client.Client

	podLabels map[string]string
	namespace string
	mutex     sync.Mutex
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

// Set the selector of Pods to garbage collect when terminated.
func (c *podGC) Set(podLabels map[string]string, namespace string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.podLabels = podLabels
	c.namespace = namespace
}

// collect once
func (c *podGC) collect(ctx context.Context) {
	c.mutex.Lock()
	podLabels := c.podLabels
	namespace := c.namespace
	c.mutex.Unlock()

	if len(podLabels) == 0 || namespace == "" {
		return
	}

	// Fetch pods that match selector
	var list corev1.PodList
	listOpts := client.MatchingLabels(podLabels).InNamespace(namespace)
	if err := c.client.List(ctx, listOpts, &list); err != nil {
		c.log.Error().Err(err).Msg("Failed to list pods")
	}

	// Consider only failed pods
	failed := selectFailedPods(list.Items)

	// Sort pods by creation date (oldest first)
	sort.Slice(failed, func(i, j int) bool {
		tsi := failed[i].CreationTimestamp
		tsj := failed[j].CreationTimestamp
		return tsi.Before(&tsj)
	})

	for {
		if len(failed) > PodCollectionQueueSize {
			// Queue full, collect first entry
		} else if len(failed) > 0 && isFailedLongAgo(failed[0]) {
			// First entry can be collected
		} else {
			// Nothing more to collect
			return
		}
		// Collect first entry
		first := failed[0]
		if err := c.client.Delete(ctx, &first); err != nil {
			c.log.Error().Err(err).Msg("Failed to collect oldest pod")
		} else {
			c.log.Debug().
				Str("pod", first.GetName()).
				Str("namespace", first.GetNamespace()).
				Msg("Collected pod")
		}
		failed = failed[1:]
	}
}

// selectFailedPods returns the selection (of given list) where
// the pod has failed.
func selectFailedPods(list []corev1.Pod) []corev1.Pod {
	var result []corev1.Pod
	for _, p := range list {
		if p.Status.Phase == corev1.PodFailed {
			result = append(result, p)
		}
	}
	return result
}

func isFailedLongAgo(p corev1.Pod) bool {
	return false
}
