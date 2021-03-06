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

package rest

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/textproto"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/rs/zerolog"
	"github.com/vincent-petithory/dataurl"
	"k8s.io/client-go/rest"

	"github.com/AljabrIO/koalja-operator/pkg/task"
	taskclient "github.com/AljabrIO/koalja-operator/pkg/task/client"
	"github.com/AljabrIO/koalja-operator/pkg/util"
)

// Config holds the configuration arguments of the service.
type Config struct {
	OutputName      string
	URLTemplate     string
	MethodTemplate  string
	BodyTemplate    string
	HeadersTemplate string
}

// Service loop of the REST task.
type Service struct {
	Config
	log        zerolog.Logger
	snsClient  taskclient.SnapshotServiceClient
	ornClient  taskclient.OutputReadyNotifierClient
	httpClient *http.Client
}

// NewService initializes a new service.
func NewService(cfg Config, log zerolog.Logger, config *rest.Config) (*Service, error) {
	// Check arguments
	if cfg.OutputName == "" {
		return nil, fmt.Errorf("OutputName expected")
	}
	if cfg.URLTemplate == "" {
		return nil, fmt.Errorf("URLTemplate expected")
	}
	if cfg.MethodTemplate == "" {
		return nil, fmt.Errorf("MethodTemplate expected")
	}

	// Create service clients
	snsClient, err := taskclient.CreateSnapshotServiceClient()
	if err != nil {
		log.Error().Err(err).Msg("Failed to create snapshot service client")
		return nil, maskAny(err)
	}
	ornClient, err := taskclient.CreateOutputReadyNotifierClient()
	if err != nil {
		log.Error().Err(err).Msg("Failed to create output ready notifier client")
		return nil, maskAny(err)
	}
	return &Service{
		Config:     cfg,
		log:        log,
		snsClient:  snsClient,
		ornClient:  ornClient,
		httpClient: http.DefaultClient,
	}, nil
}

// Run th service until the given context is canceled
func (s *Service) Run(ctx context.Context) error {
	log := s.log

	minDelay := time.Nanosecond
	delay := minDelay
	for {
		// Check context
		if err := ctx.Err(); err != nil {
			return err
		}

		// Fetch next snapshot
		resp, err := s.snsClient.Next(ctx, &task.NextRequest{
			WaitTimeout: ptypes.DurationProto(time.Second * 30),
		})
		if err != nil {
			log.Warn().Err(err).Msg("Next request to SnapshotService failed")
			delay = util.Backoff(delay, 1.5, time.Second*5)
		} else {
			delay = minDelay
		}

		if snapshot := resp.GetSnapshot(); snapshot != nil {
			// Execute REST call on snapshot
			if err := s.processSnapshot(ctx, snapshot); err != nil {
				log.Warn().Err(err).Msg("Failed to process snapshot")
			}
			// Success or permanent failure, ack the snapshot
			if _, err := s.snsClient.Ack(ctx, &task.AckRequest{
				SnapshotID: snapshot.GetID(),
			}); err != nil {
				log.Warn().Err(err).Msg("Ack request to SnapshotService failed")
			}
		}

		// Wait a bit and continue
		if delay > minDelay {
			select {
			case <-time.After(delay):
				// Continue
			case <-ctx.Done():
				// Context canceled
				return ctx.Err()
			}
		}
	}
}

// processSnapshot prepares and calls the REST call for the given snapshot.
// It notifies the output publisher of the results.
func (s *Service) processSnapshot(ctx context.Context, snapshot *task.Snapshot) error {
	// Prepare request
	req, err := s.buildRequest(ctx, snapshot)
	if err != nil {
		return maskAny(err)
	}

	// Execute request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return maskAny(err)
	}

	// Process results
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return maskAny(err)
	}

	// Check status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		s.log.Debug().
			Int("statusCode", resp.StatusCode).
			Str("body", bodyAsString(body, 512)).
			Msg("Request returned invalid response status")
		return maskAny(fmt.Errorf("Invalid response status %d", resp.StatusCode))
	}

	// Put result into data URI
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "text/plain"
	}
	dataURI := dataurl.New(body, contentType)
	if _, err := s.outputReady(ctx, snapshot, dataURI.String(), s.Config.OutputName); err != nil {
		return maskAny(err)
	}

	return nil
}

