//
// Copyright © 2018 Aljabr, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	cmdInstall = &cobra.Command{
		Use: "install",
		Run: cmdInstallRun,
	}
	installFlags struct {
		driverFolder string
	}
)

const (
	defaultVolumePluginDir = "/usr/libexec/kubernetes/kubelet-plugins/volume/exec/"
)

func init() {
	cmdMain.AddCommand(cmdInstall)

	defDir := os.Getenv("FLEX_VOLUME_PLUGIN_DIR")
	if defDir == "" {
		defDir = defaultVolumePluginDir
	}
	cmdInstall.Flags().StringVar(&installFlags.driverFolder, "driver-folder", defDir, "Flex Volume driver folder")
}

// Install this flex driver.
func cmdInstallRun(cmd *cobra.Command, args []string) {
	executable, err := os.Executable()
	if err != nil {
		cliLog.Fatal().Err(err).Msg("Failed to find executable path")
	}

	// Ensure driver dir exists
	driverDir := filepath.Join(installFlags.driverFolder, fmt.Sprintf("%s~%s", vendorName, driverName))
	if err := os.MkdirAll(driverDir, 0755); err != nil {
		cliLog.Fatal().Err(err).Msg("Failed to ensure driver folder exists")
	}

	// Open executable file
	f, err := os.Open(executable)
	if err != nil {
		cliLog.Fatal().Err(err).Msg("Failed to open executable file")
	}
	defer f.Close()

	// Copy to tmp path
	tmpPath := filepath.Join(driverDir, ".tmp_"+driverName)
	dst, err := os.Create(tmpPath)
	if err != nil {
		cliLog.Fatal().Err(err).Msg("Failed to temporary executable file")
	}
	_, err = io.Copy(dst, f)
	dst.Close()
	if err != nil {
		cliLog.Fatal().Err(err).Msg("Failed to copy to temporary executable file")
	}
	if err := os.Chmod(tmpPath, 0755); err != nil {
		cliLog.Fatal().Err(err).Msg("Failed to set permissions on temporary executable file")
	}

	// Atomically move tmp executable to target executable
	targetPath := filepath.Join(driverDir, driverName)
	if err := os.Rename(tmpPath, targetPath); err != nil {
		cliLog.Fatal().Err(err).Msg("Failed to move to temporary executable file to target path")
	}

	// Wait forever
	<-context.Background().Done()
}
