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

package partitions

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"go.githedgehog.com/dasboot/pkg/exec"
	"go.githedgehog.com/dasboot/pkg/log"

	"go.uber.org/zap"
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
	ErrPartitionExists       = errors.New("devices: partition exists")
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

func (d Devices) GetSONiCPartition() *Device {
	for _, dev := range d {
		if dev.IsSonicPartition() {
			return dev
		}
	}
	return nil
}

// DeletePartitions will find the NOS disk by identifying it through
// the location of the ONIE partition by default, and delete all
// non-EFI, non-ONIE, non-Diag and non-Hedgehog partitions. This is
// the documented disk and partition layout for x86 platforms.
// Note that all arm platforms fall into the categories of exceptions
// as mentioned in the next paragraph.
//
// However, if a platform was passed and the platform falls into a
// category of exceptions (disk cannot be identified by ONIE partition),
// then it is cleaning up with a dedicated procedure.
// `platform` is expected to be the value of the `onie_platform`
// environment variable.
//
// DeletePartitions will call `ReReadPartitionTable()` on the disk that
// it operated on.
//
// DeletePartitions will also ensure that the BoorOrder has ONIE as
// the first boot entry because after a call to this function, there
// is not going to be any NOS available anymore, and a subsequent
// reboot **MUST** ensure that it boots into ONIE again. It will also
// remove any bogus EFI boot variables which are now invalid after the
// deletion of those partitions.
// It uses the `MakeONIEDefaultBootEntryAndCleanup()` function for this
// procedure.
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
	if len(partsToDelete) > 0 {
		for _, part := range partsToDelete {
			if err := part.Delete(); err != nil {
				return err
			}
		}

		if err := disk.ReReadPartitionTable(); err != nil {
			log.L().Warn("rereading partition table failed", zap.Error(err))
		}

		// If we deleted partions, then this means that we deleted
		// NOS partitions. This means that we could have an unbootable
		// system otherwise if things go wrong in the upcoming process.
		// So we will default back to ONIE to ensure that we are good.
		// This will also cleanup any boot entries which are now invalid.
		if err := MakeONIEDefaultBootEntryAndCleanup(); err != nil {
			return err
		}
	}
	return nil
}

// CreateHedgehogIdentityPartition will find the NOS disk by identifying it through
// the location of the ONIE partition by default, and create the Hedgehog Identity
// Partition on the ONIE partition. You want to call this function **after** a
// call to `DeletePartitions()` to make sure there is room for the identity
// partition to be created.
//
// However, if a platform was passed and the platform falls into a category of
// exceptions (disk cannot be identified by ONIE partition), then it is creating the
// partition with a dedicated procedure. See the documentation for `DeletePartitions`
// for more details.
//
// CreateHedgehogIdentityPartition will call ReReadPartitionTable on the disk that
// it operated on.
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
	if d.GetHedgehogIdentityPartition() != nil {
		return ErrPartitionExists
	}
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
		fmt.Sprintf("--typecode=%d:%s", partNum, strings.ToUpper(GPTPartTypeHedgehogIdentity)),
		disk.Path,
	).Run(); err != nil {
		return fmt.Errorf("devices: sgdisk create failed: %w", err)
	}

	// reread partition table
	if err := disk.ReReadPartitionTable(); err != nil {
		log.L().Warn("rereading partition table failed", zap.Error(err))
	}
	return nil
}
