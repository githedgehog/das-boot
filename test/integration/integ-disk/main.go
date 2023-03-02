package main

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"

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

func integDisk(_ *cli.Context) error {
	// discover disks/partitions first
	l.Info("1. Initial disks/partitions discovery...")
	devs := partitions.Discover()
	if len(devs) == 0 {
		return fmt.Errorf("initial partition discovery failed: no devices discovered")
	}

	// cleanup any partitions which should not be there
	l.Info("2. Deleting any partitions which should not be present...")
	if err := devs.DeletePartitions(os.Getenv("onie_platform")); err != nil {
		return fmt.Errorf("failed to delete partitions: %w", err)
	}

	// rediscover disks/partitions after deletions
	l.Info("3. Rediscovering disks/partitions after potential partition deletions...")
	devs = partitions.Discover()
	if len(devs) == 0 {
		return fmt.Errorf("partition rediscovery after deleting partitions failed: no devices discovered")
	}
	// check partitions are as expected
	l.Info("4. Check partitions are as expected after initial discovery...")
	if err := checkPartitions(devs, false); err != nil {
		return fmt.Errorf("checking partions failed: %w", err)
	}

	// now create identity partition if it is not present yet
	l.Info("5. Ensuring Hedgehog Identity Partition exists...")
	hhip := devs.GetHedgehogIdentityPartition()
	if hhip != nil {
		l.Info("5.1 Hedgehog Identity Partition already exists")
	} else {
		l.Info("5.1 Hedgehog Identity Partition needs to be created...")
		if err := devs.CreateHedgehogIdentityPartition(os.Getenv("onie_platform")); err != nil {
			return fmt.Errorf("creating Hedgehog Identity Partition failed: %w", err)
		}

		// rediscover disks/partitions after creating hedgehog
		l.Info("5.2 Rediscovering disks/partitions after creating Hedgehog Identity Partition...")
		devs = partitions.Discover()
		if len(devs) == 0 {
			return fmt.Errorf("partition rediscovery after creating Hedgehog Identity Partition failed: no devices discovered")
		}

		// get partition again
		l.Info("5.3 Getting Hedgehog Identity Partition again...")
		hhip = devs.GetHedgehogIdentityPartition()
		if hhip == nil {
			return fmt.Errorf("Hedgehog Identity Partition missing after rediscovery for creating partition")
		}
	}

	// creating filesystem on it
	l.Info("6. Ensuring filesystem is correct on Hedgehog Identity Partition and creating it if necessary...")
	if err := hhip.MakeFilesystemForHedgehogIdentityPartition(false); err != nil && !errors.Is(err, partitions.ErrFilesystemAlreadyCreated) {
		return fmt.Errorf("ensuring filesystem for Hedgehog Identity Partition failed: %w", err)
	}

	// rediscover disks/partitions after creating filesystem
	// NOTE: we wouldn't really need to do this step anymore, however, this is to probe the discovery mechanism again
	l.Info("7. Rediscovering disks/partitions after making filesystem for Hedgehog Identity Partition...")
	devs = partitions.Discover()
	if len(devs) == 0 {
		return fmt.Errorf("partition rediscovery after creating Hedgehog Identity Partition failed: no devices discovered")
	}

	// check partitions are as expected again, this time identity partition must exist
	l.Info("8. Check partitions again after deleting/creating partitions and filesystems...")
	if err := checkPartitions(devs, true); err != nil {
		return fmt.Errorf("checking partions failed: %w", err)
	}

	// print device/disk information
	l.Info("9. Printing disks/partitions for confirmation...")
	logDevs(devs)

	// last but not least, mount Hedgehog Identity partition
	l.Info("10. Mounting Hedgehog Identity Partition", zap.String("source", hhip.Path), zap.String("target", partitions.MountPathHedgehogIdentity))
	if err := hhip.Mount(); err != nil && !errors.Is(err, partitions.ErrAlreadyMounted) {
		return fmt.Errorf("mounting of Hedgehog Identity Partition failed: %w", err)
	}

	// c'est fini
	l.Info("SUCCESS")
	return nil
}

