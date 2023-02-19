package partitions

import (
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strings"
)

type Devices []*Device

type ByPartNumber Devices

// Len implements sort.Interface
func (d ByPartNumber) Len() int {
	return len(d)
}

// Less implements sort.Interface
func (d ByPartNumber) Less(i int, j int) bool {
	return d[i].GetPartitionNumber() < d[j].GetPartitionNumber()
}

// Swap implements sort.Interface
func (d ByPartNumber) Swap(i int, j int) {
	d[i], d[j] = d[j], d[i]
}

var _ sort.Interface = ByPartNumber{}

var (
	ErrONIEPartitionNotFound = errors.New("devices: ONIE partition not found")
)

func (d Devices) GetEFIPartition() *Device {
	for _, dev := range d {
		if dev.IsEFIPartition() {
			return dev
		}
	}
	return nil
}

func (d Devices) GetONIEPartition() *Device {
	for _, dev := range d {
		if dev.IsONIEPartition() {
			return dev
		}
	}
	return nil
}

func (d Devices) GetDiagPartition() *Device {
	for _, dev := range d {
		if dev.IsDiagPartition() {
			return dev
		}
	}
	return nil
}

func (d Devices) GetHedgehogIdentityPartition() *Device {
	for _, dev := range d {
		if dev.IsHedgehogIdentityPartition() {
			return dev
		}
	}
	return nil
}

func (d Devices) GetHedgehogLocationPartition() *Device {
	for _, dev := range d {
		if dev.IsHedgehogLocationPartition() {
			return dev
		}
	}
	return nil
}

// DeletePartitions will find the NOS disk by identifying it through
// the location of the ONIE partition by default, and delete all
// non-EFI, non-ONIE and non-Hedgehog partitions.
//
// However, if a platform was passed and the platform falls into a
// category of exceptions (disk cannot be identified by ONIE partition),
// then it is cleaning up with a dedicated procedure.
//
// NOTE: it is advisable to call `Discover()` again after a call
// to this to make sure the partitions are gone from the devices
// list.
func (d Devices) DeletePartitions(platform string) error {
	switch platform {
	default:
		// no device supported with an exception at this
		// point in time
		return d.deletePartitionsByONIELocation()
	}
}

func (d Devices) deletePartitionsByONIELocation() error {
	oniePart := d.GetONIEPartition()
	if oniePart == nil {
		return ErrONIEPartitionNotFound
	}

	disk := oniePart.Disk
	if disk == nil {
		return ErrBrokenDiscovery
	}
	parts := disk.Partitions
	if len(parts) == 0 {
		return ErrBrokenDiscovery
	}
	var partsToDelete Devices
	for _, part := range parts {
		if part.IsEFIPartition() || part.IsONIEPartition() || part.IsDiagPartition() || part.IsHedgehogIdentityPartition() {
			continue
		}
		partsToDelete = append(partsToDelete, part)
	}

	// sort in descending order
	sort.Sort(sort.Reverse(ByPartNumber(partsToDelete)))

	// now delete them all, abort with an error if *any* deletion fails
	// it means the installer *must* fail as nothing is predictable anymore
	for _, part := range partsToDelete {
		if err := part.Delete(); err != nil {
			return err
		}
	}

	if err := disk.ReReadPartitionTable(); err != nil { // nolint
		// TODO: probably should get logged?
	}
	return nil
}

// CreateHedgehogIdentityPartition will find the NOS disk by identifying it through
// the location of the ONIE partition by default, and create the Hedgehog Identity
// Partition directly *after* the ONIE partition.
//
// However, if a platform was passed and the platform falls into a category of
// exceptions (disk cannot be identified by ONIE partition), then it is cleaning up
// with a dedicated procedure.
//
// NOTE: it is advisable to call `Discover()` again after a call
// to this to make sure the partition is in the list.
func (d Devices) CreateHedgehogIdentityPartition(platform string) error {
	switch platform {
	default:
		// no device supported with an exception at this
		// point in time
		return d.createHedgehogIdentityPartitionByONIELocation()
	}
}

func (d Devices) createHedgehogIdentityPartitionByONIELocation() error {
	oniePart := d.GetONIEPartition()
	if oniePart == nil {
		return ErrONIEPartitionNotFound
	}

	disk := oniePart.Disk
	if disk == nil {
		return ErrBrokenDiscovery
	}
	if disk.Path == "" {
		return ErrNoDeviceNode
	}
	parts := disk.Partitions
	if len(parts) == 0 {
		return ErrBrokenDiscovery
	}

	// new partition number is simply len + 1
	partNum := len(parts) + 1

	// sgdisk --new=${created_part}::+${created_part_size}MB \
	//     --attributes=${created_part}:=:$attr_bitmask \
	//     --change-name=${created_part}:$volume_label $blk_dev \

	// -t, --typecode=partnum:{hexcode|GUID}                                                           change partition type code
	if err := exec.Command(
		"sgdisk",
		fmt.Sprintf("--new=%d::+%dMB", partNum, DefaultPartSizeHedgehogIdentityInMB),
		fmt.Sprintf("--change-name=%d:%s", partNum, GPTPartNameHedgehogIdentity),
		fmt.Sprintf("--typecode=%d:%s", partNum, strings.ToUpper(GPTPartNameHedgehogIdentity)),
		disk.Path,
	).Run(); err != nil {
		return fmt.Errorf("devices: sgdisk create failed: %w", err)
	}

	// reread partition table
	if err := disk.ReReadPartitionTable(); err != nil { // nolint
		// TODO: log error
	}
	return nil
}
