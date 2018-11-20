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
	"os"

	"github.com/AljabrIO/koalja-operator/pkg/constants"

	"github.com/spf13/cobra"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"

	fssvc "github.com/AljabrIO/koalja-operator/pkg/fs/service"
	"github.com/AljabrIO/koalja-operator/pkg/fs/service/local"
	"github.com/AljabrIO/koalja-operator/pkg/fs/service/local/node"
)

// TODO: Add cleanup of files

var (
	cmdFileSystem = &cobra.Command{
		Use:   "filesystem",
		Run:   cmdFileSystemRun,
		Short: "Run filesystem service",
		Long:  "Run filesystem service",
	}
	cmdFileSystemNode = &cobra.Command{
		Use:   "node",
		Run:   cmdFileSystemNodeRun,
		Short: "Run filesystem node daemon",
		Long:  "Run filesystem node daemon",
	}
	fileSystemOptions struct {
		Type             string
		StorageClassName string
		LocalPathPrefix  string
	}
	fileSystemNodeOptions node.Config
)

func init() {
	cmdMain.AddCommand(cmdFileSystem)
	cmdFileSystem.AddCommand(cmdFileSystemNode)

	cmdFileSystem.Flags().StringVar(&fileSystemOptions.Type, "filesystem", "", "Set filesystem type: local")
	cmdFileSystem.Flags().StringVar(&fileSystemOptions.StorageClassName, "storageClassName", "", "Name of the StorageClass")
	cmdFileSystem.Flags().StringVar(&fileSystemOptions.LocalPathPrefix, "localPathPrefix", "/var/lib/koalja/local-fs", "Path prefix on nodes for volume storage")

	cmdFileSystemNode.Flags().IntVar(&fileSystemNodeOptions.Port, "port", 8080, "Port to listen on")
	cmdFileSystemNode.Flags().StringVar(&fileSystemNodeOptions.RegistryAddress, "registry", "", "Address of the Node Registry")
}

func cmdFileSystemRun(cmd *cobra.Command, args []string) {
	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		cliLog.Fatal().Err(err).Msg("Failed to get kubernetes API server config")
	}

	// Create a new Cmd to provide shared dependencies and start components
	var apiBuilder fssvc.APIBuilder
	switch fileSystemOptions.Type {
	case "local":
		if fileSystemOptions.StorageClassName == "" {
			fileSystemOptions.StorageClassName = "koalja-local-storage"
		}
		apiBuilder = local.NewLocalFileSystemBuilder(cliLog, fileSystemOptions.LocalPathPrefix, fileSystemOptions.StorageClassName)
	default:
		cliLog.Fatal().Str("filesystem", fileSystemOptions.Type).Msg("Unknown filesystem type")
	}
	svc, err := fssvc.NewService(cfg, apiBuilder)
	if err != nil {
		cliLog.Fatal().Err(err).Msg("Failed to create FileSystem service")
	}

	cliLog.Info().Msg("Starting the filesystem service.")

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

func cmdFileSystemNodeRun(cmd *cobra.Command, args []string) {
	// Fetch node name
	fileSystemNodeOptions.NodeName = os.Getenv(constants.EnvNodeName)

	svc, err := node.NewService(cliLog, fileSystemNodeOptions)
	if err != nil {
		cliLog.Fatal().Err(err).Msg("Failed to create FileSystem Node Daemon Service")
	}

	cliLog.Info().Msg("Starting the filesystem node daemon.")

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
