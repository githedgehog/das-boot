package partitions

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
)

var sysfsPath = filepath.Join(rootPath, "sys")

func Discover() (Devices, error) {
	var ret []*Device
	// var walkFunc func(path string, d fs.FileInfo, err error) error
	walkFunc := func(path string, d fs.DirEntry, err error) error {
		// fmt.Printf("%s\n", path)
		if d.Name() == "uevent" {
			entry, err := ReadUevent(path)
			if err != nil {
				return nil
			}
			dev := &Device{
				Uevent:    entry,
				SysfsPath: filepath.Dir(path),
			}
			ret = append(ret, dev)
		}
		return nil
	}
	if err := WalkDir(filepath.Join(sysfsPath, "block"), walkFunc, 1, "subsystem", "device", "bdi"); err != nil {
		return nil, fmt.Errorf("partitions: discover: %w", err)
	}

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
			continue
		}
		if dev.IsPartition() {
			_ = dev.discoverPartitionType()
			_ = dev.discoverFilesystemLabel()
		}
		// fmt.Printf("%#v\n", *dev)
	}
	return ret, nil
}
