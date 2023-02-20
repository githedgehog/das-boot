package partitions

import (
	"errors"
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

	errFilepathRelFailed := errors.New("filepath.Rel failed")

	tests := []struct {
		name        string
		want        Devices
		wantErr     bool
		wantErrToBe error
		filepathRel func(basepath string, targpath string) (string, error)
	}{
		{
			name:    "test",
			wantErr: false,
			want: Devices{
				part,
				disk,
			},
		},
		{
			name:        "walkdir fails",
			wantErr:     true,
			wantErrToBe: errFilepathRelFailed,
			want:        nil,
			filepathRel: func(basepath, targpath string) (string, error) {
				return "", errFilepathRelFailed
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.filepathRel != nil {
				oldFilepathRel := filepathRel
				defer func() {
					filepathRel = oldFilepathRel
				}()
				filepathRel = tt.filepathRel
			}
			got, err := Discover()
			if (err != nil) != tt.wantErr {
				t.Errorf("Discover() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr && tt.wantErrToBe != nil {
				if !errors.Is(err, tt.wantErrToBe) {
					t.Errorf("Discover() error = %v, wantErrToBe %v", err, tt.wantErrToBe)
					return
				}
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Discover() got = %v, want %v", got, tt.want)
				return

			}
			// for _, gotDev := range got {
			// 	var found bool
			// 	for _, wantDev := range tt.want {
			// 		if reflect.DeepEqual(*gotDev, *wantDev) {
			// 			found = true
			// 		}
			// 	}
			// 	if !found {
			// 		t.Errorf("not found")
			// 	}
			// }
		})
	}
}
