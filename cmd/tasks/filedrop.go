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

	"github.com/AljabrIO/koalja-operator/pkg/task/filedrop"
)

var (
	cmdFileDrop = &cobra.Command{
		Use:   "filedrop",
		Run:   cmdFileDropRun,
		Short: "Run FileDrop task executor",
		Long:  "Run FileDrop task executor",
	}
	fileDrop filedrop.Config
)

func init() {
	cmdMain.AddCommand(cmdFileDrop)

	cmdFileDrop.Flags().StringVar(&fileDrop.DropFolder, "target", "", "Directory to drop files into")
	cmdFileDrop.Flags().StringVar(&fileDrop.VolumeName, "volume-name", "", "Name of volume containing target")
	cmdFileDrop.Flags().StringVar(&fileDrop.VolumeClaimName, "volume-claim-name", "", "Name of volume claim containing target")
	cmdFileDrop.Flags().StringVar(&fileDrop.VolumePath, "volume-path", "", "Path of volume containing target")
	cmdFileDrop.Flags().StringVar(&fileDrop.SubPath, "sub-path", "", "SubPath on the volume containing target")
	cmdFileDrop.Flags().StringVar(&fileDrop.MountPath, "mount-path", "", "Mount path of volume containing target")
	cmdFileDrop.Flags().StringVar(&fileDrop.NodeName, "node-name", "", "Name of node we're running on")
	cmdFileDrop.Flags().StringVar(&fileDrop.OutputName, "output-name", "", "Name of output of the task we're serving")
}

func cmdFileDropRun(cmd *cobra.Command, args []string) {
	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		cliLog.Fatal().Err(err).Msg("Failed to get kubernetes API server config")
	}

	// Setup Scheme for all resources
	scheme := scheme.Scheme

	// Create a new filedrop service
	svc, err := filedrop.NewService(fileDrop, cliLog, cfg, scheme)
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
