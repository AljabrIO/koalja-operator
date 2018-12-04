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
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var (
	cmdUnmount = &cobra.Command{
		Use: "unmount",
		Run: cmdUnmountRun,
	}
)

func init() {
	cmdMain.AddCommand(cmdUnmount)
}

// unmount <mount dir> <json options>
func cmdUnmountRun(cmd *cobra.Command, args []string) {
	if len(args) < 1 {
		sendOutput(FlexOutput{Status: FlexStatusFailure, Message: "Not enough arguments"})
		os.Exit(1)
	}
	mountDir := args[0]

	// Check is mountDir is actually mounted
	isMounted, err := isMounted(mountDir)
	if err == nil && !isMounted {
		// umount is not needed
		sendOutput(FlexOutput{Status: FlexStatusSuccess})
		os.Exit(0)
	}

	// Call umount
	c := exec.Command("umount", mountDir)
	output, err := c.CombinedOutput()
	if err != nil {
		sendOutput(FlexOutput{Status: FlexStatusFailure, Message: fmt.Sprintf("Failed to unmount because: %s %s", err, string(output))})
		os.Exit(1)
	}
	// Unmount succeeded
	sendOutput(FlexOutput{Status: FlexStatusSuccess})
}
