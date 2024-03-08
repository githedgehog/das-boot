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
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDiscover(t *testing.T) {
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	oldRootPath := rootPath
	rootPath = filepath.Join(pwd, "testdata", "Discover")
	defer func() {
		rootPath = oldRootPath
	}()

	// test fixtures
	part := &Device{
		Uevent: Uevent{
			UeventDevname:  "loop0p1",
			UeventDevtype:  UeventDevtypePartition,
			UeventDiskseq:  "1",
			UeventMajor:    "7",
			UeventMinor:    "1",
			UeventPartn:    "1",
			UeventPartname: "EFI system partition",
		},
		SysfsPath: filepath.Join(rootPath, "sys", "block", "loop0", "loop0p1"),
		FS:        &fsOs{},
	}
	disk := &Device{
		Uevent: Uevent{
			UeventDevname: "loop0",
			UeventDevtype: UeventDevtypeDisk,
			UeventDiskseq: "2",
			UeventMajor:   "7",
			UeventMinor:   "0",
		},
		SysfsPath: filepath.Join(rootPath, "sys", "block", "loop0"),
		FS:        &fsOs{},
	}
	disk.Partitions = []*Device{part}
	part.Disk = disk

	tests := []struct {
		name string
		want Devices
	}{
		{
			name: "test",
			want: Devices{
				part,
				disk,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Discover()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Discover() got = %v, want %v", got, tt.want)
				return
			}
		})
	}
}
