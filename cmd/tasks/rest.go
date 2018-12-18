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

	"github.com/AljabrIO/koalja-operator/pkg/task/rest"
	"github.com/spf13/cobra"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"
)

var (
	cmdRest = &cobra.Command{
		Use:   "rest",
		Run:   cmdRestRun,
		Short: "Run REST queries",
		Long:  "Run REST queries",
	}
	restOptions rest.Config
)

func init() {
	cmdMain.AddCommand(cmdRest)

	cmdRest.Flags().StringVar(&restOptions.OutputName, "output-name", "", "Name of output of the task we're serving")
	cmdRest.Flags().StringVar(&restOptions.URLTemplate, "url-template", "", "Template used to build the URL of the REST query")
	cmdRest.Flags().StringVar(&restOptions.MethodTemplate, "method-template", "", "Template used to build the method of the REST query")
	cmdRest.Flags().StringVar(&restOptions.BodyTemplate, "body-template", "", "Template used to build the body  of the REST query")
	cmdRest.Flags().StringVar(&restOptions.HeadersTemplate, "headers-template", "", "Template used to build the headers of the REST query")
}

func cmdRestRun(cmd *cobra.Command, args []string) {
	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		cliLog.Fatal().Err(err).Msg("Failed to get kubernetes API server config")
	}

	// Create a new filesplit service
	svc, err := rest.NewService(restOptions, cliLog, cfg)
	if err != nil {
		cliLog.Fatal().Err(err).Msg("Failed to create REST service")
	}

	cliLog.Info().Msg("Starting the REST task.")

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
