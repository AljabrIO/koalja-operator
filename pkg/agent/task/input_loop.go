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
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/eapache/go-resiliency/breaker"
	ptypes "github.com/gogo/protobuf/types"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"

	"github.com/AljabrIO/koalja-operator/pkg/annotatedvalue"
	avclient "github.com/AljabrIO/koalja-operator/pkg/annotatedvalue/client"
	koalja "github.com/AljabrIO/koalja-operator/pkg/apis/koalja/v1alpha1"
	"github.com/AljabrIO/koalja-operator/pkg/constants"
	"github.com/AljabrIO/koalja-operator/pkg/tracking"
	"github.com/AljabrIO/koalja-operator/pkg/util/retry"
	"github.com/dchest/uniuri"
)

// inputLoop subscribes to all inputs of a task and build
// snapshots of incoming annotated values, according to the policy on each input.
type inputLoop struct {
	log             zerolog.Logger
	spec            *koalja.TaskSpec
	inputAddressMap map[string]string // map[inputName]AnnotatedValueSourceAddress
	clientID        string
	snapshot        InputSnapshot
	mutex           sync.Mutex
	executionCount  int32
	execQueue       chan (*InputSnapshot)
	executor        Executor
	snapshotService SnapshotService
	statistics      *tracking.TaskStatistics
}

// newInputLoop initializes a new input loop.
func newInputLoop(log zerolog.Logger, spec *koalja.TaskSpec, pod *corev1.Pod, executor Executor, snapshotService SnapshotService, statistics *tracking.TaskStatistics) (*inputLoop, error) {
	inputAddressMap := make(map[string]string)
	for _, tis := range spec.Inputs {
		annKey := constants.CreateInputLinkAddressAnnotationName(tis.Name)
		address := pod.GetAnnotations()[annKey]
		if address == "" {
			return nil, fmt.Errorf("No input address annotation found for input '%s'", tis.Name)
		}
		inputAddressMap[tis.Name] = address
	}
	return &inputLoop{
		log:             log,
		spec:            spec,
		inputAddressMap: inputAddressMap,
		clientID:        uniuri.New(),
		execQueue:       make(chan *InputSnapshot),
		executor:        executor,
		snapshotService: snapshotService,
		statistics:      statistics,
	}, nil
}

// Run the input loop until the given context is canceled.
func (il *inputLoop) Run(ctx context.Context) error {
	defer close(il.execQueue)
	g, lctx := errgroup.WithContext(ctx)
	if len(il.spec.Inputs) > 0 {
		// Watch inputs
		for _, tis := range il.spec.Inputs {
			tis := tis // Bring in scope
			stats := il.statistics.InputByName(tis.Name)
			g.Go(func() error {
				return il.watchInput(lctx, il.spec.SnapshotPolicy, tis, stats)
			})
		}
	}
	if il.spec.HasLaunchPolicyCustom() {
		// Custom launch policy, run executor all the time
		g.Go(func() error {
			return il.runExecWithCustomLaunchPolicy(lctx)
		})
	}
	g.Go(func() error {
		return il.processExecQueue(lctx)
	})
	if err := g.Wait(); err != nil {
		return err
	}
	return nil
}