func checkPartitions(devs partitions.Devices, mustHaveIdentity bool) error {
	// if this is the default x86 platform, then our partition layout will be predictable
	// do some assertions around that!
	if runtime.GOARCH == "amd64" {
		var numParts int

		efiPart := devs.GetEFIPartition()
		if efiPart == nil {
			l.Error("EFI partition missing which means it could have been delete before")
			l.Info("printing devs before exit")
			logDevs(devs)
			return fmt.Errorf("EFI partition missing")
		}
		partn := efiPart.GetPartitionNumber()
		if partn != 1 {
			l.Warn("EFI partition number is not (1) as it usually should be", zap.Int("partn", partn))
		}
		numParts += 1

		oniePart := devs.GetONIEPartition()
		if oniePart == nil {
			l.Error("ONIE partition missing which means it could have been deleted before")
			l.Info("printing devs before exit")
			logDevs(devs)
			return fmt.Errorf("ONIE partition missing")
		}
		partn = oniePart.GetPartitionNumber()
		if partn != 2 && partn != 3 {
			l.Warn("ONIE partition number is not (2) (or (3)) as it usually should be", zap.Int("partn", partn))
		}
		numParts += 1

		diagPart := devs.GetDiagPartition()
		if diagPart == nil {
			l.Debug("no Diag partition found")
		} else {
			partn = diagPart.GetPartitionNumber()
			if partn != 3 && partn != 2 {
				l.Warn("Diag partition number is not (3) (or (2)) as it usually should be", zap.Int("partn", partn))
			}
			numParts += 1
		}

		hhidPart := devs.GetHedgehogIdentityPartition()
		if hhidPart == nil && mustHaveIdentity {
			l.Error("Hedgehog partition missing even though it must be present now")
			l.Info("printing devs before exit")
			logDevs(devs)
			return fmt.Errorf("Hedgehog identity Partition missing")
		} else if hhidPart == nil {
			l.Debug("no Hedgehog Identity partion found")
		} else {
			partn = hhidPart.GetPartitionNumber()
			if partn != 3 && partn != 4 {
				l.Warn("Hedgehog Identity Partion number is not (3) (or (4)) as it usually should be", zap.Int("partn", partn))
			}

			// we'll check other things now as well
			// all except partition name which might not be updated in sysfs, and filesystem which will be ext2 instead of ext4 for the time being
			if hhidPart.GPTPartType != partitions.GPTPartTypeHedgehogIdentity {
				l.Error("Hedgehog Identity Partition does not have expected GPT partition type GUID", zap.String("got", hhidPart.GPTPartType), zap.String("want", partitions.GPTPartTypeHedgehogIdentity))
				return fmt.Errorf("unexpected GPT partition type for Hedgehog Identity partition")
			}
			if hhidPart.FSLabel != partitions.FSLabelHedgehogIdentity && mustHaveIdentity {
				l.Error("Hedgehog Identity Partition does not have expected FS Label", zap.String("got", hhidPart.FSLabel), zap.String("want", partitions.FSLabelHedgehogIdentity))
				return fmt.Errorf("unexpected FS Label for Hedgehog Identity partition")
			}
			if hhidPart.FSLabel != partitions.FSLabelHedgehogIdentity {
				l.Warn("Hedgehog Identity Partition does not have expected FS Label", zap.String("got", hhidPart.FSLabel), zap.String("want", partitions.FSLabelHedgehogIdentity))
			}
			numParts += 1
		}

		disk := oniePart.Disk
		if disk == nil {
			l.Error("broken internal structure: oniePart.Disk is nil")
			return fmt.Errorf("internal error")
		}

		// now check number of expected partitions
		if len(disk.Partitions) != numParts {
			l.Error("unexpected number of partitions", zap.Int("got", len(disk.Partitions)), zap.Int("want", numParts))
			return fmt.Errorf("unexpected number of partitions")
		}
	}
	return nil
}

func logDevs(devs partitions.Devices) {
	for _, dev := range devs {
		devname := dev.GetDeviceName()
		if strings.HasPrefix(devname, "ram") || strings.HasPrefix(devname, "loop") {
			// skip ram and loop devices - not very interesting and clobbers up the output
			continue
		}
		major, minor, err := dev.GetMajorMinor()
		if err != nil {
			l.Debug("GetMajorMinor failed", zap.Error(err))
		}
		if dev.IsDisk() {
			l.Info(
				"disk",
				zap.String("devname", devname),
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
				zap.String("devname", devname),
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
}
