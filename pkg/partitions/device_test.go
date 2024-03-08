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
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"go.githedgehog.com/dasboot/pkg/exec"
	"go.githedgehog.com/dasboot/test/mock/mockexec"

	gomock "github.com/golang/mock/gomock"
	"golang.org/x/sys/unix"
)

func TestDevice_IsEFIPartition(t *testing.T) {
	tests := []struct {
		name   string
		device *Device
		want   bool
	}{
		{
			name: "is EFI partition",
			device: &Device{
				Uevent: Uevent{
					UeventDevtype: UeventDevtypePartition,
				},
				GPTPartType: GPTPartTypeEFI,
			},
			want: true,
		},
		{
			name: "is not EFI partition",
			device: &Device{
				Uevent: Uevent{
					UeventDevtype: UeventDevtypePartition,
				},
				GPTPartType: GPTPartTypeONIE,
			},
			want: false,
		},
		{
			name: "is not a partition",
			device: &Device{
				Uevent: Uevent{
					UeventDevtype: UeventDevtypeDisk,
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.device.IsEFIPartition(); got != tt.want {
				t.Errorf("Device.IsEFIPartition() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDevice_IsONIEPartition(t *testing.T) {
	tests := []struct {
		name   string
		device *Device
		want   bool
	}{
		{
			name: "is ONIE partition",
			device: &Device{
				Uevent: Uevent{
					UeventDevtype: UeventDevtypePartition,
				},
				GPTPartType: GPTPartTypeONIE,
			},
			want: true,
		},
		{
			name: "is not ONIE partition",
			device: &Device{
				Uevent: Uevent{
					UeventDevtype: UeventDevtypePartition,
				},
				GPTPartType: GPTPartTypeEFI,
			},
			want: false,
		},
		{
			name: "is not a partition",
			device: &Device{
				Uevent: Uevent{
					UeventDevtype: UeventDevtypeDisk,
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.device.IsONIEPartition(); got != tt.want {
				t.Errorf("Device.IsONIEPartition() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDevice_IsDiagPartition(t *testing.T) {
	tests := []struct {
		name   string
		device *Device
		want   bool
	}{
		{
			name: "is Diag partition through partname",
			device: &Device{
				Uevent: Uevent{
					UeventDevtype:  UeventDevtypePartition,
					UeventPartname: "HH-DIAG",
				},
			},
			want: true,
		},
		{
			name: "is Diag partition through partname lower-case",
			device: &Device{
				Uevent: Uevent{
					UeventDevtype:  UeventDevtypePartition,
					UeventPartname: "hh-diag",
				},
			},
			want: true,
		},
		{
			name: "is Diag partition through filesystem label",
			device: &Device{
				Uevent: Uevent{
					UeventDevtype: UeventDevtypePartition,
				},
				FSLabel: "HH-DIAG",
			},
			want: true,
		},
		{
			name: "is Diag partition through filesystem label lower-case",
			device: &Device{
				Uevent: Uevent{
					UeventDevtype: UeventDevtypePartition,
				},
				FSLabel: "hh-diag",
			},
			want: true,
		},
		{
			name: "is not Diag partition",
			device: &Device{
				Uevent: Uevent{
					UeventDevtype: UeventDevtypePartition,
				},
				GPTPartType: GPTPartTypeEFI,
			},
			want: false,
		},
		{
			name: "is not a partition",
			device: &Device{
				Uevent: Uevent{
					UeventDevtype: UeventDevtypeDisk,
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.device.IsDiagPartition(); got != tt.want {
				t.Errorf("Device.IsDiagPartition() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDevice_IsHedgehogIdentityPartition(t *testing.T) {
	tests := []struct {
		name   string
		device *Device
		want   bool
	}{
		{
			name: "is Hedgehog Identity partition through GPT partition type",
			device: &Device{
				Uevent: Uevent{
					UeventDevtype: UeventDevtypePartition,
				},
				GPTPartType: GPTPartTypeHedgehogIdentity,
			},
			want: true,
		},
		{
			name: "is Hedgehog Identity partition through GPT partition name",
			device: &Device{
				Uevent: Uevent{
					UeventDevtype:  UeventDevtypePartition,
					UeventPartname: GPTPartNameHedgehogIdentity,
				},
			},
			want: true,
		},
		{
			name: "is Hedgehog Identity partition through filesystem label",
			device: &Device{
				Uevent: Uevent{
					UeventDevtype: UeventDevtypePartition,
				},
				FSLabel: FSLabelHedgehogIdentity,
			},
			want: true,
		},
		{
			name: "is not Hedgehog Identity partition",
			device: &Device{
				Uevent: Uevent{
					UeventDevtype: UeventDevtypePartition,
				},
				GPTPartType: GPTPartTypeEFI,
			},
			want: false,
		},
		{
			name: "is not a partition",
			device: &Device{
				Uevent: Uevent{
					UeventDevtype: UeventDevtypeDisk,
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.device.IsHedgehogIdentityPartition(); got != tt.want {
				t.Errorf("Device.IsHedgehogIdentityPartition() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDevice_IsHedgehogLocationPartition(t *testing.T) {
	tests := []struct {
		name   string
		device *Device
		want   bool
	}{
		{
			name: "is Hedgehog Location partition through GPT partition type",
			device: &Device{
				Uevent: Uevent{
					UeventDevtype: UeventDevtypePartition,
				},
				GPTPartType: GPTPartTypeHedgehogLocation,
			},
			want: true,
		},
		{
			name: "is Hedgehog Location partition through GPT partition name",
			device: &Device{
				Uevent: Uevent{
					UeventDevtype:  UeventDevtypePartition,
					UeventPartname: GPTPartNameHedgehogLocation,
				},
			},
			want: true,
		},
		{
			name: "is Hedgehog Location partition through filesystem label",
			device: &Device{
				Uevent: Uevent{
					UeventDevtype: UeventDevtypePartition,
				},
				FSLabel: FSLabelHedgehogLocation,
			},
			want: true,
		},
		{
			name: "is not Hedgehog Location partition",
			device: &Device{
				Uevent: Uevent{
					UeventDevtype: UeventDevtypePartition,
				},
				GPTPartType: GPTPartTypeEFI,
			},
			want: false,
		},
		{
			name: "is not a partition",
			device: &Device{
				Uevent: Uevent{
					UeventDevtype: UeventDevtypeDisk,
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.device.IsHedgehogLocationPartition(); got != tt.want {
				t.Errorf("Device.IsHedgehogLocationPartition() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDevice_Delete(t *testing.T) {
	sgdiskCommandFailed := errors.New("sgdisk command failed")
	tests := []struct {
		name        string
		device      *Device
		wantErr     bool
		wantErrToBe error
		cmds        func(t *testing.T, ctrl *gomock.Controller) []exec.CommandFunc
	}{
		{
			name: "success",
			device: &Device{
				Uevent: Uevent{
					UeventDevtype: UeventDevtypePartition,
					UeventPartn:   "6",
				},
				Disk: &Device{
					Path: "/path/to/disk/device",
				},
			},
			cmds: func(t *testing.T, ctrl *gomock.Controller) []exec.CommandFunc {
				return []exec.CommandFunc{
					mockexec.MockCommand(t, ctrl, []string{"sgdisk", "-d", "6", "/path/to/disk/device"}, func(tc *mockexec.TestCmd) {
						tc.EXPECT().Run().Times(1).DoAndReturn(func() error {
							return tc.IsExpectedCommand()
						})
					}),
				}
			},
			wantErr: false,
		},
		{
			name: "not a partition",
			device: &Device{
				Uevent: Uevent{
					UeventDevtype: UeventDevtypeDisk,
				},
			},
			wantErr:     true,
			wantErrToBe: ErrDeviceNotPartition,
		},
		{
			name: "invalid uevent",
			device: &Device{
				Uevent: Uevent{
					UeventDevtype: UeventDevtypePartition,
					UeventPartn:   "invalid",
				},
			},
			wantErr:     true,
			wantErrToBe: ErrInvalidUevent,
		},
		{
			name: "broken discovery",
			device: &Device{
				Uevent: Uevent{
					UeventDevtype: UeventDevtypePartition,
					UeventPartn:   "6",
				},
				Disk: nil,
			},
			wantErr:     true,
			wantErrToBe: ErrBrokenDiscovery,
		},
		{
			name: "missing device node for disk",
			device: &Device{
				Uevent: Uevent{
					UeventDevtype: UeventDevtypePartition,
					UeventPartn:   "6",
				},
				Disk: &Device{},
			},
			wantErr:     true,
			wantErrToBe: ErrNoDeviceNode,
		},
		{
			name: "sgdisk command fails",
			device: &Device{
				Uevent: Uevent{
					UeventDevtype: UeventDevtypePartition,
					UeventPartn:   "6",
				},
				Disk: &Device{
					Path: "/path/to/disk/device",
				},
			},
			cmds: func(t *testing.T, ctrl *gomock.Controller) []exec.CommandFunc {
				return []exec.CommandFunc{
					mockexec.MockCommand(t, ctrl, []string{"sgdisk", "-d", "6", "/path/to/disk/device"}, func(tc *mockexec.TestCmd) {
						tc.EXPECT().Run().Times(1).DoAndReturn(func() error {
							if err := tc.IsExpectedCommand(); err != nil {
								return err
							}
							return sgdiskCommandFailed
						})
					}),
				}
			},
			wantErr:     true,
			wantErrToBe: sgdiskCommandFailed,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			if tt.cmds != nil {
				oldCommand := exec.Command
				defer func() {
					exec.Command = oldCommand
				}()
				cmds := mockexec.NewMockCommands(tt.cmds(t, ctrl))
				defer cmds.Finish()
				exec.Command = cmds.Command()
			}
			err := tt.device.Delete()
			if (err != nil) != tt.wantErr {
				t.Errorf("Device.Delete() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr && tt.wantErrToBe != nil {
				if !errors.Is(err, tt.wantErrToBe) {
					t.Errorf("Device.Delete() error = %v, wantErrToBe %v", err, tt.wantErrToBe)
					return
				}
			}
		})
	}
}

func TestDevice_ReReadPartitionTable(t *testing.T) {
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	errIoctlFailed := errors.New("ioctl failed")
	tests := []struct {
		name   string
		device *Device
		// NOTE: don't delete! keep for now until we solve the TODO
		// unixIoctlGetInt func(fd int, req uint) (int, error)
		cmds        func(t *testing.T, ctrl *gomock.Controller) []exec.CommandFunc
		wantErr     bool
		wantErrToBe error
	}{
		// NOTE: don't delete! keep for now until we solve the TODO
		// {
		// 	name: "success",
		// 	device: &Device{
		// 		Uevent: Uevent{
		// 			UeventDevtype: UeventDevtypeDisk,
		// 		},
		// 		Path: filepath.Join(pwd, "testdata", "ReReadPartitionTable", "device"),
		// 	},
		// 	unixIoctlGetInt: func(fd int, req uint) (int, error) {
		// 		if fd <= 2 {
		// 			return 0, fmt.Errorf("not an opened device file")
		// 		}
		// 		if req != blkrrpart {
		// 			return 0, fmt.Errorf("not a BLKRRPART ioctl")
		// 		}
		// 		return 42, nil
		// 	},
		// 	wantErr: false,
		// },
		{
			name: "success",
			device: &Device{
				Uevent: Uevent{
					UeventDevtype: UeventDevtypeDisk,
				},
				Path: filepath.Join(pwd, "testdata", "ReReadPartitionTable", "device"),
			},
			cmds: func(t *testing.T, ctrl *gomock.Controller) []exec.CommandFunc {
				return []exec.CommandFunc{
					mockexec.MockCommand(t, ctrl, []string{"partprobe", filepath.Join(pwd, "testdata", "ReReadPartitionTable", "device")}, func(tc *mockexec.TestCmd) {
						tc.EXPECT().Run().Times(1).DoAndReturn(func() error {
							return tc.IsExpectedCommand()
						})
					}),
				}
			},
			wantErr: false,
		},
		{
			name: "partprobe fails",
			device: &Device{
				Uevent: Uevent{
					UeventDevtype: UeventDevtypeDisk,
				},
				Path: filepath.Join(pwd, "testdata", "ReReadPartitionTable", "device"),
			},
			cmds: func(t *testing.T, ctrl *gomock.Controller) []exec.CommandFunc {
				return []exec.CommandFunc{
					mockexec.MockCommand(t, ctrl, []string{"partprobe", filepath.Join(pwd, "testdata", "ReReadPartitionTable", "device")}, func(tc *mockexec.TestCmd) {
						tc.EXPECT().Run().Times(1).DoAndReturn(func() error {
							if err := tc.IsExpectedCommand(); err != nil {
								return err
							}
							return errIoctlFailed
						})
					}),
				}
			},
			wantErr:     true,
			wantErrToBe: errIoctlFailed,
		},
		{
			name: "not a disk",
			device: &Device{
				Uevent: Uevent{
					UeventDevtype: UeventDevtypePartition,
				},
			},
			wantErr:     true,
			wantErrToBe: ErrDeviceNotDisk,
		},
		{
			name: "device node missing",
			device: &Device{
				Uevent: Uevent{
					UeventDevtype: UeventDevtypeDisk,
				},
				Path: "",
			},
			wantErr:     true,
			wantErrToBe: ErrNoDeviceNode,
		},
		// NOTE: don't delete! keep for now until we solve the TODO
		// {
		// 	name: "device node missing for path to missing file",
		// 	device: &Device{
		// 		Uevent: Uevent{
		// 			UeventDevtype: UeventDevtypeDisk,
		// 		},
		// 		Path: filepath.Join(pwd, "testdata", "ReReadPartitionTable", "missing"),
		// 	},
		// 	wantErr:     true,
		// 	wantErrToBe: os.ErrNotExist,
		// },
		// NOTE: don't delete! keep for now until we solve the TODO
		// {
		// 	name: "ioctl fails",
		// 	device: &Device{
		// 		Uevent: Uevent{
		// 			UeventDevtype: UeventDevtypeDisk,
		// 		},
		// 		Path: filepath.Join(pwd, "testdata", "ReReadPartitionTable", "device"),
		// 	},
		// 	unixIoctlGetInt: func(fd int, req uint) (int, error) {
		// 		return 0, errIoctlFailed
		// 	},
		// 	wantErr:     true,
		// 	wantErrToBe: errIoctlFailed,
		// },
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// NOTE: don't delete! keep for now until we solve the TODO
			// if tt.unixIoctlGetInt != nil {
			// 	oldUnixIoctlGetInt := unixIoctlGetInt
			// 	defer func() {
			// 		unixIoctlGetInt = oldUnixIoctlGetInt
			// 	}()
			// 	unixIoctlGetInt = tt.unixIoctlGetInt
			// }
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			if tt.cmds != nil {
				oldCommand := exec.Command
				defer func() {
					exec.Command = oldCommand
				}()
				cmds := mockexec.NewMockCommands(tt.cmds(t, ctrl))
				defer cmds.Finish()
				exec.Command = cmds.Command()
			}
			err := tt.device.ReReadPartitionTable()
			if (err != nil) != tt.wantErr {
				t.Errorf("Device.ReReadPartitionTable() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr && tt.wantErrToBe != nil {
				if !errors.Is(err, tt.wantErrToBe) {
					t.Errorf("Device.ReReadPartitionTable() error = %v, wantErrToBe %v", err, tt.wantErrToBe)
					return
				}
			}
		})
	}
}

func TestDevice_IsMounted(t *testing.T) {
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	tests := []struct {
		name          string
		device        *Device
		rootPath      string
		want          bool
		wantMountPath string
	}{
		{
			name: "is mounted",
			device: &Device{
				Path: "/dev/vda3",
				FS:   &fsOs{},
			},
			rootPath:      filepath.Join(pwd, "testdata", "IsMounted", "one"),
			wantMountPath: "/mnt/hedgehog-identity",
			want:          true,
		},
		{
			name: "is mounted on path with space",
			device: &Device{
				Path: "/dev/vda3",
			},
			rootPath:      filepath.Join(pwd, "testdata", "IsMounted", "two"),
			wantMountPath: "/mnt/t t",
			want:          true,
		},
		{
			name: "is mounted with bogus line",
			device: &Device{
				Path: "/dev/vda3",
			},
			rootPath:      filepath.Join(pwd, "testdata", "IsMounted", "three"),
			wantMountPath: "/mnt/hedgehog-identity",
			want:          true,
		},
		{
			name: "is not mounted",
			device: &Device{
				Path: "/dev/vda3",
			},
			rootPath: filepath.Join(pwd, "testdata", "IsMounted", "four"),
			want:     false,
		},
		{
			name:   "has no device node",
			device: &Device{},
			want:   false,
		},
		{
			name: "fails to open mounts files",
			device: &Device{
				Path: "/dev/vda3",
			},
			rootPath: "/does/not/exist",
			want:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.rootPath != "" {
				oldRootPath := rootPath
				defer func() {
					rootPath = oldRootPath
				}()
				rootPath = tt.rootPath
			}
			if got := tt.device.IsMounted(); got != tt.want {
				t.Errorf("Device.IsMounted() = %v, want %v", got, tt.want)
				return
			}
			if tt.device.MountPath != tt.wantMountPath {
				t.Errorf("Device.IsMounted() Device.MountPath = %v, want %v", tt.device.MountPath, tt.wantMountPath)
				return
			}
		})
	}
}

func TestDevice_Mount(t *testing.T) {
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	goodPath := filepath.Join(pwd, "testdata", "Mount")
	errOsStatFailed := errors.New("os.Stat failed unrecoverably")
	errUnixMountFailed := errors.New("unix.Mount failed")

	tests := []struct {
		name          string
		device        *Device
		wantErr       bool
		wantErrToBe   error
		wantMountPath string
		rootPath      string
		unixMount     func(source string, target string, fstype string, flags uintptr, data string) error
		osStat        func(name string) (fs.FileInfo, error)
	}{
		{
			name: "success for hedgehog identity partition",
			device: &Device{
				Uevent: Uevent{
					UeventDevtype: UeventDevtypePartition,
				},
				GPTPartType: GPTPartTypeHedgehogIdentity,
				Path:        "/path/to/device",
				FS:          &fsOs{},
			},
			rootPath: goodPath,
			unixMount: func(source, target, fstype string, flags uintptr, data string) error {
				if source != "/path/to/device" {
					return fmt.Errorf("source is wrong: %s", source)
				}
				if target != filepath.Join(goodPath, MountPathHedgehogIdentity) {
					return fmt.Errorf("target is wrong: %s", target)
				}
				if fstype != FSExt4 {
					return fmt.Errorf("unexpected filesystem: %s", fstype)
				}
				if flags != (unix.MS_NODEV | unix.MS_NOEXEC) {
					return fmt.Errorf("flags are unexpected: 0x%x", flags)
				}
				if data != "" {
					return fmt.Errorf("unexpected data: %s", data)
				}
				return nil
			},
			wantMountPath: filepath.Join(goodPath, MountPathHedgehogIdentity),
			wantErr:       false,
		},
		{
			name: "success for hedgehog location partition",
			device: &Device{
				Uevent: Uevent{
					UeventDevtype: UeventDevtypePartition,
				},
				GPTPartType: GPTPartTypeHedgehogLocation,
				Path:        "/path/to/device",
				FS:          &fsOs{},
			},
			rootPath: goodPath,
			unixMount: func(source, target, fstype string, flags uintptr, data string) error {
				if source != "/path/to/device" {
					return fmt.Errorf("source is wrong: %s", source)
				}
				if target != filepath.Join(goodPath, MountPathHedgehogLocation) {
					return fmt.Errorf("target is wrong: %s", target)
				}
				if fstype != FSExt4 {
					return fmt.Errorf("unexpected filesystem: %s", fstype)
				}
				if flags != (unix.MS_NODEV | unix.MS_NOEXEC) {
					return fmt.Errorf("flags are unexpected: 0x%x", flags)
				}
				if data != "" {
					return fmt.Errorf("unexpected data: %s", data)
				}
				return nil
			},
			wantMountPath: filepath.Join(goodPath, MountPathHedgehogLocation),
			wantErr:       false,
		},
		{
			name: "no device node",
			device: &Device{
				Path: "",
			},
			wantErr:     true,
			wantErrToBe: ErrNoDeviceNode,
		},
		{
			name: "already mounted",
			device: &Device{
				Path: "/dev/vda3",
			},
			rootPath:      filepath.Join(pwd, "testdata", "IsMounted", "one"),
			wantErr:       true,
			wantErrToBe:   ErrAlreadyMounted,
			wantMountPath: "/mnt/hedgehog-identity",
		},
		{
			name: "hedgehog identity partition fails ensureMount unrecoverably",
			device: &Device{
				Uevent: Uevent{
					UeventDevtype: UeventDevtypePartition,
				},
				GPTPartType: GPTPartTypeHedgehogIdentity,
				Path:        "/path/to/device",
			},
			rootPath: goodPath,
			osStat: func(name string) (fs.FileInfo, error) {
				return nil, errOsStatFailed
			},
			wantErr:     true,
			wantErrToBe: errOsStatFailed,
		},
		{
			name: "hedgehog location partition fails ensureMount unrecoverably",
			device: &Device{
				Uevent: Uevent{
					UeventDevtype: UeventDevtypePartition,
				},
				GPTPartType: GPTPartTypeHedgehogLocation,
				Path:        "/path/to/device",
			},
			rootPath: goodPath,
			osStat: func(name string) (fs.FileInfo, error) {
				return nil, errOsStatFailed
			},
			wantErr:     true,
			wantErrToBe: errOsStatFailed,
		},
		{
			name: "hedgehog identity partition fails ensureMount unrecoverably",
			device: &Device{
				Uevent: Uevent{
					UeventDevtype: UeventDevtypePartition,
				},
				GPTPartType: GPTPartTypeHedgehogIdentity,
				Path:        "/path/to/device",
			},
			rootPath: goodPath,
			unixMount: func(source, target, fstype string, flags uintptr, data string) error {
				return errUnixMountFailed
			},
			wantErr:     true,
			wantErrToBe: errUnixMountFailed,
		},
		{
			name: "hedgehog location partition fails ensureMount unrecoverably",
			device: &Device{
				Uevent: Uevent{
					UeventDevtype: UeventDevtypePartition,
				},
				GPTPartType: GPTPartTypeHedgehogLocation,
				Path:        "/path/to/device",
			},
			rootPath: goodPath,
			unixMount: func(source, target, fstype string, flags uintptr, data string) error {
				return errUnixMountFailed
			},
			wantErr:     true,
			wantErrToBe: errUnixMountFailed,
		},
		{
			name: "unsupported device for mount",
			device: &Device{
				Uevent: Uevent{
					UeventDevtype: UeventDevtypePartition,
				},
				Path: "/path/to/device",
			},
			rootPath:    goodPath,
			wantErr:     true,
			wantErrToBe: ErrUnsupportedMountForDevice,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.rootPath != "" {
				oldRootPath := rootPath
				defer func() {
					rootPath = oldRootPath
				}()
				rootPath = tt.rootPath
			}
			if tt.unixMount != nil {
				oldUnixMount := unixMount
				defer func() {
					unixMount = oldUnixMount
				}()
				unixMount = tt.unixMount
			}
			if tt.osStat != nil {
				oldOsStat := osStat
				defer func() {
					osStat = oldOsStat
				}()
				osStat = tt.osStat
			}
			err := tt.device.Mount()
			if (err != nil) != tt.wantErr {
				t.Errorf("Device.Mount() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr && tt.wantErrToBe != nil {
				if !errors.Is(err, tt.wantErrToBe) {
					t.Errorf("Device.Mount() error = %v, wantErrToBe %v", err, tt.wantErrToBe)
					return
				}
			}
			// test this regardless if there was an error or not
			// need to make sure that an error does not mutate the mount path unexpectedly
			if tt.device.MountPath != tt.wantMountPath {
				t.Errorf("Device.MountPath = %v, want %v", tt.device.MountPath, tt.wantMountPath)
				return
			}
		})
	}
}

func TestDevice_Unmount(t *testing.T) {
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	errUnmountFailed := errors.New("unmount failed")
	tests := []struct {
		name          string
		device        *Device
		wantErr       bool
		wantErrToBe   error
		wantMountPath string
		rootPath      string
		unixUnmount   func(target string, flags int) error
	}{
		{
			name: "success",
			device: &Device{
				Path:      "/dev/vda3",
				MountPath: "/mnt/hedgehog-identity",
				FS:        &fsOs{},
			},
			unixUnmount: func(target string, flags int) error {
				return nil
			},
			rootPath:      filepath.Join(pwd, "testdata", "IsMounted", "one"),
			wantErr:       false,
			wantMountPath: "",
		},
		{
			name: "not mounted",
			device: &Device{
				MountPath: "",
			},
			wantErr:       false,
			wantMountPath: "",
		},
		{
			name: "unmount fails",
			device: &Device{
				Path:      "/dev/vda3",
				MountPath: "/mnt/hedgehog-identity",
			},
			unixUnmount: func(target string, flags int) error {
				return errUnmountFailed
			},
			rootPath:      filepath.Join(pwd, "testdata", "IsMounted", "one"),
			wantErr:       true,
			wantErrToBe:   errUnmountFailed,
			wantMountPath: "/mnt/hedgehog-identity",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.rootPath != "" {
				oldRootPath := rootPath
				defer func() {
					rootPath = oldRootPath
				}()
				rootPath = tt.rootPath
			}
			if tt.unixUnmount != nil {
				oldUnixUnmount := unixUnmount
				defer func() {
					unixUnmount = oldUnixUnmount
				}()
				unixUnmount = tt.unixUnmount
			}
			err := tt.device.Unmount()
			if (err != nil) != tt.wantErr {
				t.Errorf("Device.Unmount() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr && tt.wantErrToBe != nil {
				if !errors.Is(err, tt.wantErrToBe) {
					t.Errorf("Device.Unmount() error = %v, wantErrToBe %v", err, tt.wantErrToBe)
					return
				}
			}
			// test this regardless if there was an error or not
			// need to make sure that an error does not mutate the mount path unexpectedly
			if tt.device.MountPath != tt.wantMountPath {
				t.Errorf("Device.MountPath = %v, want %v", tt.device.MountPath, tt.wantMountPath)
				return
			}
		})
	}
}

func Test_ensureMountPath(t *testing.T) {
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	base := filepath.Join(pwd, "testdata", "ensureMountPath")
	type args struct {
		path string
	}
	errOsStatFailed := errors.New("os.Stat failed unrecoverably")
	errOsRemoveFailed := errors.New("os.Remove failed")
	errOsMkdirAllFailed := errors.New("os.MkdirAll failed")
	tests := []struct {
		name        string
		args        args
		wantErr     bool
		wantErrToBe error
		osStat      func(name string) (fs.FileInfo, error)
		osRemove    func(name string) error
		osMkdirAll  func(path string, perm fs.FileMode) error
		pre         func()
		cleanupPath bool
	}{
		{
			name: "already exists",
			args: args{
				path: filepath.Join(base, "exists"),
			},
			pre: func() {
				if err := os.MkdirAll(filepath.Join(base, "exists"), 0750); err != nil {
					panic(err)
				}
			},
			cleanupPath: true,
			wantErr:     false,
		},
		{
			name: "already exists as a file",
			args: args{
				path: filepath.Join(base, "exists"),
			},
			pre: func() {
				if _, err := os.Create(filepath.Join(base, "exists")); err != nil {
					panic(err)
				}
			},
			cleanupPath: true,
			wantErr:     false,
		},
		{
			name: "osStat failed unrecoverably",
			args: args{
				path: filepath.Join(base, "fail"),
			},
			osStat: func(name string) (fs.FileInfo, error) {
				if name != filepath.Join(base, "fail") {
					return nil, fmt.Errorf("unexpected path: %s", name)
				}
				return nil, errOsStatFailed
			},
			wantErr:     true,
			wantErrToBe: errOsStatFailed,
		},
		{
			name: "already exists as a file but removal fails",
			args: args{
				path: filepath.Join(base, "exists"),
			},
			pre: func() {
				if _, err := os.Create(filepath.Join(base, "exists")); err != nil {
					panic(err)
				}
			},
			osRemove: func(name string) error {
				if name != filepath.Join(base, "exists") {
					return fmt.Errorf("unexpected path: %s", name)
				}
				return errOsRemoveFailed
			},
			cleanupPath: true,
			wantErr:     true,
			wantErrToBe: errOsRemoveFailed,
		},
		{
			name: "does not exist yet",
			args: args{
				path: filepath.Join(base, "createme"),
			},
			cleanupPath: true,
			wantErr:     false,
		},
		{
			name: "does not exist yet but create failed",
			args: args{
				path: filepath.Join(base, "createme"),
			},
			osMkdirAll: func(path string, perm fs.FileMode) error {
				if path != filepath.Join(base, "createme") {
					return fmt.Errorf("unexpected path: %s", path)
				}
				if perm != 0750 {
					return fmt.Errorf("unexpected permissions: %o", perm)
				}
				return errOsMkdirAllFailed
			},
			wantErr:     true,
			wantErrToBe: errOsMkdirAllFailed,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.osStat != nil {
				oldOsStat := osStat
				defer func() {
					osStat = oldOsStat
				}()
				osStat = tt.osStat
			}
			if tt.osRemove != nil {
				oldOsRemove := osRemove
				defer func() {
					osRemove = oldOsRemove
				}()
				osRemove = tt.osRemove
			}
			if tt.osMkdirAll != nil {
				oldOsMkdirAll := osMkdirAll
				defer func() {
					osMkdirAll = oldOsMkdirAll
				}()
				osMkdirAll = tt.osMkdirAll
			}
			if tt.cleanupPath {
				defer func() {
					os.Remove(tt.args.path)
				}()
			}
			if tt.pre != nil {
				tt.pre()
			}
			err := ensureMountPath(tt.args.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ensureMountPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr && tt.wantErrToBe != nil {
				if !errors.Is(err, tt.wantErrToBe) {
					t.Errorf("ensureMountPath() error = %v, wantErrToBe %v", err, tt.wantErrToBe)
					return
				}
			}
		})
	}
}

func TestDevice_MakeFilesystemForHedgehogIdentityPartition(t *testing.T) {
	errMkfsCmdFailed := errors.New("mkfs failed")
	type args struct {
		force bool
	}
	tests := []struct {
		name        string
		device      *Device
		args        args
		wantErr     bool
		wantErrToBe error
		cmds        func(t *testing.T, ctrl *gomock.Controller) []exec.CommandFunc
	}{
		{
			name: "success",
			args: args{
				force: false,
			},
			device: &Device{
				Uevent: Uevent{
					UeventDevtype: UeventDevtypePartition,
				},
				GPTPartType: GPTPartTypeHedgehogIdentity,
				Path:        "/path/to/device",
				Filesystem:  "",
			},
			cmds: func(t *testing.T, ctrl *gomock.Controller) []exec.CommandFunc {
				return []exec.CommandFunc{
					mockexec.MockCommand(t, ctrl, []string{"mkfs.ext4", "-L", FSLabelHedgehogIdentity, "-F", "/path/to/device"}, func(tc *mockexec.TestCmd) {
						tc.EXPECT().Run().Times(1).DoAndReturn(func() error {
							return tc.IsExpectedCommand()
						})
					}),
				}
			},
			wantErr: false,
		},
		{
			name: "exists already but force is set",
			args: args{
				force: true,
			},
			device: &Device{
				Uevent: Uevent{
					UeventDevtype: UeventDevtypePartition,
				},
				GPTPartType: GPTPartTypeHedgehogIdentity,
				Path:        "/path/to/device",
				Filesystem:  "ext4",
				FSLabel:     FSLabelHedgehogIdentity,
			},
			cmds: func(t *testing.T, ctrl *gomock.Controller) []exec.CommandFunc {
				return []exec.CommandFunc{
					mockexec.MockCommand(t, ctrl, []string{"mkfs.ext4", "-L", FSLabelHedgehogIdentity, "-F", "/path/to/device"}, func(tc *mockexec.TestCmd) {
						tc.EXPECT().Run().Times(1).DoAndReturn(func() error {
							return tc.IsExpectedCommand()
						})
					}),
				}
			},
			wantErr: false,
		},
		{
			name: "mkfs fails",
			args: args{
				force: false,
			},
			device: &Device{
				Uevent: Uevent{
					UeventDevtype: UeventDevtypePartition,
				},
				GPTPartType: GPTPartTypeHedgehogIdentity,
				Path:        "/path/to/device",
				Filesystem:  "",
			},
			cmds: func(t *testing.T, ctrl *gomock.Controller) []exec.CommandFunc {
				return []exec.CommandFunc{
					mockexec.MockCommand(t, ctrl, []string{"mkfs.ext4", "-L", FSLabelHedgehogIdentity, "-F", "/path/to/device"}, func(tc *mockexec.TestCmd) {
						tc.EXPECT().Run().Times(1).DoAndReturn(func() error {
							if err := tc.IsExpectedCommand(); err != nil {
								return err
							}
							return errMkfsCmdFailed
						})
					}),
				}
			},
			wantErr:     true,
			wantErrToBe: errMkfsCmdFailed,
		},
		{
			name: "exists already and no force is set",
			args: args{
				force: false,
			},
			device: &Device{
				Uevent: Uevent{
					UeventDevtype: UeventDevtypePartition,
				},
				GPTPartType: GPTPartTypeHedgehogIdentity,
				Path:        "/path/to/device",
				Filesystem:  "ext4",
				FSLabel:     FSLabelHedgehogIdentity,
			},
			wantErr:     true,
			wantErrToBe: ErrFilesystemAlreadyCreated,
		},
		{
			name: "exists already but is different filesystem",
			args: args{
				force: false,
			},
			device: &Device{
				Uevent: Uevent{
					UeventDevtype: UeventDevtypePartition,
				},
				GPTPartType: GPTPartTypeHedgehogIdentity,
				Path:        "/path/to/device",
				Filesystem:  "ext4",
				FSLabel:     "SONiC",
			},
			cmds: func(t *testing.T, ctrl *gomock.Controller) []exec.CommandFunc {
				return []exec.CommandFunc{
					mockexec.MockCommand(t, ctrl, []string{"mkfs.ext4", "-L", FSLabelHedgehogIdentity, "-F", "/path/to/device"}, func(tc *mockexec.TestCmd) {
						tc.EXPECT().Run().Times(1).DoAndReturn(func() error {
							return tc.IsExpectedCommand()
						})
					}),
				}
			},
			wantErr: false,
		},
		{
			name: "no device node",
			args: args{
				force: false,
			},
			device: &Device{
				Uevent: Uevent{
					UeventDevtype: UeventDevtypePartition,
				},
				GPTPartType: GPTPartTypeHedgehogIdentity,
				Path:        "",
				Filesystem:  "",
			},
			wantErr:     true,
			wantErrToBe: ErrNoDeviceNode,
		},
		{
			name: "not hedgehog identity partition",
			args: args{
				force: false,
			},
			device: &Device{
				Uevent: Uevent{
					UeventDevtype: UeventDevtypePartition,
				},
				GPTPartType: GPTPartTypeEFI,
			},
			wantErr:     true,
			wantErrToBe: ErrUnsupportedMkfsForDevice,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			if tt.cmds != nil {
				oldCommand := exec.Command
				defer func() {
					exec.Command = oldCommand
				}()
				cmds := mockexec.NewMockCommands(tt.cmds(t, ctrl))
				defer cmds.Finish()
				exec.Command = cmds.Command()
			}
			err := tt.device.MakeFilesystemForHedgehogIdentityPartition(tt.args.force)
			if (err != nil) != tt.wantErr {
				t.Errorf("Device.MakeFilesystemForHedgehogIdentityPartition() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr && tt.wantErrToBe != nil {
				if !errors.Is(err, tt.wantErrToBe) {
					t.Errorf("Device.MakeFilesystemForHedgehogIdentityPartition() error = %v, wantErrToBe %v", err, tt.wantErrToBe)
					return
				}
			}
		})
	}
}

func TestDevice_discoverFilesystemLabel(t *testing.T) {
	errCmdFailed := errors.New("command failed")
	tests := []struct {
		name        string
		device      *Device
		wantErr     bool
		wantErrToBe error
		wantFSLabel string
		cmds        func(t *testing.T, ctrl *gomock.Controller) []exec.CommandFunc
	}{
		{
			name: "no device node",
			device: &Device{
				Path: "",
			},
			wantErr:     true,
			wantErrToBe: ErrNoDeviceNode,
		},
		{
			name: "command fails",
			device: &Device{
				Path: "/path/to/device",
			},
			cmds: func(t *testing.T, ctrl *gomock.Controller) []exec.CommandFunc {
				return []exec.CommandFunc{
					mockexec.MockCommand(t, ctrl, []string{"grub-probe", "-d", "/path/to/device", "-t", "fs_label"}, func(tc *mockexec.TestCmd) {
						tc.EXPECT().Output().Times(1).DoAndReturn(func() ([]byte, error) {
							if err := tc.IsExpectedCommand(); err != nil {
								return nil, err
							}
							return nil, errCmdFailed
						})
					}),
				}
			},
			wantErr:     true,
			wantErrToBe: errCmdFailed,
		},
		{
			name: "success",
			device: &Device{
				Path: "/path/to/device",
			},
			cmds: func(t *testing.T, ctrl *gomock.Controller) []exec.CommandFunc {
				return []exec.CommandFunc{
					mockexec.MockCommand(t, ctrl, []string{"grub-probe", "-d", "/path/to/device", "-t", "fs_label"}, func(tc *mockexec.TestCmd) {
						tc.EXPECT().Output().Times(1).DoAndReturn(func() ([]byte, error) {
							if err := tc.IsExpectedCommand(); err != nil {
								return nil, err
							}
							return []byte("my_fancy_fs_label"), nil
						})
					}),
				}
			},
			wantErr:     false,
			wantFSLabel: "my_fancy_fs_label",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			if tt.cmds != nil {
				oldCommand := exec.Command
				defer func() {
					exec.Command = oldCommand
				}()
				cmds := mockexec.NewMockCommands(tt.cmds(t, ctrl))
				defer cmds.Finish()
				exec.Command = cmds.Command()
			}
			err := tt.device.discoverFilesystemLabel()
			if (err != nil) != tt.wantErr {
				t.Errorf("Device.discoverFilesystemLabel() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr && tt.wantErrToBe != nil {
				if !errors.Is(err, tt.wantErrToBe) {
					t.Errorf("Device.discoverFilesystemLabel() error = %v, wantErrToBe %v", err, tt.wantErrToBe)
					return
				}
			}
			// test this regardless if there was an error or not
			// need to make sure that an error does not mutate the field unexpectedly
			if tt.device.FSLabel != tt.wantFSLabel {
				t.Errorf("Device.FSLabel = %v, want %v", tt.device.FSLabel, tt.wantFSLabel)
				return
			}
		})
	}
}

func TestDevice_discoverFilesystem(t *testing.T) {
	errCmdFailed := errors.New("command failed")
	tests := []struct {
		name           string
		device         *Device
		wantErr        bool
		wantErrToBe    error
		wantFilesystem string
		cmds           func(t *testing.T, ctrl *gomock.Controller) []exec.CommandFunc
	}{
		{
			name: "no device node",
			device: &Device{
				Path: "",
			},
			wantErr:     true,
			wantErrToBe: ErrNoDeviceNode,
		},
		{
			name: "command fails",
			device: &Device{
				Path: "/path/to/device",
			},
			cmds: func(t *testing.T, ctrl *gomock.Controller) []exec.CommandFunc {
				return []exec.CommandFunc{
					mockexec.MockCommand(t, ctrl, []string{"grub-probe", "-d", "/path/to/device", "-t", "fs"}, func(tc *mockexec.TestCmd) {
						tc.EXPECT().Output().Times(1).DoAndReturn(func() ([]byte, error) {
							if err := tc.IsExpectedCommand(); err != nil {
								return nil, err
							}
							return nil, errCmdFailed
						})
					}),
				}
			},
			wantErr:     true,
			wantErrToBe: errCmdFailed,
		},
		{
			name: "success",
			device: &Device{
				Path: "/path/to/device",
			},
			cmds: func(t *testing.T, ctrl *gomock.Controller) []exec.CommandFunc {
				return []exec.CommandFunc{
					mockexec.MockCommand(t, ctrl, []string{"grub-probe", "-d", "/path/to/device", "-t", "fs"}, func(tc *mockexec.TestCmd) {
						tc.EXPECT().Output().Times(1).DoAndReturn(func() ([]byte, error) {
							if err := tc.IsExpectedCommand(); err != nil {
								return nil, err
							}
							return []byte("unittestfs"), nil
						})
					}),
				}
			},
			wantErr:        false,
			wantFilesystem: "unittestfs",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			if tt.cmds != nil {
				oldCommand := exec.Command
				defer func() {
					exec.Command = oldCommand
				}()
				cmds := mockexec.NewMockCommands(tt.cmds(t, ctrl))
				defer cmds.Finish()
				exec.Command = cmds.Command()
			}
			err := tt.device.discoverFilesystem()
			if (err != nil) != tt.wantErr {
				t.Errorf("Device.discoverFilesystem() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr && tt.wantErrToBe != nil {
				if !errors.Is(err, tt.wantErrToBe) {
					t.Errorf("Device.discoverFilesystem() error = %v, wantErrToBe %v", err, tt.wantErrToBe)
					return
				}
			}
			// test this regardless if there was an error or not
			// need to make sure that an error does not mutate the field unexpectedly
			if tt.device.Filesystem != tt.wantFilesystem {
				t.Errorf("Device.Filesystem = %v, want %v", tt.device.Filesystem, tt.wantFilesystem)
				return
			}
		})
	}
}

func TestDevice_discoverPartitionType(t *testing.T) {
	errCmdFailed := errors.New("command failed")
	tests := []struct {
		name            string
		device          *Device
		wantErr         bool
		wantErrToBe     error
		wantGPTPartType string
		cmds            func(t *testing.T, ctrl *gomock.Controller) []exec.CommandFunc
	}{
		{
			name: "no device node",
			device: &Device{
				Path: "",
			},
			wantErr:     true,
			wantErrToBe: ErrNoDeviceNode,
		},
		{
			name: "command fails",
			device: &Device{
				Path: "/path/to/device",
			},
			cmds: func(t *testing.T, ctrl *gomock.Controller) []exec.CommandFunc {
				return []exec.CommandFunc{
					mockexec.MockCommand(t, ctrl, []string{"grub-probe", "-d", "/path/to/device", "-t", "gpt_parttype"}, func(tc *mockexec.TestCmd) {
						tc.EXPECT().Output().Times(1).DoAndReturn(func() ([]byte, error) {
							if err := tc.IsExpectedCommand(); err != nil {
								return nil, err
							}
							return nil, errCmdFailed
						})
					}),
				}
			},
			wantErr:     true,
			wantErrToBe: errCmdFailed,
		},
		{
			name: "success",
			device: &Device{
				Path: "/path/to/device",
			},
			cmds: func(t *testing.T, ctrl *gomock.Controller) []exec.CommandFunc {
				return []exec.CommandFunc{
					mockexec.MockCommand(t, ctrl, []string{"grub-probe", "-d", "/path/to/device", "-t", "gpt_parttype"}, func(tc *mockexec.TestCmd) {
						tc.EXPECT().Output().Times(1).DoAndReturn(func() ([]byte, error) {
							if err := tc.IsExpectedCommand(); err != nil {
								return nil, err
							}
							return []byte("very-unique-gpt-partition-type"), nil
						})
					}),
				}
			},
			wantErr:         false,
			wantGPTPartType: "very-unique-gpt-partition-type",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			if tt.cmds != nil {
				oldCommand := exec.Command
				defer func() {
					exec.Command = oldCommand
				}()
				cmds := mockexec.NewMockCommands(tt.cmds(t, ctrl))
				defer cmds.Finish()
				exec.Command = cmds.Command()
			}
			err := tt.device.discoverPartitionType()
			if (err != nil) != tt.wantErr {
				t.Errorf("Device.discoverPartitionType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr && tt.wantErrToBe != nil {
				if !errors.Is(err, tt.wantErrToBe) {
					t.Errorf("Device.discoverPartitionType() error = %v, wantErrToBe %v", err, tt.wantErrToBe)
					return
				}
			}
			// test this regardless if there was an error or not
			// need to make sure that an error does not mutate the field unexpectedly
			if tt.device.GPTPartType != tt.wantGPTPartType {
				t.Errorf("Device.GPTPartType = %v, want %v", tt.device.GPTPartType, tt.wantGPTPartType)
				return
			}
		})
	}
}

func TestDevice_ensureDevicePath(t *testing.T) {
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	oldRootPath := rootPath
	rootPath = filepath.Join(pwd, "testdata", "ensureDevicePath")
	defer func() {
		rootPath = oldRootPath
	}()

	// these must be created out of band as it requires root privileges to do so
	// we'll skip the test if they don't exist
	loop0Dev := filepath.Join(pwd, "testdata", "ensureDevicePath", "dev", "loop0")
	if _, err := os.Stat(loop0Dev); err != nil {
		t.Skipf("SKIPPING: testdata must be initialized: loop0 missing: run 'sudo mknod %s b 7 0'", loop0Dev)
	}

	errOsStatFailed := errors.New("os.Stat failed")
	errOsStatFailed2 := errors.New("os.Stat failed second time")
	errOsRemoveFailed := errors.New("os.Remove failed")
	errUnixMknodFailed := errors.New("unix.Mknod failed")
	tests := []struct {
		name        string
		device      *Device
		wantErr     bool
		wantErrToBe error
		wantPath    string
		osStat      func() func(name string) (fs.FileInfo, error)
		osRemove    func(name string) error
		unixMknod   func(path string, mode uint32, dev int) (err error)
	}{
		{
			name: "already exists",
			device: &Device{
				Uevent: Uevent{
					UeventDevname: "loop0",
				},
			},
			wantErr:  false,
			wantPath: loop0Dev,
		},
		{
			name: "invalid uevent",
			device: &Device{
				Uevent: Uevent{},
			},
			wantErr:     true,
			wantErrToBe: ErrInvalidUevent,
		},
		{
			name: "invalid major minor",
			device: &Device{
				Uevent: Uevent{
					UeventDevname: "notexist",
					UeventMajor:   "not-a-number",
					UeventMinor:   "not-a-number",
				},
			},
			osStat: func() func(name string) (fs.FileInfo, error) {
				return func(name string) (fs.FileInfo, error) {
					return nil, fmt.Errorf("just fail this")
				}
			},
			wantErr:     true,
			wantErrToBe: strconv.ErrSyntax,
		},
		{
			name: "os stat fails unexpectedly",
			device: &Device{
				Uevent: Uevent{
					UeventDevname: "notexist",
					UeventMajor:   "6",
					UeventMinor:   "0",
				},
			},
			osStat: func() func(name string) (fs.FileInfo, error) {
				secondCall := false
				return func(name string) (fs.FileInfo, error) {
					if name != filepath.Join(pwd, "testdata", "ensureDevicePath", "dev", "notexist") {
						return nil, fmt.Errorf("unexpected path: %s", name)
					}
					if !secondCall {
						secondCall = true
						return nil, fmt.Errorf("just fail this")
					}
					return nil, errOsStatFailed
				}
			},
			wantErr:     true,
			wantErrToBe: errOsStatFailed,
		},
		{
			name: "remove of invalid device node fails",
			device: &Device{
				Uevent: Uevent{
					UeventDevname: "exists",
					UeventMajor:   "6",
					UeventMinor:   "0",
				},
			},
			osRemove: func(name string) error {
				if name != filepath.Join(pwd, "testdata", "ensureDevicePath", "dev", "exists") {
					return fmt.Errorf("unexpected path: %s", name)
				}
				return errOsRemoveFailed
			},
			wantErr:     true,
			wantErrToBe: errOsRemoveFailed,
		},
		{
			name: "mknod fails",
			device: &Device{
				Uevent: Uevent{
					UeventDevname: "exists",
					UeventMajor:   "6",
					UeventMinor:   "0",
				},
			},
			osStat: func() func(name string) (fs.FileInfo, error) {
				secondCall := false
				return func(name string) (fs.FileInfo, error) {
					if name != filepath.Join(pwd, "testdata", "ensureDevicePath", "dev", "exists") {
						return nil, fmt.Errorf("unexpected path: %s", name)
					}
					if !secondCall {
						secondCall = true
						return nil, fmt.Errorf("just fail this")
					}
					return os.Stat(name)
				}
			},
			osRemove: func(name string) error {
				if name != filepath.Join(pwd, "testdata", "ensureDevicePath", "dev", "exists") {
					return fmt.Errorf("unexpected path: %s", name)
				}
				// don't remove this, we just fake this
				return nil
			},
			unixMknod: func(path string, mode uint32, dev int) (err error) {
				if path != filepath.Join(pwd, "testdata", "ensureDevicePath", "dev", "exists") {
					return fmt.Errorf("unexpected path: %s", path)
				}
				if mode != unix.S_IFBLK {
					return fmt.Errorf("unexpected mode: 0x%x", mode)
				}
				if dev != int(unix.Mkdev(6, 0)) {
					return fmt.Errorf("unexpected dev: major %d, minor %d", unix.Major(uint64(dev)), unix.Minor(uint64(dev)))
				}
				return errUnixMknodFailed
			},
			wantErr:     true,
			wantErrToBe: errUnixMknodFailed,
		},
		{
			name: "mknod succeeds",
			device: &Device{
				Uevent: Uevent{
					UeventDevname: "loop0",
					UeventMajor:   "6",
					UeventMinor:   "0",
				},
			},
			osStat: func() func(name string) (fs.FileInfo, error) {
				secondCall := false
				return func(name string) (fs.FileInfo, error) {
					if name != loop0Dev {
						return nil, fmt.Errorf("unexpected path: %s", name)
					}
					if !secondCall {
						secondCall = true
						return nil, fmt.Errorf("just fail this")
					}
					return os.Stat(name)
				}
			},
			osRemove: func(name string) error {
				if name != loop0Dev {
					return fmt.Errorf("unexpected path: %s", name)
				}
				// don't remove this, we just fake this
				return nil
			},
			unixMknod: func(path string, mode uint32, dev int) (err error) {
				if path != loop0Dev {
					return fmt.Errorf("unexpected path: %s", path)
				}
				if mode != unix.S_IFBLK {
					return fmt.Errorf("unexpected mode: 0x%x", mode)
				}
				if dev != int(unix.Mkdev(6, 0)) {
					return fmt.Errorf("unexpected dev: major %d, minor %d", unix.Major(uint64(dev)), unix.Minor(uint64(dev)))
				}
				return nil
			},
			wantErr:  false,
			wantPath: loop0Dev,
		},
		{
			name: "mknod succeeds but device path still fails",
			device: &Device{
				Uevent: Uevent{
					UeventDevname: "loop0",
					UeventMajor:   "6",
					UeventMinor:   "0",
				},
			},
			osStat: func() func(name string) (fs.FileInfo, error) {
				calls := 0
				return func(name string) (fs.FileInfo, error) {
					if name != loop0Dev {
						return nil, fmt.Errorf("unexpected path: %s", name)
					}
					defer func() {
						calls += 1
					}()
					switch calls {
					case 1:
						return os.Stat(name)
					case 2:
						return nil, errOsStatFailed2
					default:
						return nil, fmt.Errorf("just fail this")
					}
				}
			},
			osRemove: func(name string) error {
				if name != loop0Dev {
					return fmt.Errorf("unexpected path: %s", name)
				}
				// don't remove this, we just fake this
				return nil
			},
			unixMknod: func(path string, mode uint32, dev int) error {
				if path != loop0Dev {
					return fmt.Errorf("unexpected path: %s", path)
				}
				if mode != unix.S_IFBLK {
					return fmt.Errorf("unexpected mode: 0x%x", mode)
				}
				if dev != int(unix.Mkdev(6, 0)) {
					return fmt.Errorf("unexpected dev: major %d, minor %d", unix.Major(uint64(dev)), unix.Minor(uint64(dev)))
				}
				return nil
			},
			wantErr:     true,
			wantErrToBe: errOsStatFailed2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.osStat != nil {
				oldOsStat := osStat
				defer func() {
					osStat = oldOsStat
				}()
				osStat = tt.osStat()
			}
			if tt.osRemove != nil {
				oldOsRemove := osRemove
				defer func() {
					osRemove = oldOsRemove
				}()
				osRemove = tt.osRemove
			}
			if tt.unixMknod != nil {
				oldUnixMknod := unixMknod
				defer func() {
					unixMknod = oldUnixMknod
				}()
				unixMknod = tt.unixMknod
			}
			err := tt.device.ensureDevicePath()
			if (err != nil) != tt.wantErr {
				t.Errorf("Device.ensureDevicePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr && tt.wantErrToBe != nil {
				if !errors.Is(err, tt.wantErrToBe) {
					t.Errorf("Device.ensureDevicePath() error = %v, wantErrToBe %v", err, tt.wantErrToBe)
					return
				}
			}
			// test this regardless if there was an error or not
			// need to make sure that an error does not mutate the field unexpectedly
			if tt.device.Path != tt.wantPath {
				t.Errorf("Device.Path = %v, want %v", tt.device.Path, tt.wantPath)
				return
			}
		})
	}
}

func Test_unescapeMountPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "success",
			path: `/mnt/with/space/\040before`,
			want: "/mnt/with/space/ before",
		},
		{
			name: "success",
			path: `/mnt/with/space/\040before/and\134/backslash`,
			want: `/mnt/with/space/ before/and\/backslash`,
		},
		{
			name: "nothing to escape",
			path: "/mnt/hedgehog-identity",
			want: "/mnt/hedgehog-identity",
		},
		{
			name: "escaped octal string not long enough",
			path: `\4`,
			want: `\4`,
		},
		{
			name: "does not begin with octal number",
			path: `\940blah`,
			want: `\940blah`,
		},
		{
			name: "does not continue with octal number",
			path: `\090blah`,
			want: `\090blah`,
		},
		{
			name: "does not end with octal number",
			path: `\049blah`,
			want: `\049blah`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := unescapeMountPath(tt.path); got != tt.want {
				t.Errorf("unescapeMountPath() = %v, want %v", got, tt.want)
			}
		})
	}
}
