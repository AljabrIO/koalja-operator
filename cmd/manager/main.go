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

	contourapi "github.com/heptio/contour/apis/contour/v1beta1"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"

	"github.com/AljabrIO/koalja-operator/pkg/apis"
	"github.com/AljabrIO/koalja-operator/pkg/constants"
	"github.com/AljabrIO/koalja-operator/pkg/controller"
	"github.com/AljabrIO/koalja-operator/pkg/controller/pipeline"
	"github.com/AljabrIO/koalja-operator/pkg/controller/pipeline/contour"
	"github.com/AljabrIO/koalja-operator/pkg/util"
	"github.com/AljabrIO/koalja-operator/pkg/util/retry"
)

var (
	cliLog = util.MustCreateLogger()
)

func main() {
	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		cliLog.Fatal().Err(err).Msg("Failed to get kubernetes API server config")
	}

	// Create a new Cmd to provide shared dependencies and start components
	var mgr manager.Manager
	ctx := context.Background()
	if err := retry.Do(ctx, func(ctx context.Context) error {
		var err error
		mgr, err = manager.New(cfg, manager.Options{})
		return err
	}, retry.Timeout(constants.TimeoutK8sClient)); err != nil {
		cliLog.Fatal().Err(err).Msg("Failed to create manager")
	}

	cliLog.Info().Msg("Registering Components.")

	// Setup Scheme for all resources
	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		cliLog.Fatal().Err(err).Msg("Failed to add API to scheme")
	}
	if err := contourapi.AddToScheme(mgr.GetScheme()); err != nil {
		cliLog.Fatal().Err(err).Msg("Failed to add Contour to scheme")
	}

	// Setup network reconciler
	// TODO make this configurable
	if networkReconciler, err := contour.New(mgr); err != nil {
		cliLog.Fatal().Err(err).Msg("Failed to create Contour Network Reconciler")
	} else {
		pipeline.SetGlobalNetworkReconciler(networkReconciler)
	}

	// Setup all Controllers
	if err := controller.AddToManager(mgr); err != nil {
		cliLog.Fatal().Err(err).Msg("Failed to add controller to manager")
	}

	cliLog.Info().Msg("Starting the manager.")

	// Start the Cmd
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		cliLog.Fatal().Err(err).Msg("Service failed")
	}
}
