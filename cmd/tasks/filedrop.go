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
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"

	_ "github.com/AljabrIO/koalja-operator/pkg/agent/task/protocols"
	"github.com/AljabrIO/koalja-operator/pkg/task/filedrop"
)

var (
	cmdFileDrop = &cobra.Command{
		Use:   "filedrop",
		Run:   cmdFileDropRun,
		Short: "Run FileDrop task executor",
		Long:  "Run FileDrop task executor",
	}
)

func init() {
	cmdMain.AddCommand(cmdFileDrop)
}

func cmdFileDropRun(cmd *cobra.Command, args []string) {
	// Create a new filedrop service
	svc, err := filedrop.NewService()
	if err != nil {
		cliLog.Fatal().Err(err).Msg("Failed to create filedrop service")
	}

	cliLog.Info().Msg("Starting the filedrop task.")

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
