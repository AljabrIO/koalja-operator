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

	"github.com/AljabrIO/koalja-operator/pkg/task/filesplit"
)

var (
	cmdFileSplit = &cobra.Command{
		Use:   "filesplit",
		Run:   cmdFileSplitRun,
		Short: "Run FileSplit task executor",
		Long:  "Run FileSplit task executor",
	}
	fileSplit filesplit.Config
)

func init() {
	cmdMain.AddCommand(cmdFileSplit)

	cmdFileSplit.Flags().StringVar(&fileSplit.SourcePath, "source", "", "Path of file to split")
	cmdFileSplit.Flags().StringVar(&fileSplit.OutputName, "output-name", "", "Name of output of the task we're serving")
}

func cmdFileSplitRun(cmd *cobra.Command, args []string) {
	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		cliLog.Fatal().Err(err).Msg("Failed to get kubernetes API server config")
	}

	// Create a new filesplit service
	svc, err := filesplit.NewService(fileSplit, cliLog, cfg)
	if err != nil {
		cliLog.Fatal().Err(err).Msg("Failed to create filesplit service")
	}

	cliLog.Info().Msg("Starting the filesplit task.")

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
