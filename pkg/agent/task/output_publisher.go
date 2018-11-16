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
	"strings"
	"sync/atomic"
	"time"

	"github.com/dchest/uniuri"
	ptypes "github.com/gogo/protobuf/types"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"

	"github.com/AljabrIO/koalja-operator/pkg/annotatedvalue"
	avclient "github.com/AljabrIO/koalja-operator/pkg/annotatedvalue/client"
	koalja "github.com/AljabrIO/koalja-operator/pkg/apis/koalja/v1alpha1"
	"github.com/AljabrIO/koalja-operator/pkg/constants"
	ptask "github.com/AljabrIO/koalja-operator/pkg/task"
	"github.com/AljabrIO/koalja-operator/pkg/tracking"
	"github.com/AljabrIO/koalja-operator/pkg/util/retry"
)

// OutputPublisher specifies the API of the output publisher
type OutputPublisher interface {
	// Publish pushes the given annotated value onto the application output channel.
	Publish(ctx context.Context, outputName string, av annotatedvalue.AnnotatedValue, snapshot *InputSnapshot) (bool, *annotatedvalue.AnnotatedValue, error)
}

// outputPublisher is responsible for publishing events to one or more EventPublishers.
type outputPublisher struct {
	log                zerolog.Logger
	spec               *koalja.TaskSpec
	outputAddressesMap map[string][]string                              // map[outputName]annotatedValuePublisherAddresses
	canPublishChannels map[string][]chan chan bool                      // map[outputName][]chan <canPublishIndicator>
	publishChannels    map[string][]chan *annotatedvalue.AnnotatedValue // map[outputName][]chan annotatedvalue
	statistics         *tracking.TaskStatistics
}

// newOutputPublisher initializes a new outputPublisher.
func newOutputPublisher(log zerolog.Logger, spec *koalja.TaskSpec, pod *corev1.Pod, statistics *tracking.TaskStatistics) (*outputPublisher, error) {
	outputAddressesMap := make(map[string][]string)
	canPublishChannels := make(map[string][]chan chan bool)
	publishChannels := make(map[string][]chan *annotatedvalue.AnnotatedValue)
	for _, tos := range spec.Outputs {
		annKey := constants.CreateOutputLinkAddressesAnnotationName(tos.Name)
		addressesStr := pod.GetAnnotations()[annKey]
		if addressesStr == "" {
			return nil, fmt.Errorf("No output addresses annotation found for input '%s'", tos.Name)
		}
		addresses := strings.Split(addressesStr, ",")
		outputAddressesMap[tos.Name] = addresses
		canPublishChans := make([]chan chan bool, 0, len(addresses))
		publishChans := make([]chan *annotatedvalue.AnnotatedValue, 0, len(addresses))
		for range addresses {
			canPublishChans = append(canPublishChans, make(chan chan bool))
			publishChans = append(publishChans, make(chan *annotatedvalue.AnnotatedValue))
		}
		canPublishChannels[tos.Name] = canPublishChans
		publishChannels[tos.Name] = publishChans
	}
	return &outputPublisher{
		log:                log,
		spec:               spec,
		outputAddressesMap: outputAddressesMap,
		canPublishChannels: canPublishChannels,
		publishChannels:    publishChannels,
		statistics:         statistics,
	}, nil
}

// Run the output publisher until the given context is canceled.
func (op *outputPublisher) Run(ctx context.Context) error {
	g, lctx := errgroup.WithContext(ctx)
	for _, tos := range op.spec.Outputs {
		tos := tos // bring into scope
		addresses := op.outputAddressesMap[tos.Name]
		canPublishChans := op.canPublishChannels[tos.Name]
		publishChans := op.publishChannels[tos.Name]
		outputStats := op.statistics.OutputByName(tos.Name)
		for i, publishChan := range publishChans {
			publishChan := publishChan // bring into scope
			canPublishChan := canPublishChans[i]
			addr := addresses[i]
			g.Go(func() error {
				if err := op.runForOutput(lctx, tos, addr, canPublishChan, publishChan, outputStats); err != nil {
					return maskAny(err)
				}
				return nil
			})
		}
	}
	if err := g.Wait(); err != nil {
		return maskAny(err)
	}
	return nil
}

// OutputReady implements the notification endpoint for tasks with an "Auto" ready setting.
func (op *outputPublisher) OutputReady(ctx context.Context, req *ptask.OutputReadyRequest) (*ptask.OutputReadyResponse, error) {
	log := op.log.With().
		Str("output", req.GetOutputName()).
		Str("data", limitDataLength(req.GetAnnotatedValueData())).
		Logger()
	log.Debug().Msg("Received OutputReady request")

	av := annotatedvalue.AnnotatedValue{
		Data: req.GetAnnotatedValueData(),
	}
	accepted, publishedAv, err := op.Publish(ctx, req.GetOutputName(), av, nil)
	if err != nil {
		return nil, maskAny(err)
	}
	return &ptask.OutputReadyResponse{
		Accepted:         accepted,
		AnnotatedValueID: publishedAv.GetID(),
	}, nil
}

