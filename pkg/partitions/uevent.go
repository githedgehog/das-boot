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
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// Uevent represents the contents of a "uevent" file as it is exposed through sysfs
// for block storage devices at /sys/block
type Uevent map[string]string

func ReadUevent(path string) (Uevent, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	ret := make(map[string]string)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		split := strings.SplitN(line, "=", 2)
		if len(split) != 2 {
			continue
		}
		ret[strings.TrimSpace(split[0])] = strings.TrimSpace(split[1])
	}
	return ret, scanner.Err()
}

var (
	// ErrInvalidUevent will be returned if this is an invalid uevent object from the block hierarchy
	ErrInvalidUevent      = errors.New("uevent: invalid block uevent object")
	ErrNotABlockDevice    = errors.New("uevent: not a block device")
	ErrStatNotFromSyscall = errors.New("uevent: stat not from syscall")
)

// internal constants for accessing the uevent map
const (
	UeventDevtype  = "DEVTYPE"
	UeventDevname  = "DEVNAME"
	UeventPartn    = "PARTN"
	UeventPartname = "PARTNAME"
	UeventMajor    = "MAJOR"
	UeventMinor    = "MINOR"
	UeventDiskseq  = "DISKSEQ"
)

// known values for some uevent map entries
const (
	UeventDevtypeDisk      = "disk"
	UeventDevtypePartition = "partition"
)

// our C port of file testing as Golang doesn't seem to come with block device testing?!
// or I humbly misunderstood how this is supposed to get used - mea culpa
const (
	s_ifmt  uint32 = 0170000
	s_ifblk uint32 = 060000
)

func s_isblk(mode uint32) bool {
	return mode&s_ifmt == s_ifblk
}

func (u Uevent) IsDisk() bool {
	val, ok := u[UeventDevtype]
	if !ok {
		return false
	}
	return val == UeventDevtypeDisk
}

func (u Uevent) IsPartition() bool {
	val, ok := u[UeventDevtype]
	if !ok {
		return false
	}
	return val == UeventDevtypePartition
}

func (u Uevent) GetPartitionNumber() int {
	val, ok := u[UeventPartn]
	if !ok {
		return -1
	}
	ret, err := strconv.ParseUint(val, 0, 8)
	if err != nil {
		return -1
	}
	return int(ret)
}

func (u Uevent) GetPartitionName() string {
	val, ok := u[UeventPartname]
	if !ok {
		return ""
	}
	return val
}

func (u Uevent) GetMajorMinor() (uint32, uint32, error) {
	maj, ok := u[UeventMajor]
	if !ok {
		return 0, 0, ErrInvalidUevent
	}
	min, ok := u[UeventMinor]
	if !ok {
		return 0, 0, ErrInvalidUevent
	}

	majUint, err := strconv.ParseUint(maj, 0, 32)
	if err != nil {
		return 0, 0, fmt.Errorf("uevent: parsing major: %w", err)
	}
	minUint, err := strconv.ParseUint(min, 0, 32)
	if err != nil {
		return 0, 0, fmt.Errorf("uevent: parsing minor: %w", err)
	}
	return uint32(majUint), uint32(minUint), nil
}

func (u Uevent) GetDeviceName() string {
	val, ok := u[UeventDevname]
	if !ok {
		return ""
	}
	return val
}

// DevicePath returns the device path from the uevent.
// It checks that the file exists and is a block
// device. It throws an error otherwise.
func (u Uevent) DevicePath() (string, error) {
	val, ok := u[UeventDevname]
	if !ok {
		return "", ErrInvalidUevent
	}

	// guess the file name (that's a very good guess)
	path := filepath.Join(rootPath, "dev", val)
	gostat, err := osStat(path)
	if err != nil {
		return "", fmt.Errorf("uevent: %w", err)
	}

	stat, ok := gostat.Sys().(*syscall.Stat_t)
	if !ok {
		return "", ErrStatNotFromSyscall
	}

	if !s_isblk(stat.Mode) {
		return "", ErrNotABlockDevice
	}

	return path, nil
}
