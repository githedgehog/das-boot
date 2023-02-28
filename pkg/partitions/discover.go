package partitions

import (
	"io/fs"
	"path/filepath"
	"strings"

	dbfilepath "go.githedgehog.com/dasboot/pkg/filepath"

	"go.uber.org/zap"
)

func Discover() Devices {
	var ret []*Device
	walkFunc := func(path string, d fs.DirEntry, err error) error {
		// fmt.Printf("%s\n", path)
		if d.Name() == "uevent" {
			entry, err := ReadUevent(path)
			if err != nil {
				// we will just log an error but move on
				Logger.Warn("ReadUevent failed", zap.Error(err))
				return nil
			}
			dev := &Device{
				Uevent:    entry,
				SysfsPath: filepath.Dir(path),
				FS:        &fsOs{},
			}
			ret = append(ret, dev)
		}
		return nil
	}
	// we don't fail in `walkFunc` so this does not fail
	_ = dbfilepath.WalkDir(filepath.Join(rootPath, "sys", "block"), walkFunc, 1, "subsystem", "device", "bdi")

	// stupid, but I don't know right now what else to do
	// this identifies partitions and disks and their relationships
	for _, dev := range ret {
		if dev.IsDisk() {
			for _, dev2 := range ret {
				if dev2.IsPartition() && strings.HasPrefix(dev2.SysfsPath, dev.SysfsPath) {
					dev2.Disk = dev
					dev.Partitions = append(dev.Partitions, dev2)
				}
			}
		}
	}

	// we are only interested in partitions for the next discovery phase
	// because we *know* that all we care about is located on partitions
	// However, we should ensure for all devices that a block device node
	// is available
	for _, dev := range ret {
		if err := dev.ensureDevicePath(); err != nil {
			Logger.Warn("ensuring device path failed", zap.String("devname", dev.GetDeviceName()), zap.Error(err))
			// technically that might be faster, but let's just try everything anyways
			// they will most likely abort because of the missing device node anyways
			// continue
		}
		if err := dev.discoverFilesystem(); err != nil {
			Logger.Debug("discover filesystem failed", zap.String("devname", dev.GetDeviceName()), zap.Error(err))
		}
		if err := dev.discoverFilesystemLabel(); err != nil {
			Logger.Debug("discover filesystem label failed", zap.String("devname", dev.GetDeviceName()), zap.Error(err))
		}
		if dev.IsPartition() {
			if err := dev.discoverPartitionType(); err != nil {
				Logger.Debug("discover partition type failed", zap.String("devname", dev.GetDeviceName()), zap.Error(err))
			}
		}
	}
	return ret
}
