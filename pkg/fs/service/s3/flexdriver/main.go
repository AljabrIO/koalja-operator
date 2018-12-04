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
	"github.com/spf13/cobra"

	"github.com/AljabrIO/koalja-operator/pkg/constants"
	"github.com/AljabrIO/koalja-operator/pkg/util"
)

const (
	vendorName = constants.FlexVolumeS3VendorName
	driverName = constants.FlexVolumeS3DriverName
)

var (
	cliLog  = util.MustCreateLogger()
	cmdMain = &cobra.Command{
		Use: driverName,
		Run: cmdUsage,
	}
	notSupportedCommands = []string{"attach", "detach", "waitforattach", "isattached", "mountdevice", "unmountdevice"}
)

func main() {
	cmdMain.Execute()
	for _, use := range notSupportedCommands {
		cmd := &cobra.Command{
			Use: use,
			Run: cmdNotSupported,
		}
		cmdMain.AddCommand(cmd)
	}
}

func cmdUsage(cmd *cobra.Command, args []string) {
	cmd.Usage()
}

func cmdNotSupported(cmd *cobra.Command, args []string) {
	sendOutput(FlexOutput{Status: FlexStatusNotSupported})
}