// processExecQueue pulls snapshots from the exec queue and:
// - executes them in case of tasks with auto launch policy or
// - allows the executor to pull the snapshot in case of tasks with custom launch policy.
func (il *inputLoop) processExecQueue(ctx context.Context) error {
	var lastCancel context.CancelFunc
	for {
		select {
		case snapshot, ok := <-il.execQueue:
			if !ok {
				return nil
			}
			if il.spec.HasLaunchPolicyAuto() {
				// Automatic launch policy; Go launch an executor
				if err := il.execOnSnapshot(ctx, snapshot); ctx.Err() != nil {
					return ctx.Err()
				} else if err != nil {
					il.log.Error().Err(err).Msg("Failed to execute task")
				}
			} else if il.spec.HasLaunchPolicyRestart() {
				// Restart launch policy;
				// - Cancel existing executor
				if lastCancel != nil {
					lastCancel()
					lastCancel = nil
				}
				// - Go launch a new executor
				var launchCtx context.Context
				launchCtx, cancel := context.WithCancel(ctx)
				lastCancel = cancel
				go func() {
					defer cancel()
					if err := il.execOnSnapshot(launchCtx, snapshot); launchCtx.Err() != nil {
						// Context canceled, ignore
					} else if err != nil {
						il.log.Error().Err(err).Msg("Failed to execute task")
					}
				}()
			} else if il.spec.HasLaunchPolicyCustom() {
				// Custom launch policy; Make snapshot available to snapshot service
				if err := il.snapshotService.Execute(ctx, snapshot); ctx.Err() != nil {
					return ctx.Err()
				} else if err != nil {
					il.log.Error().Err(err).Msg("Failed to execute task with custom launch policy")
				}
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// runExecWithCustomLaunchPolicy runs the executor continuesly without
// providing it a valid snapshot when it starts.
// The executor must use the SnapshotService to pull for new snapshots.
func (il *inputLoop) runExecWithCustomLaunchPolicy(ctx context.Context) error {
	b := breaker.New(5, 1, time.Second*10)
	for {
		snapshot := &InputSnapshot{}
		if err := b.Run(func() error {
			if err := il.execOnSnapshot(ctx, snapshot); ctx.Err() != nil {
				return ctx.Err()
			} else if err != nil {
				il.log.Error().Err(err).Msg("Failed to execute task")
				return maskAny(err)
			}
			return nil
		}); ctx.Err() != nil {
			return ctx.Err()
		} else if err == breaker.ErrBreakerOpen {
			// Circuit break open
			select {
			case <-time.After(time.Second * 5):
				// Retry
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
}

// execOnSnapshot executes the task executor for the given snapshot.
func (il *inputLoop) execOnSnapshot(ctx context.Context, snapshot *InputSnapshot) error {
	// Update statistics
	atomic.AddInt64(&il.statistics.SnapshotsWaiting, -1)
	atomic.AddInt64(&il.statistics.SnapshotsInProgress, 1)
	snapshot.AddInProgressStatistics(1)

	// Update statistics on return
	defer func() {
		snapshot.AddInProgressStatistics(-1)
		snapshot.AddProcessedStatistics(1)
		atomic.AddInt64(&il.statistics.SnapshotsInProgress, -1)
	}()
	if err := il.executor.Execute(ctx, snapshot); ctx.Err() != nil {
		atomic.AddInt64(&il.statistics.SnapshotsFailed, 1)
		return ctx.Err()
	} else if err != nil {
		atomic.AddInt64(&il.statistics.SnapshotsFailed, 1)
		il.log.Debug().Err(err).Msg("executor.Execute failed")
		return maskAny(err)
	} else {
		// Acknowledge all annotated values in the snapshot
		atomic.AddInt64(&il.statistics.SnapshotsSucceeded, 1)
		il.log.Debug().Msg("acknowledging all annotated values in snapshot")
		if err := snapshot.AckAll(ctx); err != nil {
			il.log.Error().Err(err).Msg("Failed to acknowledge annotated values")
		}
		return nil
	}
}

// processAnnotatedValue the annotated value coming from the given input.
func (il *inputLoop) processAnnotatedValue(ctx context.Context, av *annotatedvalue.AnnotatedValue, snapshotPolicy koalja.SnapshotPolicy, tis koalja.TaskInputSpec, stats *tracking.TaskInputStatistics, ack func(context.Context, *annotatedvalue.AnnotatedValue) error) error {
	il.mutex.Lock()
	defer il.mutex.Unlock()
	// Wait until snapshot has a place in the sequence of an annotated values for given input
	for {
		seqLen := il.snapshot.GetSequenceLengthForInput(tis.Name)
		if seqLen < tis.GetMaxSequenceLength() {
			// There is space available in the sequence to add at least 1 more annotated value
			break
		}
		// Wait a bit
		il.mutex.Unlock()
		select {
		case <-time.After(time.Millisecond * 50):
			// Retry
			il.mutex.Lock()
		case <-ctx.Done():
			// Context canceled
			il.mutex.Lock()
			return ctx.Err()
		}
	}

	// Set the annotated value in the snapshot
	if err := il.snapshot.Set(ctx, tis.Name, av, tis.GetMinSequenceLength(), tis.GetMaxSequenceLength(), stats, ack); err != nil {
		return err
	}

	// Build list of inputs that we use in the snapshot (leave out ones with MergeInto)
	snapshotInputs := il.spec.SnapshotInputs()

	// See if we should execute the task now
	if !il.snapshot.IsReadyForExecution(len(snapshotInputs)) {
		// Not all inputs have received sufficient annotated values yet
		return nil
	}

	// Clone the snapshot
	clonedSnapshot := il.snapshot.Clone()
	il.executionCount++

	// Prepare snapshot for next execution
	for _, inp := range snapshotInputs {
		if snapshotPolicy.IsAllNew() {
			// Delete annotated value
			il.snapshot.Delete(inp.Name)
		} else if snapshotPolicy.IsSlidingWindow() {
			// Remove Slide number of values
			il.snapshot.Slide(inp.Name, inp.GetSlide())
			// No need to acknowledge remaining values
			il.snapshot.RemoveAck(inp.Name)
		} else if snapshotPolicy.IsSwapNew4Old() {
			// Remove need to acknowledge annotated value
			il.snapshot.RemoveAck(inp.Name)
		}
	}

	// Update statistics
	atomic.AddInt64(&il.statistics.SnapshotsWaiting, 1)

	// Push snapshot into execution queue
	il.mutex.Unlock()
	il.execQueue <- clonedSnapshot
	il.mutex.Lock()

	return nil
}

// watchInput subscribes to the given input and gathers annotated values until the given context is canceled.
func (il *inputLoop) watchInput(ctx context.Context, snapshotPolicy koalja.SnapshotPolicy, tis koalja.TaskInputSpec, stats *tracking.TaskInputStatistics) error {
	// Create client
	address := il.inputAddressMap[tis.Name]
	tisOut := tis
	if tis.HasMergeInto() {
		tisOut, _ = il.spec.InputByName(tis.MergeInto)
	}

	// Prepare loop
	subscribeAndReadLoop := func(ctx context.Context, c avclient.AnnotatedValueSourceClient) error {
		defer c.CloseConnection()
		resp, err := c.Subscribe(ctx, &annotatedvalue.SubscribeRequest{
			ClientID: il.clientID,
		})
		if err != nil {
			return err
		}
		subscr := *resp.GetSubscription()

		ack := func(ctx context.Context, av *annotatedvalue.AnnotatedValue) error {
			if err := retry.Do(ctx, func(ctx context.Context) error {
				if _, err := c.Ack(ctx, &annotatedvalue.AckRequest{
					Subscription:     &subscr,
					AnnotatedValueID: av.GetID(),
				}); err != nil {
					il.log.Debug().Err(err).Msg("Ack annotated value attempt failed")
					return err
				}
				return nil
			}, retry.Timeout(constants.TimeoutAckAnnotatedValue)); err != nil {
				il.log.Error().Err(err).Msg("Failed to ack annotated value")
				return maskAny(err)
			}
			return nil
		}

		for {
			resp, err := c.Next(ctx, &annotatedvalue.NextRequest{
				Subscription: &subscr,
				WaitTimeout:  ptypes.DurationProto(time.Second * 30),
			})
			if ctx.Err() != nil {
				return ctx.Err()
			} else if err != nil {
				// handle err
				il.log.Error().Err(err).Msg("Failed to fetch next annotated value")
			} else {
				// Process annotated value (if any)
				if av := resp.GetAnnotatedValue(); av != nil {
					atomic.AddInt64(&stats.AnnotatedValuesReceived, 1)
					if err := il.processAnnotatedValue(ctx, av, snapshotPolicy, tisOut, stats, ack); err != nil {
						il.log.Error().Err(err).Msg("Failed to process annotated value")
					}
				}
			}
		}
	}

	// Keep creating connection, subscribe and loop
	for {
		c, err := avclient.NewAnnotatedValueSourceClient(address)
		if err == nil {
			if err := subscribeAndReadLoop(ctx, c); ctx.Err() != nil {
				return ctx.Err()
			} else if err != nil {
				il.log.Error().Err(err).Msg("Failure in subscribe & read annotated value loop")
			}
		} else if ctx.Err() != nil {
			return ctx.Err()
		} else {
			il.log.Error().Err(err).Msg("Failed to create annotated value source client")
		}
		// Wait a bit
		select {
		case <-time.After(time.Second * 5):
			// Retry
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
