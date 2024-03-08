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
	"net"
	"os"

	"go.githedgehog.com/dasboot/pkg/log"
	dbnet "go.githedgehog.com/dasboot/pkg/net"
	"go.githedgehog.com/dasboot/pkg/version"

	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var l = log.L()

func main() {
	app := &cli.App{
		Name:                 "integ-netdev",
		Usage:                "integration test for network device and vlan configuration",
		UsageText:            "integ-netdev",
		Description:          "Should be running in ONIE, and will try to add/delete a vlan and IP address to/from a network device",
		Version:              version.Version,
		EnableBashCompletion: true,
		Commands: []*cli.Command{
			{
				Name:  "add",
				Usage: "adds a vlan and IP address to a network device",
				Flags: []cli.Flag{
					&cli.UintFlag{
						Name:  "vid",
						Usage: "VLAN ID / VID",
						Value: 42,
					},
					&cli.StringFlag{
						Name:  "vlan-name",
						Usage: "VLAN interface name",
						Value: "mgmt",
					},
					&cli.StringSliceFlag{
						Name:  "ip-address",
						Usage: "IP addresses with their netmask CIDR",
						Value: cli.NewStringSlice("192.168.42.101/24"),
					},
					&cli.StringSliceFlag{
						Name:  "subnet",
						Usage: "Additional subnets to be added as routes on the same VLAN interface",
						Value: cli.NewStringSlice(
							"10.142.0.0/16",
							"10.143.0.0/16",
						),
					},
					&cli.StringFlag{
						Name:  "gateway",
						Usage: "Nexthop to use for the subnet routes",
						Value: "192.168.42.1",
					},
					&cli.IntFlag{
						Name:  "flags",
						Usage: "Set flags like onlink for the route",
						Value: 0,
					},
					&cli.StringFlag{
						Name:    "device",
						Aliases: []string{"dev"},
						Usage:   "parent network device to which to add the VLAN",
						Value:   "eth0",
					},
				},
				Action: func(ctx *cli.Context) error {
					// run the test
					return integNetdevAdd(ctx)
				},
			},
			{
				Name:    "delete",
				Aliases: []string{"del", "remove", "rem"},
				Usage:   "deletes a vlan interface",
				Flags: []cli.Flag{
					&cli.StringSliceFlag{
						Name:  "ip-address",
						Usage: "IP addresses with their netmask CIDR",
						Value: cli.NewStringSlice("192.168.42.101/24"),
					},
					&cli.StringSliceFlag{
						Name:  "subnet",
						Usage: "Additional subnets to be added as routes on the same VLAN interface",
						Value: cli.NewStringSlice(
							"10.142.0.0/16",
							"10.143.0.0/16",
						),
					},
					&cli.StringFlag{
						Name:  "gateway",
						Usage: "Nexthop to use for the subnet routes",
						Value: "192.168.42.1",
					},
					&cli.IntFlag{
						Name:  "flags",
						Usage: "Set flags like onlink for the route",
						Value: 0,
					},
					&cli.StringFlag{
						Name:    "device",
						Aliases: []string{"dev"},
						Usage:   "VLAN device to delete",
						Value:   "mgmt",
					},
				},
				Action: func(ctx *cli.Context) error {
					// run the test
					return integNetdevDelete(ctx)
				},
			},
		},
	}

	l = log.NewZapWrappedLogger(zap.Must(log.NewSerialConsole(zapcore.DebugLevel, "console", true)))
	log.ReplaceGlobals(l)

	if err := app.Run(os.Args); err != nil {
		l.Fatal("integ-netdev failed", zap.Error(err), zap.String("errType", fmt.Sprintf("%T", err)))
	}
}

