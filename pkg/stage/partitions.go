package stage

import (
	"errors"
	"fmt"

	"go.githedgehog.com/dasboot/pkg/log"
	"go.githedgehog.com/dasboot/pkg/partitions"
	"go.githedgehog.com/dasboot/pkg/partitions/identity"
	"go.githedgehog.com/dasboot/pkg/partitions/location"
	"go.uber.org/zap"
)

func MountLocationPartition(l log.Interface, devices partitions.Devices) (location.LocationPartition, error) {
	lpdev := devices.GetHedgehogLocationPartition()
	if lpdev == nil {
		return nil, fmt.Errorf("location partition not found")
	}

	l.Info("Mounting Hedgehog Location Partition...", zap.String("source", lpdev.Path), zap.String("target", partitions.MountPathHedgehogLocation))
	if err := lpdev.Mount(); err != nil && !errors.Is(err, partitions.ErrAlreadyMounted) {
		l.Error("Mounting Hedgehog Location Partition failed", zap.Error(err))
		return nil, fmt.Errorf("mounting partition: %w", err)
	}

	l.Info("Opening Hedgehog Location Partition now...")
	lp, err := location.Open(lpdev)
	if err != nil {
		l.Error("Opening Hedgehog Location Partition failed", zap.Error(err))
		return nil, fmt.Errorf("opening location partition: %w", err)
	}
	return lp, nil
}

// MountIdentityPartition will find and mount the identity partition. It will be created
// if it does not exist yet.
func MountIdentityPartition(l log.Interface, devices partitions.Devices, platform string) (identity.IdentityPartition, error) {
	// we will rediscover them a couple of times potentially
	devs := devices

	// see if the partition exists already
	ipdev := devs.GetHedgehogIdentityPartition()
	if ipdev == nil {
		l.Info("Hedgehog Identity Parition does not exist yet, preparing disk...")

		// cleanup any partitions which should not be there
		l.Info("Deleting any partitions which should not be present...")
		if err := devs.DeletePartitions(platform); err != nil {
			l.Error("Cleaning up and deleting existing partitions failed", zap.Error(err))
			return nil, fmt.Errorf("deleting partitions: %w", err)
		}

		// rediscover disks/partitions after deletions
		l.Info("Rediscovering disks/partitions after potential partition deletions...")
		devs = partitions.Discover()
		if len(devs) == 0 {
			l.Error("Partition rediscovery after deleting partitions failed: no devices discovered")
			return nil, fmt.Errorf("partition rediscovery: no devices discovered")
		}

		l.Info("Hedgehog Identity Partition needs to be created...")
		if err := devs.CreateHedgehogIdentityPartition(platform); err != nil {
			l.Error("Creating Hedgehog Identity Partition failed", zap.Error(err))
			return nil, fmt.Errorf("creating partition: %w", err)
		}

		// rediscover disks/partitions after creating hedgehog
		l.Info("Rediscovering disks/partitions after creating Hedgehog Identity Partition...")
		devs = partitions.Discover()
		if len(devs) == 0 {
			l.Error("Rediscovering disks/partitions after creating Hedgehog Identity Partition failed: no devices discovered")
			return nil, fmt.Errorf("partition rediscovery: no devices discovered")
		}

		// get partition again
		l.Info("Getting Hedgehog Identity Partition again...")
		ipdev = devs.GetHedgehogIdentityPartition()
		if ipdev == nil {
			l.Error("Hedgehog Identity Partition missing after rediscovery for creating partition")
			return nil, fmt.Errorf("device not found after being created")
		}

		// creating filesystem on it
		l.Info("Creating filesystem for Hedgehog Identity Partition...")
		if err := ipdev.MakeFilesystemForHedgehogIdentityPartition(false); err != nil && !errors.Is(err, partitions.ErrFilesystemAlreadyCreated) {
			l.Error("Creating filesystem for Hedgehog Identity Partition failed", zap.Error(err))
			return nil, fmt.Errorf("creating filesystem: %w", err)
		}
	}

	// mount Hedgehog Identity partition
	l.Info("Mounting Hedgehog Identity Partition", zap.String("source", ipdev.Path), zap.String("target", partitions.MountPathHedgehogIdentity))
	if err := ipdev.Mount(); err != nil && !errors.Is(err, partitions.ErrAlreadyMounted) {
		l.Error("Mounting of Hedgehog Identity Partition failed", zap.Error(err))
		return nil, fmt.Errorf("mounting partition: %w", err)
	}

	// now open the partition according to our format
	// or initialize it if that has not been done yet
	l.Info("Opening Hedgehog Identity Partition now...")
	ip, err := identity.Open(ipdev)
	if err != nil {
		if errors.Is(err, identity.ErrUninitializedPartition) {
			l.Info("Hedgehog Idenity Partition still needs to be initialized...")
			ip, err = identity.Init(ipdev)
			if err != nil {
				l.Error("Initializing Hedgehog Identity Partition failed", zap.Error(err))
				return nil, fmt.Errorf("initializing identity partition: %w", err)
			}
		} else {
			l.Error("Opening Hedgehog Identity Partition failed", zap.Error(err))
			return nil, fmt.Errorf("opening identity partition: %w", err)
		}
	}

	return ip, nil
}
