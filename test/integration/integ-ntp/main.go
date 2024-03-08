// Copyright 2023 Hedgehog
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

package main

import (
	"fmt"
	"os"

	"go.githedgehog.com/dasboot/pkg/log"
	"go.githedgehog.com/dasboot/pkg/ntp"
	"go.githedgehog.com/dasboot/pkg/version"
	"go.uber.org/zap"

	"github.com/urfave/cli/v2"
)

var l = log.L()

func main() {
	app := &cli.App{
		Name:                 "integ-ntp",
		Usage:                "integration test for NTP queries and updating the system clock",
		UsageText:            "integ-ntp --server 192.168.42.1 --server 0.arch.pool.ntp.org",
		Description:          "Should be running in ONIE, needs networking configured, and should run with an unsynchronized system clock for good comparisons after a run",
		Version:              version.Version,
		EnableBashCompletion: true,
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:  "server",
				Usage: "NTP server IP addresses or hostnames or FQDNs",
				Value: cli.NewStringSlice("192.168.42.1"),
			},
		},
		Action: func(ctx *cli.Context) error {
			// run the test
			return integNTP(ctx)
		},
	}

	if err := app.Run(os.Args); err != nil {
		l.Fatal("integ-ntp failed", zap.Error(err), zap.String("errType", fmt.Sprintf("%T", err)))
	}
}

func integNTP(ctx *cli.Context) error {
	servers := ctx.StringSlice("server")

	l.Info("Trying to query NTP servers, and updating the system clock if successful", zap.Strings("servers", servers))
	if err := ntp.SyncClock(ctx.Context, servers); err != nil {
		return err
	}

	l.Info("Success")
	return nil
}
