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
	"os"

	"go.githedgehog.com/dasboot/pkg/devid"
	"go.githedgehog.com/dasboot/pkg/log"
	"go.githedgehog.com/dasboot/pkg/version"

	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

var l = log.L()

var description = `
hhdevid determines a unique device ID for the device that it is running on
in the form of a UUID. This ID is used throughout Hedgehog SONiC for
identifying a running device - particularly at installation time by DAS BOOT.

The hhdevid tool is meant to be running on ONIE with root privileges.
Otherwise it could fail some of the detection mechanisms like missing
executables like 'onie-sysinfo' or 'onie-syseeprom', or it cannot open
/sys/class/dmi/id/product_uuid. Note that this could be expected depending
on the device where it is running on which is why this tool is not failing in
these cases.

It tries to determine a unique ID through the following methods - falling back
to the next if the previous one failed:

1. Using ONIE sysinfo and EEPROM information: vendor ID + serial number
2. on x86: through the System UUID of the DMI
3. on arm: through the serial number of the CPU
4. through all MAC addresses of all physical network devices

To use the device ID in a script, pay attention to redirect stderr
to /dev/null to avoid capturing potential log messages:

hhdevid 2>/dev/null
`

func main() {
	app := &cli.App{
		Name:        "hhdevid",
		Usage:       "device identification tool",
		UsageText:   "hhdevid",
		Description: description[1 : len(description)-1],
		Version:     version.Version,
		Action: func(ctx *cli.Context) error {
			devid := devid.ID()
			_, err := os.Stdout.WriteString(devid + "\n")
			return err
		},
	}

	if err := app.Run(os.Args); err != nil {
		l.Fatal("hhdevid failed", zap.Error(err))
	}
}
