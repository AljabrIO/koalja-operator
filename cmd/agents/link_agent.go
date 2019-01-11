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

	"github.com/spf13/cobra"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"

	link "github.com/AljabrIO/koalja-operator/pkg/agent/link"
	"github.com/AljabrIO/koalja-operator/pkg/agent/link/arangodb"
	"github.com/AljabrIO/koalja-operator/pkg/agent/link/stub"
)

var (
	cmdLinkAgent = &cobra.Command{
		Use:   "link",
		Run:   cmdLinkAgentRun,
		Short: "Run link agent",
		Long:  "Run link agent",
	}
	linkAgentOptions struct {
		queueType string
		arangodb  arangodb.Config
	}
)

func init() {
	cmdMain.AddCommand(cmdLinkAgent)
	cmdLinkAgent.Flags().StringVar(&linkAgentOptions.queueType, "queue", "", "Set queue type: stub|arangodb")
	cmdLinkAgent.Flags().StringVar(&linkAgentOptions.arangodb.Username, "arangodb-username", "", "ArangoDB username")
	cmdLinkAgent.Flags().StringVar(&linkAgentOptions.arangodb.Password, "arangodb-password", "", "ArangoDB password")
	cmdLinkAgent.Flags().StringVar(&linkAgentOptions.arangodb.Database, "arangodb-database", "links", "ArangoDB database")
	cmdLinkAgent.Flags().StringSliceVar(&linkAgentOptions.arangodb.Endpoints, "arangodb-endpoint", nil, "ArangoDB endpoints")
}

func cmdLinkAgentRun(cmd *cobra.Command, args []string) {
	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		cliLog.Fatal().Err(err).Msg("Failed to get kubernetes API server config")
	}

	// Create a new Cmd to provide shared dependencies and start components
	var apiBuilder link.APIBuilder
	switch linkAgentOptions.queueType {
	case "stub":
		apiBuilder = stub.NewStub(cliLog.With().Str("component", "stub").Logger())
	case "arangodb":
		apiBuilder = arangodb.NewArangoDB(cliLog.With().Str("component", "arangodb").Logger(), linkAgentOptions.arangodb)
	default:
		cliLog.Fatal().Str("queue", linkAgentOptions.queueType).Msg("Unsupport queue type")
	}
	svc, err := link.NewService(cliLog, cfg, apiBuilder)
	if err != nil {
		cliLog.Fatal().Err(err).Msg("Failed to create link service")
	}

	cliLog.Info().Msg("Starting the link agent.")

	// Start the Cmd
	ctx, done := context.WithCancel(context.Background())
	go func() {
		<-signals.SetupSignalHandler()
		done()
	}()
	if err := svc.Run(ctx); err != nil {
		cliLog.Fatal().Err(err).Msg("Service failed")
	}
}