// Publish pushes the given annotated value onto the application output channel.
// Returns: (accepted,AnnotedValue,error)
func (op *outputPublisher) Publish(ctx context.Context, outputName string, av annotatedvalue.AnnotatedValue, snapshot *InputSnapshot) (bool, *annotatedvalue.AnnotatedValue, error) {
	// Check if we can publish to all attached links
	canPublishChans, found := op.canPublishChannels[outputName]
	if !found {
		return false, nil, fmt.Errorf("No channels found for output '%s'", outputName)
	}

	// Ask all outgoing links if publishing is allowed
	g, lctx := errgroup.WithContext(ctx)
	canPublish := int32(1)
	for _, canPublishChan := range canPublishChans {
		canPublishChan := canPublishChan // bring into scope
		g.Go(func() error {
			canPublishResultChan := make(chan bool)
			select {
			case canPublishChan <- canPublishResultChan:
				// Request send, now wait for answer
				// canPublishResultChan has reached other go-routine and it will be closed there
				select {
				case answer := <-canPublishResultChan:
					// Got the answer
					if !answer {
						atomic.StoreInt32(&canPublish, 0)
					}
				case <-lctx.Done():
					// Context canceled
					close(canPublishResultChan)
					return lctx.Err()
				}
				return nil
			case <-lctx.Done():
				// Context canceled
				// canPublishResultChan has not reached other go-routine, so we must close it
				close(canPublishResultChan)
				return lctx.Err()
			}
		})
	}
	if err := g.Wait(); err != nil {
		return false, nil, maskAny(err)
	}
	if canPublish == 0 {
		// Publication is not allowed now
		return false, nil, nil
	}

	// Fill in the blanks of the annotated value
	if av.GetID() == "" {
		av.ID = uniuri.New()
	}
	if av.GetCreatedAt() == nil {
		av.CreatedAt = ptypes.TimestampNow()
	}
	if av.GetSourceTask() == "" {
		av.SourceTask = op.spec.Name
	}
	if av.GetSourceTaskOutput() == "" {
		av.SourceTaskOutput = outputName
	}
	if snapshot != nil && len(av.SourceInputs) == 0 {
		for _, inp := range op.spec.Inputs {
			inpAvSeq := snapshot.GetSequence(inp.Name)
			inpAvIDs := make([]string, len(inpAvSeq))
			for i, inpAv := range inpAvSeq {
				inpAvIDs[i] = inpAv.GetID()
			}
			av.SourceInputs = append(av.SourceInputs, &annotatedvalue.AnnotatedValueSourceInput{
				IDs:       inpAvIDs,
				InputName: inp.Name,
			})
		}
	}

	// Publish on all outgoing links
	publishChans, found := op.publishChannels[outputName]
	if !found {
		return false, nil, fmt.Errorf("No channels found for output '%s'", outputName)
	}
	g, lctx = errgroup.WithContext(ctx)
	for _, publishChan := range publishChans {
		publishChan := publishChan // bring into scope
		g.Go(func() error {
			select {
			case publishChan <- &av:
				// Done
				return nil
			case <-lctx.Done():
				// Context canceled
				return lctx.Err()
			}
		})
	}
	if err := g.Wait(); err != nil {
		return false, nil, maskAny(err)
	}
	return true, &av, nil
}

// runForOutput keeps publishing annotated values for the given output until the given context is canceled.
func (op *outputPublisher) runForOutput(ctx context.Context, tos koalja.TaskOutputSpec, addr string, canPublishChan chan chan bool, publishChan chan *annotatedvalue.AnnotatedValue, stats *tracking.TaskOutputStatistics) error {
	defer close(canPublishChan)
	defer close(publishChan)
	log := op.log.With().Str("address", addr).Str("output", tos.Name).Logger()

	runUntilError := func() error {
		avpClient, err := avclient.NewAnnotatedValuePublisherClient(addr)
		if err != nil {
			return maskAny(err)
		}
		defer avpClient.CloseConnection()

		for {
			select {
			case canPublishResultChan := <-canPublishChan:
				// Is publication allowed?
				if err := retry.Do(ctx, func(ctx context.Context) error {
					resp, err := avpClient.CanPublish(ctx, &annotatedvalue.CanPublishRequest{})
					if err != nil {
						log.Debug().Err(err).Msg("canpublish attempt failed")
						return maskAny(err)
					}
					canPublishResultChan <- resp.Allowed
					close(canPublishResultChan)
					return nil
				}, retry.Timeout(constants.TimeoutPublishAnnotatedValue)); err != nil {
					log.Error().Err(err).Msg("Failed to query can-publish")
					canPublishResultChan <- false // We don't know the answer, so block publication to be safe
					close(canPublishResultChan)
					return maskAny(err)
				}
			case av := <-publishChan:
				// Publish annotated value
				if err := retry.Do(ctx, func(ctx context.Context) error {
					if _, err := avpClient.Publish(ctx, &annotatedvalue.PublishRequest{AnnotatedValue: av}); err != nil {
						log.Debug().Err(err).Str("id", av.ID).Msg("published annotated value attempt failed")
						return maskAny(err)
					}
					return nil
				}, retry.Timeout(constants.TimeoutPublishAnnotatedValue)); err != nil {
					log.Error().Err(err).Str("id", av.ID).Msg("Failed to publish annotated value")
					return maskAny(err)
				}
				log.Debug().Str("id", av.ID).Msg("published annotated value")
				atomic.AddInt64(&stats.AnnotatedValuesPublished, 1)
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	for {
		if err := runUntilError(); ctx.Err() != nil {
			return ctx.Err()
		} else if err != nil {
			log.Error().Err(err).Msg("Error in publication loop")
		}
		select {
		case <-time.After(time.Second * 2):
			// Continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// limitDataLength returns the given data, trimmed to a reasonable length.
func limitDataLength(data string) string {
	maxLen := 128
	if len(data) > maxLen {
		return data[:maxLen] + "..."
	}
	return data
}
