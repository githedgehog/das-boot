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
)

var l = log.L()

func main() {
	app := &cli.App{
		Name:                 "integ-netdev",
		Usage:                "integration test for network device and vlan configuration",
		UsageText:            "integ-netdevr",
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
	for _, ipaddrstr := range ipaddrs {
		ipaddr, ipnet, err := net.ParseCIDR(ipaddrstr)
		if err != nil {
			return fmt.Errorf("failed to parse IP address and netmask: %w", err)
		}
		ipnet.IP = ipaddr
		ipnets = append(ipnets, ipnet)
	}

	l.Info("Adding VLAN interface and IP address...",
		zap.String("device", dev),
		zap.Uint16("vid", vid),
		zap.String("vlanName", vlanName),
		zap.Strings("ipnets", ipaddrs),
	)
	if err := dbnet.AddVLANDeviceWithIP(dev, vid, vlanName, ipnets); err != nil {
		return fmt.Errorf("adding VLAN and address failed: %w", err)
	}
	l.Info("Success")
	return nil
}

func integNetdevDelete(ctx *cli.Context) error {
	dev := ctx.String("device")

	if err := dbnet.DeleteVLANDevice(dev); err != nil {
		return fmt.Errorf("deleting VLAN interface failed: %w", err)
	}
	l.Info("Success")
	return nil
}