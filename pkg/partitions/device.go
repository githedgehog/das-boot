package partitions

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

type Device struct {
	Uevent
	SysfsPath   string
	Path        string
	GPTPartType string
	FSLabel     string
	Disk        *Device
	Partitions  []*Device
}

const (
	FSLabelONIE             = "ONIE-BOOT"
	FSLabelSONiC            = "SONiC-OS"
	FSLabelHedgehogIdentity = "HEDGEHOG_IDENTITY"
	FSLabelHedgehogLocation = "HEDGEHOG_LOCATION"

	GPTPartNameONIE             = FSLabelONIE
	GPTPartNameHedgehogIdentity = FSLabelHedgehogIdentity
	GPTPartNameHedgehogLocation = FSLabelHedgehogLocation

	GPTPartTypeONIE             = "7412f7d5-a156-4b13-81dc-867174929325"
	GPTPartTypeEFI              = "c12a7328-f81f-11d2-ba4b-00a0c93ec93b"
	GPTPartTypeHedgehogIdentity = "e982e2bd-867c-4d7a-89a2-9c5a9bc5dfdd"
	GPTPartTypeHedgehogLocation = "e23c5ebc-5f53-488c-959d-a4ab90befefe"

	DefaultPartSizeHedgehogIdentityInMB int = 100

	blkrrpart = 0x125f
)

var (
	ErrNoDeviceNode       = errors.New("device: no device node present")
	ErrDeviceNotPartition = errors.New("device: not a partition")
	ErrDeviceNotDisk      = errors.New("device: not a disk")
	ErrBrokenDiscovery    = errors.New("device: broken discovery")
)

func (d *Device) ensureDevicePath() error {
	path, err := d.Uevent.DevicePath()
	if err != nil {
		// TODO: create device node if it does not exist
		return err
	}
	d.Path = path
	return nil
}

func (d *Device) discoverPartitionType() error {
	if d.Path == "" {
		return ErrNoDeviceNode
	}
	out, err := exec.Command("grub-probe", "-d", d.Path, "-t", "gpt_parttype").Output()
	if err != nil {
		return fmt.Errorf("device: grub-probe failed: %w", err)
	}
	d.GPTPartType = strings.TrimSpace(strings.ToLower(string(out)))
	return nil
}

func (d *Device) discoverFilesystemLabel() error {
	if d.Path == "" {
		return ErrNoDeviceNode
	}
	out, err := exec.Command("grub-probe", "-d", d.Path, "-t", "fs_label").Output()
	if err != nil {
		return err
	}
	d.FSLabel = strings.TrimSpace(string(out))
	return nil
}

func (d *Device) IsEFIPartition() bool {
	if d.IsPartition() {
		// the labels for the EFI system partition or filesystem vary from OS to OS, the only reliable indicator is the partition type
		return d.GPTPartType == GPTPartTypeEFI
	}
	return false
}

func (d *Device) IsONIEPartition() bool {
	if d.IsPartition() {
		return d.GPTPartType == GPTPartTypeONIE || d.GetPartitionName() == GPTPartNameONIE || d.FSLabel == FSLabelONIE
	}
	return false
}

func (d *Device) IsHedgehogIdentityPartition() bool {
	if d.IsPartition() {
		return d.GPTPartType == GPTPartTypeHedgehogIdentity || d.GetPartitionName() == GPTPartNameHedgehogIdentity || d.FSLabel == FSLabelHedgehogIdentity
	}
	return false
}

func (d *Device) IsHedgehogLocationPartition() bool {
	if d.IsPartition() {
		return d.GPTPartType == GPTPartTypeHedgehogLocation || d.GetPartitionName() == GPTPartNameHedgehogLocation || d.FSLabel == FSLabelHedgehogLocation
	}
	return false
}

func (d *Device) Delete() error {
	if !d.IsPartition() {
		return ErrDeviceNotPartition
	}

	// build a command to call sgdisk
	// get the partition number
	partNum := d.GetPartitionNumber()
	if partNum <= 0 {
		return ErrInvalidUevent
	}

	// and the device path of the disk (NOT the partition)
	disk := d.Disk
	if disk == nil {
		return ErrBrokenDiscovery
	}
	if disk.Path == "" {
		return ErrNoDeviceNode
	}

	if err := exec.Command("sgdisk", "-d", strconv.Itoa(partNum), disk.Path).Run(); err != nil {
		return fmt.Errorf("device: sgdisk -d failed: %w", err)
	}
	return nil
}

func (d *Device) ReReadPartitionTable() error {
	if !d.IsDisk() {
		return ErrDeviceNotDisk
	}
	if d.Path == "" {
		return ErrNoDeviceNode
	}
	f, err := os.Open(d.Path)
	if err != nil {
		return err
	}
	if _, err = unix.IoctlGetInt(int(f.Fd()), blkrrpart); err != nil {
		return fmt.Errorf("device: unable to re-read partition table: %ww", err)
	}
	return nil
}
