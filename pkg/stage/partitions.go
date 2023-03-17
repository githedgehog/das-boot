package stage

import (
	"errors"
	"fmt"

	"go.githedgehog.com/dasboot/pkg/partitions"
	"go.githedgehog.com/dasboot/pkg/partitions/location"
)

func MountLocationPartition(devices partitions.Devices) (location.LocationPartition, error) {
	lpdev := devices.GetHedgehogLocationPartition()
	if lpdev == nil {
		return nil, fmt.Errorf("location partition not found")
	}
	if err := lpdev.Mount(); err != nil && !errors.Is(err, partitions.ErrAlreadyMounted) {
		return nil, fmt.Errorf("failed to mount location partition: %w", err)
	}
	lp, err := location.Open(lpdev)
	if err != nil {
		return nil, fmt.Errorf("failed to open location partition: %w", err)
	}
	return lp, nil
}
