package partitions

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"go.githedgehog.com/dasboot/pkg/exec"

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
	FS          FS
}

const (
	FSExt4 = "ext4"

	FSLabelONIE             = "ONIE-BOOT"
	FSLabelSONiC            = "SONiC-OS"
	FSLabelHedgehogIdentity = "HH_IDENTITY"
	FSLabelHedgehogLocation = "HH_LOCATION"

	GPTPartNameONIE             = FSLabelONIE
	GPTPartNameSONiC            = FSLabelSONiC
	GPTPartNameHedgehogIdentity = "HEDGEHOG_IDENTITY"
	GPTPartNameHedgehogLocation = "HEDGEHOG_LOCATION"

	GPTPartTypeONIE             = "7412f7d5-a156-4b13-81dc-867174929325"
	GPTPartTypeEFI              = "c12a7328-f81f-11d2-ba4b-00a0c93ec93b"
	GPTPartTypeHedgehogIdentity = "e982e2bd-867c-4d7a-89a2-9c5a9bc5dfdd"
	GPTPartTypeHedgehogLocation = "e23c5ebc-5f53-488c-959d-a4ab90befefe"

	MountPathHedgehogIdentity = "/mnt/hedgehog-identity"
	MountPathHedgehogLocation = "/mnt/hedgehog-location"
	MountPathSonic            = "/mnt/sonic"

	DefaultPartSizeHedgehogIdentityInMB int = 100

	blkrrpart = 0x125f //nolint: unused
)

var (
	ErrNoDeviceNode              = errors.New("device: no device node present")
	ErrDeviceNotPartition        = errors.New("device: not a partition")
	ErrDeviceNotDisk             = errors.New("device: not a disk")
	ErrBrokenDiscovery           = errors.New("device: broken discovery")
	ErrUnsupportedMkfsForDevice  = errors.New("device: unsupported device for mkfs")
	ErrAlreadyMounted            = errors.New("device: already mounted")
	ErrNotMounted                = errors.New("device: not mounted")
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
		info, err := osStat(p)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("device: ensure device path: stat: %w", err)
		}
		if info != nil {
			if err := osRemove(p); err != nil {
				return fmt.Errorf("device: ensure device path: remove %s: %w", p, err)
			}
		}
		if err := unixMknod(p, unix.S_IFBLK, int(unix.Mkdev(major, minor))); err != nil {
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

func (d *Device) IsSonicPartition() bool {
	if d.IsPartition() {
		return d.FSLabel == FSLabelSONiC || d.GetPartitionName() == GPTPartNameSONiC
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
	// TODO: this does not seem to be enough nowadays
	// We should find out what exactly `partprobe` does and replicate the calls directly.
	// It's probably another set of ioctls apart from blkrrpart
	// NOTE: don't delete! keep for now until we solve the TODO
	//
	// f, err := os.Open(d.Path)
	// if err != nil {
	// 	return err
	// }
	// if _, err = unixIoctlGetInt(int(f.Fd()), blkrrpart); err != nil {
	// 	return fmt.Errorf("device: unable to re-read partition table: %ww", err)
	// }
	if err := exec.Command("partprobe", d.Path).Run(); err != nil {
		return fmt.Errorf("device: unable to re-read partition table: partprobe: %w", err)
	}
	return nil
}

// IsMounted will check /proc/mounts to check if the
// device is already mounted.
//
// It will update `MountPath` if it is. If the filesystem
// is mounted multiple times the first entry of found in
// /proc/mounts will be taken and used for `Mountpath`.
//
// NOTE: This is not short-circuited and cached on purpose.
// If a third-party in the meantime is mounting/unmounting
// the device, we can not rely on internal state, and
// receiving notifications for this from the kernel is
// overkill for its purpose here.
func (d *Device) IsMounted() bool {
	// no need to check
	if d.Path == "" {
		return false
	}

	// NOTE: /proc/mounts is notorious for being broken
	// However, for our "simple" use-case in ONIE, I don't
	// think we need to worry about a better alternative
	// unless we really run into an issue here.
	procMountsPath := filepath.Join(rootPath, "proc", "mounts")
	f, err := os.Open(procMountsPath)
	if err != nil {
		return false
	}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		split := strings.SplitN(line, " ", 3)
		// if this is not 3, then we have a bogus procfs line
		if len(split) != 3 {
			continue
		}
		if split[0] == d.Path {
			d.MountPath = unescapeMountPath(split[1])
			if d.FS != nil {
				d.FS.SetBase(d.MountPath)
			}
			return true
		}
	}
	return false
}

// unescapeMountPath is shamelessly taken and adopted from:
// https://github.com/moby/sys/blob/main/mountinfo/mountinfo_linux.go#L167
//
// Golang developers did not see a need to add unescaping of octals into
// `strconv.Unquote`, so we need to have our own.
func unescapeMountPath(path string) string {
	// try to avoid copying
	if strings.IndexByte(path, '\\') == -1 {
		return path
	}

	// The following code is UTF-8 transparent as it only looks for some
	// specific characters (backslash and 0..7) with values < utf8.RuneSelf,
	// and everything else is passed through as is.
	buf := make([]byte, len(path))
	bufLen := 0
	for i := 0; i < len(path); i++ {
		if path[i] != '\\' {
			buf[bufLen] = path[i]
			bufLen++
			continue
		}
		s := path[i:]
		if len(s) < 4 {
			// too short
			return path
		}
		c := s[1]
		switch c {
		case '0', '1', '2', '3', '4', '5', '6', '7':
			v := c - '0'
			for j := 2; j < 4; j++ { // one digit already; two more
				if s[j] < '0' || s[j] > '7' {
					return path
				}
				x := s[j] - '0'
				v = (v << 3) | x
			}
			buf[bufLen] = v
			bufLen++
			i += 3
			continue
		default:
			return path
		}
	}

	return string(buf[:bufLen])
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
		mountPath := filepath.Join(rootPath, MountPathHedgehogIdentity)
		if err := ensureMountPath(mountPath); err != nil {
			return err
		}

		// now mount it
		if err := unixMount(d.Path, mountPath, FSExt4, unix.MS_NODEV|unix.MS_NOEXEC, ""); err != nil {
			return fmt.Errorf("device: mount: %w", err)
		}
		d.MountPath = mountPath
		if d.FS != nil {
			d.FS.SetBase(d.MountPath)
		}
		return nil
	}

	if d.IsHedgehogLocationPartition() {
		// ensure mount path exists and is a directory
		mountPath := filepath.Join(rootPath, MountPathHedgehogLocation)
		if err := ensureMountPath(mountPath); err != nil {
			return err
		}

		// now mount it
		if err := unixMount(d.Path, mountPath, FSExt4, unix.MS_NODEV|unix.MS_NOEXEC, ""); err != nil {
			return fmt.Errorf("device: mount: %w", err)
		}
		d.MountPath = mountPath
		if d.FS != nil {
			d.FS.SetBase(d.MountPath)
		}
		return nil
	}

	if d.IsSonicPartition() {
		// ensure mount path exists and is a director
		mountPath := filepath.Join(rootPath, MountPathSonic)
		if err := ensureMountPath(mountPath); err != nil {
			return err
		}

		// now mount it
		if err := unixMount(d.Path, mountPath, FSExt4, unix.MS_NODEV, ""); err != nil {
			return fmt.Errorf("device: mount: %w", err)
		}
		d.MountPath = mountPath
		if d.FS != nil {
			d.FS.SetBase(d.MountPath)
		}
		return nil
	}

	return ErrUnsupportedMountForDevice
}

