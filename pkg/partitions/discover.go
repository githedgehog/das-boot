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
	"io/fs"
	"path/filepath"
	"strings"

	dbfilepath "go.githedgehog.com/dasboot/pkg/filepath"
	"go.githedgehog.com/dasboot/pkg/log"

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
				log.L().Warn("ReadUevent failed", zap.Error(err))
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
			log.L().Warn("ensuring device path failed", zap.String("devname", dev.GetDeviceName()), zap.Error(err))
			// technically that might be faster, but let's just try everything anyways
			// they will most likely abort because of the missing device node anyways
			// continue
		}
		if err := dev.discoverFilesystem(); err != nil {
			log.L().Debug("discover filesystem failed", zap.String("devname", dev.GetDeviceName()), zap.Error(err))
		}
		if err := dev.discoverFilesystemLabel(); err != nil {
			log.L().Debug("discover filesystem label failed", zap.String("devname", dev.GetDeviceName()), zap.Error(err))
		}
		if dev.IsPartition() {
			if err := dev.discoverPartitionType(); err != nil {
				log.L().Debug("discover partition type failed", zap.String("devname", dev.GetDeviceName()), zap.Error(err))
			}
		}
	}
	return ret
}
