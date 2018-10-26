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
	"log"

	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"

	"github.com/AljabrIO/koalja-operator/pkg/event/registry"
	"github.com/AljabrIO/koalja-operator/pkg/event/registry/stub"
	"github.com/spf13/cobra"
)

var (
	cmdEvent = &cobra.Command{
		Use: "event",
		Run: cmdUsage,
	}
	cmdEventRegistry = &cobra.Command{
		Use:   "registry",
		Run:   cmdEventRegistryRun,
		Short: "Run event registry",
		Long:  "Run event registry",
	}
	eventRegistryType string
)

func init() {
	cmdMain.AddCommand(cmdEvent)
	cmdEvent.AddCommand(cmdEventRegistry)
	cmdEventRegistry.Flags().StringVar(&eventRegistryType, "registry", "", "Set registry type: stub")
}

func cmdEventRegistryRun(cmd *cobra.Command, args []string) {
	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		cliLog.Fatal().Err(err).Msg("Failed to get kubernetes API server config")
	}

	// Create a new Cmd to provide shared dependencies and start components
	var apiBuilder registry.APIBuilder
	switch eventRegistryType {
	case "stub":
		apiBuilder = stub.NewStub(cliLog.With().Str("component", "stub").Logger())
	default:
		cliLog.Fatal().Str("registry", eventRegistryType).Msg("Unknown registry type")
	}
	svc, err := registry.NewService(cfg, apiBuilder)
	if err != nil {
		log.Fatal(err)
	}

	cliLog.Info().Msg("Starting the event registry.")

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
