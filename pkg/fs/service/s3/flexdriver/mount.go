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
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/AljabrIO/koalja-operator/pkg/constants"
	goofys "github.com/kahing/goofys/api"
	daemon "github.com/sevlyar/go-daemon"
	"github.com/spf13/cobra"
)

var (
	cmdMount = &cobra.Command{
		Use: "mount",
		Run: cmdMountRun,
	}
)

const (
	mountAccessKeyKey = "kubernetes.io/secret/" + constants.SecretKeyS3AccessKey
	mountSecretKeyKey = "kubernetes.io/secret/" + constants.SecretKeyS3SecretKey
)

func init() {
	cmdMain.AddCommand(cmdMount)
}

func waitForSignal(wg *sync.WaitGroup, waitedForSignal *os.Signal) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGUSR1, syscall.SIGUSR2)

	wg.Add(1)
	go func() {
		*waitedForSignal = <-signalChan
		wg.Done()
	}()
}

func kill(pid int, s os.Signal) (err error) {
	p, err := os.FindProcess(pid)
	if err != nil {
		return err
	}

	defer p.Release()

	err = p.Signal(s)
	if err != nil {
		return err
	}
	return
}

func isMounted(mountDir string) (bool, error) {
	// findmnt -n /mnt/test --output TARGET
	c := exec.Command("findmnt", "-n", mountDir, "--output", "TARGET")
	output, err := c.CombinedOutput()
	if err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			// Not just exitcode 1
			return false, err
		}
	}
	result := strings.TrimSpace(string(output))
	return result == mountDir, nil
}

// Return the UID and GID of this process.
func myUserAndGroup() (uid int, gid int, err error) {
	// Ask for the current user.
	user, err := user.Current()
	if err != nil {
		return 0, 0, err
	}

	// Parse UID.
	uid64, err := strconv.ParseInt(user.Uid, 10, 32)
	if err != nil {
		return 0, 0, fmt.Errorf("Parsing UID (%s): %v", user.Uid, err)
	}

	// Parse GID.
	gid64, err := strconv.ParseInt(user.Gid, 10, 32)
	if err != nil {
		return 0, 0, fmt.Errorf("Parsing GID (%s): %v", user.Gid, err)
	}

	uid = int(uid64)
	gid = int(gid64)

	return uid, gid, nil
}

// mount <mount dir> <json options>
func cmdMountRun(cmd *cobra.Command, args []string) {
	if len(args) < 2 {
		sendOutput(FlexOutput{Status: FlexStatusFailure, Message: "Not enough arguments"})
		os.Exit(1)
	}
	mountDir := args[0]
	jsonOptions, err := parseJSONOptions(args[1])
	if err != nil {
		sendOutput(FlexOutput{Status: FlexStatusFailure, Message: err.Error()})
		os.Exit(1)
	}

	// Check options
	requireJSONOpt := func(key string) string {
		v, found := jsonOptions[key]
		if !found {
			sendOutput(FlexOutput{Status: FlexStatusFailure, Message: fmt.Sprintf("Missing JSON option with key '%s'", key)})
			os.Exit(1)
		}
		return v
	}
	endpoint := requireJSONOpt(constants.FlexVolumeOptionS3EndpointKey)
	bucket := requireJSONOpt(constants.FlexVolumeOptionS3BucketKey)
	region := jsonOptions[constants.FlexVolumeOptionS3RegionKey]
	if region == "" {
		region = constants.DefaultFlexVolumeOptionS3Region
	}
	accessKeyBase64 := requireJSONOpt(mountAccessKeyKey)
	secretKeyBase64 := requireJSONOpt(mountSecretKeyKey)
	accessKey, _ := base64.StdEncoding.DecodeString(accessKeyBase64)
	secretKey, _ := base64.StdEncoding.DecodeString(secretKeyBase64)

	log := cliLog.With().
		Str("endpoint", endpoint).
		Str("bucket", bucket).
		Str("region", region).
		Str("mountpoint", mountDir).
		Logger()

	// Fetch UID/GID
	/*uid, gid, err := myUserAndGroup()
	if err != nil {
		sendOutput(FlexOutput{Status: FlexStatusFailure, Message: fmt.Sprintf("Unable to fetch UID/GID: %s", err)})
		os.Exit(1)
	}*/

	// Test if mount already exists
	if mounted, err := isMounted(mountDir); err != nil {
		// isMounted check failed
		sendOutput(FlexOutput{Status: FlexStatusFailure, Message: fmt.Sprintf("Unable to check mount status: %s", err)})
		os.Exit(1)
	} else if mounted {
		// We're done
		sendOutput(FlexOutput{Status: FlexStatusSuccess})
		os.Exit(0)
	}

	// Now fork a child process
	var wg sync.WaitGroup
	var waitedForSignal os.Signal
	waitForSignal(&wg, &waitedForSignal)

	ctx := &daemon.Context{
		LogFileName: filepath.Join("/var/log/", driverName+".log"),
	}
	child, err := ctx.Reborn()
	if err != nil {
		sendOutput(FlexOutput{Status: FlexStatusFailure, Message: fmt.Sprintf("Unable to daemonize: %s", err)})
		os.Exit(1)
	}

	if child == nil {
		// I'm running as child process now
		// kill our own waiting goroutine
		kill(os.Getpid(), syscall.SIGUSR1)
		wg.Wait()
		defer ctx.Release()

		// Ensure mountDir exists
		os.MkdirAll(mountDir, 0755)

		// Prepare to mount
		config := goofys.Config{
			MountPoint: mountDir,
			DirMode:    0755,
			FileMode:   0644,
			//Uid:          uint32(uid),
			//Gid:          uint32(gid),
			StorageClass: "STANDARD",
			DebugFuse:    true,
			DebugS3:      true,
			Endpoint:     endpoint,
			Region:       region,
			AccessKey:    string(accessKey),
			SecretKey:    string(secretKey),
		}

		log.Debug().Msg("Mounting...")
		if _, mp, err := goofys.Mount(context.Background(), bucket, &config); err != nil {
			// Signal parent that the mount failed
			kill(os.Getppid(), syscall.SIGUSR2)
			// Log & exit
			log.Fatal().Err(err).Msg("Unable to mount")
		} else {
			// Mount succeeded
			log.Info().Msg("Mount succeeded")
			// Signal parent that the mount succeeded
			kill(os.Getppid(), syscall.SIGUSR1)
			// Wait until mount terminates
			mp.Join(context.Background())
			log.Info().Msg("Mount removed")
		}
	} else {
		// I'm running as parent process now.
		// Attempt to wait for child to notify parent
		wg.Wait()
		if waitedForSignal == syscall.SIGUSR1 {
			// Child signaled that the mount succeeded
			sendOutput(FlexOutput{Status: FlexStatusSuccess})
			os.Exit(0)
		} else {
			sendOutput(FlexOutput{Status: FlexStatusFailure, Message: fmt.Sprintf("Unexpected signal '%s'", waitedForSignal)})
			os.Exit(1)
		}
	}
}