func integNetdevAdd(ctx *cli.Context) error {
	vid := uint16(ctx.Uint("vid"))
	dev := ctx.String("device")
	vlanName := ctx.String("vlan-name")

	l.Info("Parsing IP and netmasks from input...")
	ipaddrs := ctx.StringSlice("ip-address")
	var ipnets []*net.IPNet
	if len(ipaddrs) > 0 {
		var err error
		ipnets, err = dbnet.StringsToIPNets(ipaddrs)
		if err != nil {
			return fmt.Errorf("failed to parse IP addresses and netmask: %w", err)
		}
	}

	subnets := ctx.StringSlice("subnet")
	var routedests []*net.IPNet
	if len(subnets) > 0 {
		l.Info("Parsing subnets from input...")
		var err error
		routedests, err = dbnet.StringsToIPNets(subnets)
		if err != nil {
			return fmt.Errorf("failed to parse subnets: %w", err)
		}
	}

	gw := ctx.String("gateway")
	var routegw net.IP
	if gw != "" {
		l.Info("Parsing gateway from input...")
		routegw = net.ParseIP(gw)
		if routegw == nil {
			return fmt.Errorf("failed to parse gateway IP from: '%s'", gw)
		}
	}
	if (len(routedests) > 0 && routegw == nil) || (len(routedests) == 0 && routegw != nil) {
		l.Error("you must specify subnets and gateway together")
		return fmt.Errorf("subnets and gateway must be specified together")
	}

	routeflags := ctx.Int("flags")

	// now build the routes
	var routes []*dbnet.Route
	if len(routedests) > 0 && routegw != nil {
		routes = []*dbnet.Route{
			{
				Dests: routedests,
				Gw:    routegw,
				Flags: routeflags,
			},
		}
	}

	l.Info("Adding VLAN interface and IP address...",
		zap.String("device", dev),
		zap.Uint16("vid", vid),
		zap.String("vlanName", vlanName),
		zap.Reflect("ipnets", ipnets),
	)
	if err := dbnet.AddVLANDeviceWithIP(dev, vid, vlanName, ipnets, routes); err != nil {
		return fmt.Errorf("adding VLAN and address failed: %w", err)
	}
	l.Info("Success")
	return nil
}

func integNetdevDelete(ctx *cli.Context) error {
	dev := ctx.String("device")

	l.Info("Parsing IP and netmasks from input...")
	ipaddrs := ctx.StringSlice("ip-address")
	var ipnets []*net.IPNet
	if len(ipaddrs) > 0 {
		var err error
		ipnets, err = dbnet.StringsToIPNets(ipaddrs)
		if err != nil {
			return fmt.Errorf("failed to parse IP addresses and netmask: %w", err)
		}
	}

	subnets := ctx.StringSlice("subnet")
	var routedests []*net.IPNet
	if len(subnets) > 0 {
		l.Info("Parsing subnets from input...")
		var err error
		routedests, err = dbnet.StringsToIPNets(subnets)
		if err != nil {
			return fmt.Errorf("failed to parse subnets: %w", err)
		}
	}

	gw := ctx.String("gateway")
	var routegw net.IP
	if gw != "" {
		l.Info("Parsing gateway from input...")
		routegw = net.ParseIP(gw)
		if routegw == nil {
			return fmt.Errorf("failed to parse gateway IP from: '%s'", gw)
		}
	}
	if (len(routedests) > 0 && routegw == nil) || (len(routedests) == 0 && routegw != nil) {
		l.Error("you must specify subnets and gateway together")
		return fmt.Errorf("subnets and gateway must be specified together")
	}

	routeflags := ctx.Int("flags")

	// now build the routes
	var routes []*dbnet.Route
	if len(routedests) > 0 && routegw != nil {
		routes = []*dbnet.Route{
			{
				Dests: routedests,
				Gw:    routegw,
				Flags: routeflags,
			},
		}
	}

	if err := dbnet.DeleteVLANDevice(dev, ipnets, routes); err != nil {
		return fmt.Errorf("deleting VLAN interface failed: %w", err)
	}
	l.Info("Success")
	return nil
}