// bodyAsString converts the given body into a string with a given max-length.
func bodyAsString(body []byte, maxLen int) string {
	if len(body) > maxLen {
		body = body[:maxLen]
	}
	return string(body)
}

// buildRequest prepares REST call for the given snapshot.
func (s *Service) buildRequest(ctx context.Context, snapshot *task.Snapshot) (*http.Request, error) {
	// Prepare request
	method, err := s.executeTemplate(ctx, s.MethodTemplate, snapshot)
	if err != nil {
		return nil, maskAny(err)
	}
	url, err := s.executeTemplate(ctx, s.URLTemplate, snapshot)
	if err != nil {
		return nil, maskAny(err)
	}
	body, err := s.executeTemplate(ctx, s.BodyTemplate, snapshot)
	if err != nil {
		return nil, maskAny(err)
	}
	headers, err := s.executeTemplate(ctx, s.HeadersTemplate, snapshot)
	if err != nil {
		return nil, maskAny(err)
	}
	var hdr textproto.MIMEHeader
	if len(strings.TrimSpace(string(headers))) > 0 {
		// Parse header
		rd := textproto.NewReader(bufio.NewReader(bytes.NewReader(headers)))
		hdr, err = rd.ReadMIMEHeader()
		if err != nil {
			s.log.Debug().Str("header", string(headers)).Err(err).Msg("ReadMIMEHeader failed")
			return nil, maskAny(err)
		}
	}

	// Build request
	req, err := http.NewRequest(string(method), string(url), bytes.NewReader(body))
	if err != nil {
		s.log.Debug().Err(err).Msg("NewRequest failed")
		return nil, maskAny(err)
	}

	// Put in header (if any)
	if hdr != nil {
		for k, v := range hdr {
			req.Header[k] = v
		}
	}

	// Debug
	s.log.Debug().Str("body", bodyAsString(body, 512)).Msg("Prepared HTTP request")

	return req, nil
}

// executeTemplate executes a given template with given data into a byte buffer.
func (s *Service) executeTemplate(ctx context.Context, templateSource string, snapshot *task.Snapshot) ([]byte, error) {
	if len(strings.TrimSpace(templateSource)) == 0 {
		return nil, nil
	}
	resp, err := s.snsClient.ExecuteTemplate(ctx, &task.ExecuteTemplateRequest{
		Snapshot: snapshot,
		Template: templateSource,
	})
	if err != nil {
		return nil, maskAny(err)
	}
	return resp.GetResult(), nil
}

const (
	maxOutputReadyFailures = 10
)

// outputReady is to be called by a service runner for publishing output notifications.
// Returns: (annotatedValueID, error)
func (s *Service) outputReady(ctx context.Context, snapshot *task.Snapshot, annotatedValueData, outputName string) (string, error) {
	delay := time.Millisecond * 100
	recentFailures := 0
	for {
		if resp, err := s.ornClient.OutputReady(ctx, &task.OutputReadyRequest{
			AnnotatedValueData: annotatedValueData,
			OutputName:         outputName,
		}); err != nil {
			recentFailures++
			if recentFailures > maxOutputReadyFailures {
				s.log.Error().Err(err).Msg("OutputReady failed too many times")
				return "", maskAny(err)
			}
			s.log.Debug().Err(err).Msg("OutputReady attempt failed")
		} else if resp.Accepted {
			// Output was accepted
			return resp.GetAnnotatedValueID(), nil
		} else {
			// OutputReady call succeeded, but output was not (yet) accepted
			s.log.Debug().Err(err).Msg("OutputReady did not accept our value. Wait and try again...")
			recentFailures = 0
		}
		// Output was not accepted, or call failed, try again soon.
		select {
		case <-time.After(delay):
			// Try again
			delay = util.Backoff(delay, 1.5, time.Minute)
		case <-ctx.Done():
			// Context canceled
			s.log.Debug().Err(ctx.Err()).Msg("Context canceled during outputReady")
			return "", ctx.Err()
		}
	}
}
