package partitions

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

type Device struct {
	Uevent
	SysfsPath   string
	Path        string
	MountPath   string
	Filesystem  string
	GPTPartType string
	FSLabel     string
	Disk        *Device
	Partitions  []*Device
}

const (
	FSExt4 = "ext4"

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

	MountPathHedgehogIdentity = "/mnt/hedgehog-identity"
	MountPathHedgehogLocation = "/mnt/hedgehog-location"

	DefaultPartSizeHedgehogIdentityInMB int = 100

	blkrrpart = 0x125f
)

var (
	ErrNoDeviceNode              = errors.New("device: no device node present")
	ErrDeviceNotPartition        = errors.New("device: not a partition")
	ErrDeviceNotDisk             = errors.New("device: not a disk")
	ErrBrokenDiscovery           = errors.New("device: broken discovery")
	ErrUnsupportedMkfsForDevice  = errors.New("device: unsupported device for mkfs")
	ErrAlreadyMounted            = errors.New("device: already mounted")
	ErrUnsupportedMountForDevice = errors.New("device: unsupported device for mount")
	ErrFilesystemAlreadyCreated  = errors.New("device: filesystem already present")
)

func (d *Device) ensureDevicePath() error {
	var err error
	var path string
	path, err = d.Uevent.DevicePath()
	if err != nil {
		if errors.Is(err, ErrInvalidUevent) {
			// we cannot recover from this
			return err
		}
		major, minor, err := d.GetMajorMinor()
		if err != nil {
			// we cannot recover from this
			return err
		}

		// create device node if it does not exist
		// ensure to delete any existing file or directory
		// or whatever else might be in the way
		p := filepath.Join(rootPath, "dev", d.Uevent[UeventDevname])
		info, err := os.Stat(p)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("device: ensure device path: stat: %w", err)
		}
		if info != nil {
			if err := os.Remove(p); err != nil {
				return fmt.Errorf("device: ensure device path: remove %s: %w", p, err)
			}
		}
		if err := unix.Mknod(p, unix.S_IFBLK, int(unix.Mkdev(major, minor))); err != nil {
			return fmt.Errorf("device: mknod: %w", err)
		}

		// now try again
		path, err = d.Uevent.DevicePath()
		if err != nil {
			return err
		}
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
		return fmt.Errorf("device: grub-probe gpt_parttype: %w", err)
	}
	d.GPTPartType = strings.TrimSpace(strings.ToLower(string(out)))
	return nil
}

func (d *Device) discoverFilesystem() error {
	if d.Path == "" {
		return ErrNoDeviceNode
	}
	// NOTE: grub-probe actually does not distinguish between ext2/ext3/ext4.
	// Technically they all have the same superblock magic which is why they
	// are being picked up as the same filesystem.
	// This is also not wrong because an ext4 filesystem can be opened by the
	// ext2 kernel driver.
	// The filesystems only distinguish themselves by the set of features.
	// This is not really a problem for us right now
	out, err := exec.Command("grub-probe", "-d", d.Path, "-t", "fs").Output()
	if err != nil {
		return fmt.Errorf("device: grub-probe fs: %w", err)
	}
	d.Filesystem = strings.TrimSpace(string(out))
	return nil
}

func (d *Device) discoverFilesystemLabel() error {
	if d.Path == "" {
		return ErrNoDeviceNode
	}
	out, err := exec.Command("grub-probe", "-d", d.Path, "-t", "fs_label").Output()
	if err != nil {
		return fmt.Errorf("device: grub-probe fs_label: %w", err)
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

func (d *Device) IsDiagPartition() bool {
	if d.IsPartition() {
		return strings.HasSuffix(d.GetPartitionName(), "-DIAG") || strings.HasSuffix(d.GetPartitionName(), "-diag") || strings.HasSuffix(d.FSLabel, "-DIAG") || strings.HasSuffix(d.FSLabel, "-diag")
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

func (d *Device) IsMounted() bool {
	// TODO: should maybe actually check /proc/mounts in case this was mounted/unmounted outside of ourselves?
	return d.MountPath != ""
}

func (d *Device) Mount() error {
	if d.Path == "" {
		return ErrNoDeviceNode
	}
	if d.IsMounted() {
		return ErrAlreadyMounted
	}

	if d.IsHedgehogIdentityPartition() {
		// ensure mount path exists and is a directory
		if err := ensureMountPath(MountPathHedgehogIdentity); err != nil {
			return err
		}

		// now mount it
		if err := unix.Mount(d.Path, MountPathHedgehogIdentity, FSExt4, unix.MS_NODEV|unix.MS_NOEXEC, ""); err != nil {
			return fmt.Errorf("device: mount: %w", err)
		}
		d.MountPath = MountPathHedgehogIdentity
	}

	if d.IsHedgehogLocationPartition() {
		// ensure mount path exists and is a directory
		if err := ensureMountPath(MountPathHedgehogLocation); err != nil {
			return err
		}

		// now mount it
		if err := unix.Mount(d.Path, MountPathHedgehogLocation, FSExt4, unix.MS_NODEV|unix.MS_NOEXEC, ""); err != nil {
			return fmt.Errorf("device: mount: %w", err)
		}
		d.MountPath = MountPathHedgehogLocation
	}

	return ErrUnsupportedMountForDevice
}

func (d *Device) Unmount() error {
	if !d.IsMounted() {
		return nil
	}
	if err := unix.Unmount(d.MountPath, 0); err != nil {
		return fmt.Errorf("device: umount: %w", err)
	}
	return nil
}

func ensureMountPath(path string) error {
	st, err := os.Stat(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("device: stat on mount path %s: %w", path, err)
	}
	if st != nil && !st.IsDir() {
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("device: removing file at mount path %s: %w", path, err)
		}
	}
	if err := os.MkdirAll(path, 0750); err != nil {
		return fmt.Errorf("device: mkdir on mount path %s: %w", path, err)
	}
	return nil
}

func (d *Device) MakeFilesystemForHedgehogIdentityPartition(force bool) error {
	if !d.IsHedgehogIdentityPartition() {
		return ErrUnsupportedMkfsForDevice
	}
	if d.Path == "" {
		return ErrNoDeviceNode
	}
	if d.Filesystem != "" && !force {
		return ErrFilesystemAlreadyCreated
	}
	if err := exec.Command("mkfs.ext4", "-L", FSLabelHedgehogIdentity, d.Path).Run(); err != nil {
		return fmt.Errorf("device: mkfs.ext4: %w", err)
	}
	d.Filesystem = FSExt4
	return nil
}
