package partitions

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

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
		execCommand func(t *testing.T, ctrl *gomock.Controller) func(name string, arg ...string) Cmd
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
			execCommand: func(t *testing.T, ctrl *gomock.Controller) func(name string, arg ...string) Cmd {
				return func(name string, arg ...string) Cmd {
					cmd := NewMockCmd(ctrl)
					testCmd := &testCmd{
						Cmd:             cmd,
						name:            name,
						arg:             arg,
						expectedNameArg: []string{"sgdisk", "-d", "6", "/path/to/disk/device"},
					}
					cmd.EXPECT().Run().Times(1).DoAndReturn(func() error {
						return testCmd.IsExpectedCommand()
					})
					return testCmd
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
			execCommand: func(t *testing.T, ctrl *gomock.Controller) func(name string, arg ...string) Cmd {
				return func(name string, arg ...string) Cmd {
					cmd := NewMockCmd(ctrl)
					cmd.EXPECT().Run().Times(1).DoAndReturn(func() error {
						return sgdiskCommandFailed
					})
					return cmd
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
			if tt.execCommand != nil {
				oldExecCommand := execCommand
				defer func() {
					execCommand = oldExecCommand
				}()
				execCommand = tt.execCommand(t, ctrl)
			}
			err := tt.device.Delete()
			if (err != nil) != tt.wantErr {
				t.Errorf("Device.Delete() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.wantErr && tt.wantErrToBe != nil {
				if !errors.Is(err, tt.wantErrToBe) {
					t.Errorf("Uevent.DevicePath() error = %v, wantErrToBe %v", err, tt.wantErrToBe)
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
		name            string
		device          *Device
		unixIoctlGetInt func(fd int, req uint) (int, error)
		wantErr         bool
		wantErrToBe     error
	}{
		{
			name: "success",
			device: &Device{
				Uevent: Uevent{
					UeventDevtype: UeventDevtypeDisk,
				},
				Path: filepath.Join(pwd, "testdata", "ReReadPartitionTable", "device"),
			},
			unixIoctlGetInt: func(fd int, req uint) (int, error) {
				if fd <= 2 {
					return 0, fmt.Errorf("not an opened device file")
				}
				if req != blkrrpart {
					return 0, fmt.Errorf("not a BLKRRPART ioctl")
				}
				return 42, nil
			},
			wantErr: false,
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
		{
			name: "device node missing",
			device: &Device{
				Uevent: Uevent{
					UeventDevtype: UeventDevtypeDisk,
				},
				Path: filepath.Join(pwd, "testdata", "ReReadPartitionTable", "missing"),
			},
			wantErr:     true,
			wantErrToBe: os.ErrNotExist,
		},
		{
			name: "ioctl fails",
			device: &Device{
				Uevent: Uevent{
					UeventDevtype: UeventDevtypeDisk,
				},
				Path: filepath.Join(pwd, "testdata", "ReReadPartitionTable", "device"),
			},
			unixIoctlGetInt: func(fd int, req uint) (int, error) {
				return 0, errIoctlFailed
			},
			wantErr:     true,
			wantErrToBe: errIoctlFailed,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.unixIoctlGetInt != nil {
				oldUnixIoctlGetInt := unixIoctlGetInt
				defer func() {
					unixIoctlGetInt = oldUnixIoctlGetInt
				}()
				unixIoctlGetInt = tt.unixIoctlGetInt
			}
			err := tt.device.ReReadPartitionTable()
			if (err != nil) != tt.wantErr {
				t.Errorf("Device.ReReadPartitionTable() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.wantErr && tt.wantErrToBe != nil {
				if !errors.Is(err, tt.wantErrToBe) {
					t.Errorf("Uevent.DevicePath() error = %v, wantErrToBe %v", err, tt.wantErrToBe)
				}
			}
		})
	}
}

func TestDevice_IsMounted(t *testing.T) {
	tests := []struct {
		name   string
		device *Device
		want   bool
	}{
		{
			name: "is mounted",
			device: &Device{
				MountPath: "/mount/path",
			},
			want: true,
		},
		{
			name: "is not mounted",
			device: &Device{
				MountPath: "",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.device.IsMounted(); got != tt.want {
				t.Errorf("Device.IsMounted() = %v, want %v", got, tt.want)
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
				Path:      "/path/to/device",
				MountPath: "/path/to/mount/path",
			},
			wantErr:     true,
			wantErrToBe: ErrAlreadyMounted,
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
			if err == nil && tt.device.MountPath != tt.wantMountPath {
				t.Errorf("Device.MountPath = %v, want %v", tt.device.MountPath, tt.wantMountPath)
				return
			}
		})
	}
}
