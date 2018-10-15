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

package defaultlink

//go:generate protoc -I ..:../../../../vendor --go_out=plugins=grpc:.. ../agent_api.proto

import (
	"context"
	fmt "fmt"
	"log"
	"net"
	"sort"
	"strconv"

	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/AljabrIO/koalja-operator/pkg/agent/link"
	agentsv1alpha1 "github.com/AljabrIO/koalja-operator/pkg/apis/agents/v1alpha1"
	"github.com/AljabrIO/koalja-operator/pkg/constants"
)

// Service implements the link agent.
type Service struct {
	client.Client
	Namespace string
}

// NewService creates a new Service instance.
func NewService(config *rest.Config) (*Service, error) {
	client, err := client.New(config, client.Options{})
	if err != nil {
		return nil, err
	}
	ns, err := constants.GetNamespace()
	if err != nil {
		return nil, err
	}
	return &Service{
		Client:    client,
		Namespace: ns,
	}, nil
}

// Run the pipeline agent until the given context is canceled.
func (s *Service) Run(ctx context.Context) error {
	port, err := constants.GetAPIPort()
	if err != nil {
		return err
	}
	lis, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	svr := grpc.NewServer()
	link.RegisterAgentServer(svr, s)
	// Register reflection service on gRPC server.
	reflection.Register(svr)
	go func() {
		if err := svr.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()
	<-ctx.Done()
	svr.GracefulStop()
	return nil
}

// CreateSourceSidecar returns a filled out Container that is injected
// into a Task Execution Pod to serve as link endpoint for the source
// of this link.
func (s *Service) CreateSourceSidecar(ctx context.Context, req *link.SidecarRequest) (*link.SidecarResponse, error) {
	ep, err := s.getLinkSidecar(ctx, req.GetProtocol(), req.GetFormat())
	if err != nil {
		return nil, err
	}
	c := ep.Spec.Container.DeepCopy()
	setContainerDefaults(c)
	setContainerEnvVars(c, map[string]string{
		constants.EnvAPIPort:  strconv.Itoa(constants.LinkSidecarAPIPort),
		constants.EnvProtocol: req.GetProtocol(),
		constants.EnvFormat:   req.GetFormat(),
		constants.EnvLinkSide: constants.LinkSideSource,
	})
	return &link.SidecarResponse{
		Container: c,
	}, nil
}

// CreateDestinationSidecar returns a filled out Container that is injected
// into a Task Execution Pod to serve as link endpoint for the destination
// of this link.
func (s *Service) CreateDestinationSidecar(ctx context.Context, req *link.SidecarRequest) (*link.SidecarResponse, error) {
	ep, err := s.getLinkSidecar(ctx, req.GetProtocol(), req.GetFormat())
	if err != nil {
		return nil, err
	}
	c := ep.Spec.Container.DeepCopy()
	setContainerDefaults(c)
	setContainerEnvVars(c, map[string]string{
		constants.EnvAPIPort:  strconv.Itoa(constants.LinkSidecarAPIPort),
		constants.EnvProtocol: req.GetProtocol(),
		constants.EnvFormat:   req.GetFormat(),
		constants.EnvLinkSide: constants.LinkSideDestination,
	})
	return &link.SidecarResponse{
		Container: c,
	}, nil
}

// getLinkSidecar selects the sidecar that matches the given protocol & format best.
func (s *Service) getLinkSidecar(ctx context.Context, protocol, format string) (*agentsv1alpha1.LinkSidecar, error) {
	var list agentsv1alpha1.LinkSidecarList
	if err := s.List(ctx, &client.ListOptions{Namespace: s.Namespace}, &list); err != nil {
		return nil, err
	}
	// Find match
	var valid []agentsv1alpha1.LinkSidecar
	for _, lep := range list.Items {
		// Protocol must always match
		if protocol != lep.Spec.Protocol {
			continue
		}
		// If specified, format must match
		if format != "" && lep.Spec.Format != "" && format != lep.Spec.Format {
			continue
		}
		valid = append(valid, lep)
	}
	if len(valid) == 0 {
		// No valid endpoints found
		return nil, fmt.Errorf("No endpoint found for protocol '%s' and format '%s'", protocol, format)
	}
	// Order valid items such that the most specific is in front
	sort.Slice(valid, func(i, j int) bool {
		a, b := valid[i], valid[j]
		return a.Spec.Format != "" && b.Spec.Format == ""
	})
	// Take the first valid endpoint
	return &valid[0], nil
}

// setContainerDefaults applies default values to a container used to run a link sidecar
func setContainerDefaults(c *corev1.Container) {
	if c.Name == "" {
		c.Name = "endpoint"
	}
	if len(c.Ports) == 0 {
		c.Ports = []corev1.ContainerPort{
			corev1.ContainerPort{
				Name:          "api",
				ContainerPort: constants.LinkSidecarAPIPort,
			},
		}
	}
}

// setContainerEnvVars sets the given environment variables into the given container
func setContainerEnvVars(c *corev1.Container, vars map[string]string) {
	for k, v := range vars {
		found := false
		for i, ev := range c.Env {
			if ev.Name == k {
				c.Env[i].Value = v
				c.Env[i].ValueFrom = nil
				found = true
				break
			}
		}
		if !found {
			c.Env = append(c.Env, corev1.EnvVar{
				Name:  k,
				Value: v,
			})
		}
	}
	c.Env = append(c.Env, corev1.EnvVar{
		Name: constants.EnvNamespace,
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: "metadata.namespace",
			},
		},
	})
}
