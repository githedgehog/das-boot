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
