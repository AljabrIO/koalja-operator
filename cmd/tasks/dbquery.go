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

	"github.com/AljabrIO/koalja-operator/pkg/task/dbquery"
)

var (
	cmdDBQuery = &cobra.Command{
		Use:   "dbquery",
		Run:   cmdDBQueryRun,
		Short: "Run DBQuery task executor",
		Long:  "Run DBQuery task executor",
	}
	dbQuery dbquery.Config
)

func init() {
	cmdMain.AddCommand(cmdDBQuery)

	cmdDBQuery.Flags().StringVar(&dbQuery.LargeDataFolder, "target", "", "Directory to store large data files into")
	cmdDBQuery.Flags().StringVar(&dbQuery.VolumeName, "volume-name", "", "Name of volume containing target")
	cmdDBQuery.Flags().StringVar(&dbQuery.MountPath, "mount-path", "", "Mount path of volume containing target")
	cmdDBQuery.Flags().StringVar(&dbQuery.NodeName, "node-name", "", "Name of node we're running on")
	cmdDBQuery.Flags().StringVar(&dbQuery.OutputName, "output-name", "", "Name of output of the task we're serving")
	cmdDBQuery.Flags().StringVar(&dbQuery.DatabaseConfigMap, "database-config-map", "", "Name of the ConfigMap containing type & access to database to query")
	cmdDBQuery.Flags().StringVar(&dbQuery.Query, "query", "", "Query to execute")
}

func cmdDBQueryRun(cmd *cobra.Command, args []string) {
	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		cliLog.Fatal().Err(err).Msg("Failed to get kubernetes API server config")
	}

	// Setup Scheme for all resources
	scheme := scheme.Scheme

	// Create a new dbquery service
	svc, err := dbquery.NewService(dbQuery, cliLog, cfg, scheme)
	if err != nil {
		cliLog.Fatal().Err(err).Msg("Failed to create dbquery service")
	}

	cliLog.Info().Msg("Starting the dbquery task.")

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
