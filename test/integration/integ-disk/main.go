package main

import (
	"fmt"
	"os"

	"go.githedgehog.com/dasboot/pkg/log"
	"go.githedgehog.com/dasboot/pkg/partitions"
	"go.githedgehog.com/dasboot/pkg/version"
	"go.uber.org/zap"

	"github.com/urfave/cli/v2"
)

var l = log.L()

func main() {
	var acknowledgeDanger bool
	app := &cli.App{
		Name:                 "integ-disk",
		Usage:                "integration test for disk partitioning and formating",
		UsageText:            "integ-disk --acknowledge-danger",
		Description:          "Should be running in ONIE, and will find the ONIE partition and remove any unknown partitions and create a Hedgehog Identity partition and format it",
		Version:              version.Version,
		EnableBashCompletion: true,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "acknowledge-danger",
				Destination: &acknowledgeDanger,
				Usage:       "acknowledge that this is a dangerous operation",
				Required:    true,
			},
		},
		Action: func(ctx *cli.Context) error {
			// prevent a hooman from doing something stupid
			if !acknowledgeDanger {
				return fmt.Errorf("You *MUST* pass the --acknowledge-danger flag set to true to execute this integration test! **NOTE:** This is going to DELETE partitions if executed successfully, and create new ones in the process. Be sure you run this on a dedicated test bed system!")
			}

			// run the test
			return integDisk(ctx)
		},
	}

	if err := app.Run(os.Args); err != nil {
		l.Fatal("integ-disk failed", zap.Error(err))
	}
}

func integDisk(ctx *cli.Context) error {
	// discover disks/partitions first
	l.Info("1. Initial disks/partitions discovery...")
	devs, err := partitions.Discover()
	if err != nil {
		return fmt.Errorf("initial partition discovery failed: %w", err)
	}

	// cleanup any partitions which should not be there
	l.Info("2. Deleting any partitions which should not be present...")
	if err := devs.DeletePartitions(os.Getenv("onie_platform")); err != nil {
		return fmt.Errorf("failed to delete partitions: %w", err)
	}

	// rediscover disks/partitions after deletions
	l.Info("3. Rediscovering disks/partitions after potential partition deletions...")
	devs, err = partitions.Discover()
	if err != nil {
		return fmt.Errorf("partition rediscovery after deleting partitions failed: %w", err)
	}

	// now create identity partition if it is not present yet
	l.Info("4. Ensuring Hedgehog Identity Partition exists...")
	if hhip := devs.GetHedgehogIdentityPartition(); hhip != nil {
		l.Info("4.1 Hedgehog Identity Partition already exists")
	} else {
		l.Info("4.1 Hedgehog Identity Partition needs to be created...")
		if err := devs.CreateHedgehogIdentityPartition(os.Getenv("onie_platform")); err != nil {
			return fmt.Errorf("creating Hedgehog Identity Partition failed: %w", err)
		}

		// rediscover disks/partitions after creating hedgehog
		l.Info("4.2 Rediscovering disks/partitions after creating Hedgehog Identity Partition...")
		devs, err = partitions.Discover()
		if err != nil {
			return fmt.Errorf("partition rediscovery after creating Hedgehog Identity Partition failed: %w", err)
		}
	}

	// success, now just print device/disk information
	l.Info("5. Success! Printing disks/partitions for confirmation...")
	for _, dev := range devs {
		major, minor, err := dev.GetMajorMinor()
		if err != nil {
			l.Debug("GetMajorMinor failed", zap.Error(err))
		}
		if dev.IsDisk() {
			l.Info(
				"disk",
				zap.String("devname", dev.GetDeviceName()),
				zap.Uint32("major", major),
				zap.Uint32("minor", minor),
				zap.String("path", dev.Path),
				zap.String("sysfs_path", dev.SysfsPath),
			)
			continue
		}
		if dev.IsPartition() {
			l.Info(
				"partition",
				zap.String("devname", dev.GetDeviceName()),
				zap.Uint32("major", major),
				zap.Uint32("minor", minor),
				zap.String("partname", dev.GetPartitionName()),
				zap.Int("partn", dev.GetPartitionNumber()),
				zap.String("gpt_parttype", dev.GPTPartType),
				zap.String("filesystem", dev.Filesystem),
				zap.String("fs_label", dev.FSLabel),
				zap.Bool("is_efi", dev.IsEFIPartition()),
				zap.Bool("is_onie", dev.IsONIEPartition()),
				zap.Bool("is_diag", dev.IsDiagPartition()),
				zap.Bool("is_hh_identity", dev.IsHedgehogIdentityPartition()),
				zap.Bool("is_hh_location", dev.IsHedgehogLocationPartition()),
			)
		}
	}

	// c'est fini
	return nil
}
