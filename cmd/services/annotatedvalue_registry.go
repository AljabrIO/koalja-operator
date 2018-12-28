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

	"github.com/AljabrIO/koalja-operator/pkg/annotatedvalue/registry"
	"github.com/AljabrIO/koalja-operator/pkg/annotatedvalue/registry/arangodb"
	"github.com/AljabrIO/koalja-operator/pkg/annotatedvalue/registry/stub"
	"github.com/spf13/cobra"
)

var (
	cmdAnnotated = &cobra.Command{
		Use: "annotated",
		Run: cmdUsage,
	}
	cmdAnnotatedValue = &cobra.Command{
		Use: "value",
		Run: cmdUsage,
	}
	cmdAnnotatedValueRegistry = &cobra.Command{
		Use:   "registry",
		Run:   cmdAnnotatedValueRegistryRun,
		Short: "Run annotated value registry",
		Long:  "Run annotated value registry",
	}
	registryOptions struct {
		avRegistryType string
		arangodb       arangodb.Config
	}
)

func init() {
	cmdMain.AddCommand(cmdAnnotated)
	cmdAnnotated.AddCommand(cmdAnnotatedValue)
	cmdAnnotatedValue.AddCommand(cmdAnnotatedValueRegistry)
	cmdAnnotatedValueRegistry.Flags().StringVar(&registryOptions.avRegistryType, "registry", "", "Set registry type: stub|arangodb")
	cmdAnnotatedValueRegistry.Flags().StringVar(&registryOptions.arangodb.Username, "arangodb-username", "", "ArangoDB username")
	cmdAnnotatedValueRegistry.Flags().StringVar(&registryOptions.arangodb.Password, "arangodb-password", "", "ArangoDB password")
	cmdAnnotatedValueRegistry.Flags().StringVar(&registryOptions.arangodb.Database, "arangodb-database", "av_registry", "ArangoDB database")
	cmdAnnotatedValueRegistry.Flags().StringSliceVar(&registryOptions.arangodb.Endpoints, "arangodb-endpoint", nil, "ArangoDB endpoints")
}

func cmdAnnotatedValueRegistryRun(cmd *cobra.Command, args []string) {
	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		cliLog.Fatal().Err(err).Msg("Failed to get kubernetes API server config")
	}

	// Create a new Cmd to provide shared dependencies and start components
	var apiBuilder registry.APIBuilder
	switch registryOptions.avRegistryType {
	case "stub":
		apiBuilder = stub.NewStub(cliLog.With().Str("component", "stub").Logger())
	case "arangodb":
		apiBuilder = arangodb.NewArangoDB(cliLog.With().Str("component", "arangodb").Logger(), registryOptions.arangodb)
	default:
		cliLog.Fatal().Str("registry", registryOptions.avRegistryType).Msg("Unknown registry type")
	}
	svc, err := registry.NewService(cfg, apiBuilder)
	if err != nil {
		log.Fatal(err)
	}

	cliLog.Info().Msg("Starting the annotated value registry.")

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