func (d *Device) Unmount() error {
	if !d.IsMounted() {
		return nil
	}
	if err := unixUnmount(d.MountPath, 0); err != nil {
		return fmt.Errorf("device: umount: %w", err)
	}
	d.MountPath = ""
	if d.FS != nil {
		d.FS.SetBase(d.MountPath)
	}
	return nil
}

func ensureMountPath(path string) error {
	st, err := osStat(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("device: stat on mount path %s: %w", path, err)
	}
	if st != nil && !st.IsDir() {
		if err := osRemove(path); err != nil {
			return fmt.Errorf("device: removing file at mount path %s: %w", path, err)
		}
	}
	if err := osMkdirAll(path, 0750); err != nil {
		return fmt.Errorf("device: mkdir on mount path %s: %w", path, err)
	}
	return nil
}

// MakeFilesystemForHedgehogIdentityPartition will create a filesystem for
// the Hedgehog Identity partition if `IsHedgehogIdentityPartition() == true`.
// It will update `d.Filesystem` and `d.FSLabel` accordingly on success.
//
// If the Hedgehog Identity Partition already exists, the function will return
// with `ErrFilesystemAlreadyCreated`. However, if there is already a filesystem
// but **not** the Hedgehog Identity Partition, then it will continue and
// recreate the filesystem for the Hedgehog Identity Partition regardless.
//
// If you want to recreate an existing Hedgehog Identity Partion filesystem
// (which is determined by the `FSLabel`), then set the `force` flag to true.
// See the above paragraph that this is not needed if the filesystem is
// a different filesystem. This is particularly useful if you are creating
// a filesystem on a disk which already had an ext4 filesystem before (for
// example from a previous non-Hedgehog SONiC installation), and you want to
// continue creating the filesystem without force in this case.
func (d *Device) MakeFilesystemForHedgehogIdentityPartition(force bool) error {
	if !d.IsHedgehogIdentityPartition() {
		return ErrUnsupportedMkfsForDevice
	}
	return d.makeFilesystem(FSExt4, FSLabelHedgehogIdentity, force)
}

func (d *Device) makeFilesystem(fsType, fsLabel string, force bool) error {
	if d.Path == "" {
		return ErrNoDeviceNode
	}
	if d.Filesystem != "" && d.FSLabel == FSLabelHedgehogIdentity && !force {
		return ErrFilesystemAlreadyCreated
	}
	var fsOpts []string
	switch fsType {
	case FSExt4:
		// if a filesystem does already exist, which for example can be the case when
		// SONiC was already installed previously, we need to make sure that the
		// "default" answer of N is not being selected by mkfs.ext4, which would
		// abort the command but with a successful exit code of 0.
		fsOpts = []string{"-F"}
	}
	args := []string{"-L", fsLabel}
	if len(fsOpts) > 0 {
		args = append(args, fsOpts...)
	}
	args = append(args, d.Path)
	if err := exec.Command("mkfs."+fsType, args...).Run(); err != nil {
		return fmt.Errorf("device: mkfs.%s: %w", fsType, err)
	}
	d.Filesystem = fsType
	d.FSLabel = fsLabel
	return nil
}
