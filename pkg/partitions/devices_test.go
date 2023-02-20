package partitions

import (
	"errors"
	"reflect"
	"testing"

	gomock "github.com/golang/mock/gomock"
)

func TestDevices_GetEFIPartition(t *testing.T) {
	tests := []struct {
		name string
		d    Devices
		want *Device
	}{
		{
			name: "success",
			d: Devices{
				{
					Uevent: Uevent{
						UeventDevtype: UeventDevtypePartition,
					},
					GPTPartType: GPTPartTypeONIE,
				},
				{
					Uevent: Uevent{
						UeventDevtype: UeventDevtypePartition,
					},
					GPTPartType: GPTPartTypeEFI,
				},
			},
			want: &Device{
				Uevent: Uevent{
					UeventDevtype: UeventDevtypePartition,
				},
				GPTPartType: GPTPartTypeEFI,
			},
		},
		{
			name: "failure",
			d: Devices{
				{
					Uevent: Uevent{
						UeventDevtype: UeventDevtypePartition,
					},
					GPTPartType: GPTPartTypeONIE,
				},
				{
					Uevent: Uevent{
						UeventDevtype: UeventDevtypePartition,
					},
					GPTPartType: GPTPartTypeHedgehogIdentity,
				},
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.d.GetEFIPartition(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Devices.GetEFIPartition() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDevices_GetONIEPartition(t *testing.T) {
	tests := []struct {
		name string
		d    Devices
		want *Device
	}{
		{
			name: "success",
			d: Devices{
				{
					Uevent: Uevent{
						UeventDevtype: UeventDevtypePartition,
					},
					GPTPartType: GPTPartTypeONIE,
				},
				{
					Uevent: Uevent{
						UeventDevtype: UeventDevtypePartition,
					},
					GPTPartType: GPTPartTypeEFI,
				},
			},
			want: &Device{
				Uevent: Uevent{
					UeventDevtype: UeventDevtypePartition,
				},
				GPTPartType: GPTPartTypeONIE,
			},
		},
		{
			name: "failure",
			d: Devices{
				{
					Uevent: Uevent{
						UeventDevtype: UeventDevtypePartition,
					},
					GPTPartType: GPTPartTypeEFI,
				},
				{
					Uevent: Uevent{
						UeventDevtype: UeventDevtypePartition,
					},
					GPTPartType: GPTPartTypeHedgehogIdentity,
				},
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.d.GetONIEPartition(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Devices.GetONIEPartition() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDevices_GetDiagPartition(t *testing.T) {
	tests := []struct {
		name string
		d    Devices
		want *Device
	}{
		{
			name: "success",
			d: Devices{
				{
					Uevent: Uevent{
						UeventDevtype: UeventDevtypePartition,
					},
					GPTPartType: GPTPartTypeEFI,
				},
				{
					Uevent: Uevent{
						UeventDevtype:  UeventDevtypePartition,
						UeventPartname: "HH-DIAG",
					},
				},
			},
			want: &Device{
				Uevent: Uevent{
					UeventDevtype:  UeventDevtypePartition,
					UeventPartname: "HH-DIAG",
				},
			},
		},
		{
			name: "failure",
			d: Devices{
				{
					Uevent: Uevent{
						UeventDevtype: UeventDevtypePartition,
					},
					GPTPartType: GPTPartTypeONIE,
				},
				{
					Uevent: Uevent{
						UeventDevtype: UeventDevtypePartition,
					},
					GPTPartType: GPTPartTypeHedgehogIdentity,
				},
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.d.GetDiagPartition(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Devices.GetDiagPartition() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDevices_GetHedgehogIdentityPartition(t *testing.T) {
	tests := []struct {
		name string
		d    Devices
		want *Device
	}{
		{
			name: "success",
			d: Devices{
				{
					Uevent: Uevent{
						UeventDevtype: UeventDevtypePartition,
					},
					GPTPartType: GPTPartTypeEFI,
				},
				{
					Uevent: Uevent{
						UeventDevtype: UeventDevtypePartition,
					},
					GPTPartType: GPTPartTypeHedgehogIdentity,
				},
			},
			want: &Device{
				Uevent: Uevent{
					UeventDevtype: UeventDevtypePartition,
				},
				GPTPartType: GPTPartTypeHedgehogIdentity,
			},
		},
		{
			name: "failure",
			d: Devices{
				{
					Uevent: Uevent{
						UeventDevtype: UeventDevtypePartition,
					},
					GPTPartType: GPTPartTypeONIE,
				},
				{
					Uevent: Uevent{
						UeventDevtype: UeventDevtypePartition,
					},
					GPTPartType: GPTPartTypeHedgehogLocation,
				},
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.d.GetHedgehogIdentityPartition(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Devices.GetHedgehogIdentityPartition() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDevices_GetHedgehogLocationPartition(t *testing.T) {
	tests := []struct {
		name string
		d    Devices
		want *Device
	}{
		{
			name: "success",
			d: Devices{
				{
					Uevent: Uevent{
						UeventDevtype: UeventDevtypePartition,
					},
					GPTPartType: GPTPartTypeEFI,
				},
				{
					Uevent: Uevent{
						UeventDevtype: UeventDevtypePartition,
					},
					GPTPartType: GPTPartTypeHedgehogLocation,
				},
			},
			want: &Device{
				Uevent: Uevent{
					UeventDevtype: UeventDevtypePartition,
				},
				GPTPartType: GPTPartTypeHedgehogLocation,
			},
		},
		{
			name: "failure",
			d: Devices{
				{
					Uevent: Uevent{
						UeventDevtype: UeventDevtypePartition,
					},
					GPTPartType: GPTPartTypeEFI,
				},
				{
					Uevent: Uevent{
						UeventDevtype: UeventDevtypePartition,
					},
					GPTPartType: GPTPartTypeONIE,
				},
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.d.GetHedgehogLocationPartition(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Devices.GetHedgehogLocationPartition() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDevices_DeletePartitions(t *testing.T) {
	// some error fixtures
	errDeleteFailed := errors.New("sgdisk -d failed")

	// create a set of realistic GOOD test data
	disk := &Device{
		Uevent: Uevent{
			UeventDevtype: UeventDevtypeDisk,
		},
		Path: "/path/to/disk/device",
	}
	partEFI := &Device{
		Uevent: Uevent{
			UeventDevtype: UeventDevtypePartition,
			UeventPartn:   "1",
		},
		GPTPartType: GPTPartTypeEFI,
	}
	partONIE := &Device{
		Uevent: Uevent{
			UeventDevtype: UeventDevtypePartition,
			UeventPartn:   "2",
		},
		GPTPartType: GPTPartTypeONIE,
	}
	partDiag := &Device{
		Uevent: Uevent{
			UeventDevtype:  UeventDevtypePartition,
			UeventPartn:    "3",
			UeventPartname: "HH-DIAG",
		},
	}
	partHHIdentity := &Device{
		Uevent: Uevent{
			UeventDevtype: UeventDevtypePartition,
			UeventPartn:   "4",
		},
		GPTPartType: GPTPartTypeHedgehogIdentity,
	}
	partNOS := &Device{
		Uevent: Uevent{
			UeventDevtype: UeventDevtypePartition,
			UeventPartn:   "5",
		},
	}
	disk.Partitions = []*Device{partEFI, partONIE, partDiag, partHHIdentity, partNOS}
	partEFI.Disk = disk
	partONIE.Disk = disk
	partDiag.Disk = disk
	partHHIdentity.Disk = disk
	partNOS.Disk = disk

	// create some broken objects
	partONIENoDisk := &Device{
		Uevent: Uevent{
			UeventDevtype: UeventDevtypePartition,
			UeventPartn:   "2",
		},
		GPTPartType: GPTPartTypeONIE,
	}
	diskNoPartitions := &Device{
		Uevent: Uevent{
			UeventDevtype: UeventDevtypeDisk,
		},
		Path: "/path/to/disk/device",
	}
	partONIEBrokenDisk := &Device{
		Uevent: Uevent{
			UeventDevtype: UeventDevtypePartition,
			UeventPartn:   "2",
		},
		GPTPartType: GPTPartTypeONIE,
	}
	partONIEBrokenDisk.Disk = diskNoPartitions

	type args struct {
		platform string
	}
	tests := []struct {
		name        string
		d           Devices
		args        args
		wantErr     bool
		wantErrToBe error
		execCommand func(t *testing.T, ctrl *gomock.Controller) func(name string, arg ...string) Cmd
	}{
		{
			name: "success",
			d: Devices{
				partEFI,
				partONIE,
				partDiag,
				partHHIdentity,
				partNOS,
			},
			execCommand: func(t *testing.T, ctrl *gomock.Controller) func(name string, arg ...string) Cmd {
				return func(name string, arg ...string) Cmd {
					cmd := NewMockCmd(ctrl)
					testCmd := &testCmd{
						Cmd:             cmd,
						name:            name,
						arg:             arg,
						expectedNameArg: []string{"sgdisk", "-d", "5", "/path/to/disk/device"},
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
			name: "delete failed",
			d: Devices{
				partEFI,
				partONIE,
				partDiag,
				partHHIdentity,
				partNOS,
			},
			execCommand: func(t *testing.T, ctrl *gomock.Controller) func(name string, arg ...string) Cmd {
				return func(name string, arg ...string) Cmd {
					cmd := NewMockCmd(ctrl)
					testCmd := &testCmd{
						Cmd:             cmd,
						name:            name,
						arg:             arg,
						expectedNameArg: []string{"sgdisk", "-d", "5", "/path/to/disk/device"},
					}
					cmd.EXPECT().Run().Times(1).DoAndReturn(func() error {
						if err := testCmd.IsExpectedCommand(); err != nil {
							return err
						}
						return errDeleteFailed
					})
					return testCmd
				}
			},
			wantErr:     true,
			wantErrToBe: errDeleteFailed,
		},
		{
			name: "ONIE partition missing",
			d: Devices{
				partEFI,
				partDiag,
				partHHIdentity,
				partNOS,
			},
			wantErr:     true,
			wantErrToBe: ErrONIEPartitionNotFound,
		},
		{
			name: "broken discovery ONIE missing disk",
			d: Devices{
				partEFI,
				partONIENoDisk,
				partDiag,
				partHHIdentity,
				partNOS,
			},
			wantErr:     true,
			wantErrToBe: ErrBrokenDiscovery,
		},
		{
			name: "broken discovery disk with no partitions",
			d: Devices{
				partEFI,
				partONIEBrokenDisk,
				partDiag,
				partHHIdentity,
				partNOS,
			},
			wantErr:     true,
			wantErrToBe: ErrBrokenDiscovery,
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
			err := tt.d.DeletePartitions(tt.args.platform)
			if (err != nil) != tt.wantErr {
				t.Errorf("Devices.DeletePartitions() error = %v, wantErr %v", err, tt.wantErr)
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
