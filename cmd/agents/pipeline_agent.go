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
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"

	"github.com/AljabrIO/koalja-operator/pkg/agent/pipeline"
	"github.com/AljabrIO/koalja-operator/pkg/apis"
)

var (
	cmdPipelineAgent = &cobra.Command{
		Use:   "pipeline",
		Run:   cmdPipelineAgentRun,
		Short: "Run pipeline agent",
		Long:  "Run pipeline agent",
	}
)

func init() {
	cmdMain.AddCommand(cmdPipelineAgent)
}

func cmdPipelineAgentRun(cmd *cobra.Command, args []string) {
	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		cliLog.Fatal().Err(err).Msg("Failed to get kubernetes API server config")
	}

	// Setup Scheme for all resources
	scheme := scheme.Scheme
	if err := apis.AddToScheme(scheme); err != nil {
		cliLog.Fatal().Err(err).Msg("Failed to add API to scheme")
	}

	// Create a new Cmd to provide shared dependencies and start components
	svc, err := pipeline.NewService(cliLog, cfg, scheme)
	if err != nil {
		cliLog.Fatal().Err(err).Msg("Failed to create pipeline service")
	}

	cliLog.Info().Msg("Starting the pipeline agent.")

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
