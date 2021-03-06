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

	"github.com/AljabrIO/koalja-operator/pkg/task/jsonquery"
)

var (
	cmdJSONQuery = &cobra.Command{
		Use:   "jsonquery",
		Run:   cmdJSONQueryRun,
		Short: "Run JSONQuery task executor",
		Long:  "Run JSONQuery task executor",
	}
	jsonQuery jsonquery.Config
)

func init() {
	cmdMain.AddCommand(cmdJSONQuery)

	cmdJSONQuery.Flags().StringVar(&jsonQuery.SourcePath, "source", "", "Path of file to split")
	cmdJSONQuery.Flags().StringVar(&jsonQuery.Query, "query", "", "Query to apply")
	cmdJSONQuery.Flags().StringVar(&jsonQuery.OutputName, "output-name", "", "Name of output of the task we're serving")
}

func cmdJSONQueryRun(cmd *cobra.Command, args []string) {
	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		cliLog.Fatal().Err(err).Msg("Failed to get kubernetes API server config")
	}

	// Create a new jsonquery service
	svc, err := jsonquery.NewService(jsonQuery, cliLog, cfg)
	if err != nil {
		cliLog.Fatal().Err(err).Msg("Failed to create jsonquery service")
	}

	cliLog.Info().Msg("Starting the jsonquery task.")

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
