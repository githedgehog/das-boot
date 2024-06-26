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
	"reflect"
	"testing"

	"go.githedgehog.com/dasboot/pkg/exec"
	"go.githedgehog.com/dasboot/test/mock/mockexec"
	"go.githedgehog.com/dasboot/test/mock/mockuefi"

	efiguid "github.com/0x5a17ed/uefi/efi/efiguid"
	"github.com/0x5a17ed/uefi/efi/efivario"
	"github.com/0x5a17ed/uefi/efi/efivars"
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
	errMakeONIEDefaultFailed := errors.New("MakeONIEDefaultAndCleanup() failed")

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

	// good set for multiple deletes
	partEFI2 := &Device{
		Uevent: Uevent{
			UeventDevtype: UeventDevtypePartition,
			UeventPartn:   "1",
		},
		GPTPartType: GPTPartTypeEFI,
	}
	partONIE2 := &Device{
		Uevent: Uevent{
			UeventDevtype: UeventDevtypePartition,
			UeventPartn:   "2",
		},
		GPTPartType: GPTPartTypeONIE,
	}
	partNOS21 := &Device{
		Uevent: Uevent{
			UeventDevtype: UeventDevtypePartition,
			UeventPartn:   "3",
		},
	}
	partNOS22 := &Device{
		Uevent: Uevent{
			UeventDevtype: UeventDevtypePartition,
			UeventPartn:   "4",
		},
	}
	partNOS23 := &Device{
		Uevent: Uevent{
			UeventDevtype: UeventDevtypePartition,
			UeventPartn:   "5",
		},
	}
	disk2 := &Device{
		Uevent: Uevent{
			UeventDevtype: UeventDevtypeDisk,
		},
		Path: "/path/to/disk/device",
	}
	disk2.Partitions = []*Device{partEFI2, partONIE2, partNOS22, partNOS23, partNOS21}
	partEFI2.Disk = disk2
	partONIE2.Disk = disk2
	partNOS21.Disk = disk2
	partNOS22.Disk = disk2
	partNOS23.Disk = disk2

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
		name                                         string
		d                                            Devices
		args                                         args
		callsMakeONIEDefaultBootEntryAndCleanup      bool
		callsMakeONIEDefaultBootEntryAndCleanupFails bool
		wantErr                                      bool
		wantErrToBe                                  error
		cmds                                         func(t *testing.T, ctrl *gomock.Controller) []exec.CommandFunc
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
			cmds: func(t *testing.T, ctrl *gomock.Controller) []exec.CommandFunc {
				return []exec.CommandFunc{
					mockexec.MockCommand(t, ctrl, []string{"sgdisk", "-d", "5", "/path/to/disk/device"}, func(tc *mockexec.TestCmd) {
						tc.EXPECT().Run().Times(1).DoAndReturn(func() error {
							return tc.IsExpectedCommand()
						})
					}),
					mockexec.MockCommand(t, ctrl, []string{"partprobe", "/path/to/disk/device"}, func(tc *mockexec.TestCmd) {
						tc.EXPECT().Run().Times(1).DoAndReturn(func() error {
							return tc.IsExpectedCommand()
						})
					}),
				}
			},
			callsMakeONIEDefaultBootEntryAndCleanup: true,
			wantErr:                                 false,
		},
		{
			name: "success but rereading partition table failed",
			d: Devices{
				partEFI,
				partONIE,
				partDiag,
				partHHIdentity,
				partNOS,
			},
			cmds: func(t *testing.T, ctrl *gomock.Controller) []exec.CommandFunc {
				return []exec.CommandFunc{
					mockexec.MockCommand(t, ctrl, []string{"sgdisk", "-d", "5", "/path/to/disk/device"}, func(tc *mockexec.TestCmd) {
						tc.EXPECT().Run().Times(1).DoAndReturn(func() error {
							return tc.IsExpectedCommand()
						})
					}),
					mockexec.MockCommand(t, ctrl, []string{"partprobe", "/path/to/disk/device"}, func(tc *mockexec.TestCmd) {
						tc.EXPECT().Run().Times(1).DoAndReturn(func() error {
							if err := tc.IsExpectedCommand(); err != nil {
								panic(err)
							}
							return fmt.Errorf("rereading partition table failed which should be ignored")
						})
					}),
				}
			},
			callsMakeONIEDefaultBootEntryAndCleanup: true,
			wantErr:                                 false,
		},
		{
			name: "success exercise sort and multiple delete",
			d: Devices{
				partEFI2,
				partONIE2,
				partNOS21,
				partNOS22,
				partNOS23,
			},
			cmds: func(t *testing.T, ctrl *gomock.Controller) []exec.CommandFunc {
				return []exec.CommandFunc{
					mockexec.MockCommand(t, ctrl, []string{"sgdisk", "-d", "5", "/path/to/disk/device"}, func(tc *mockexec.TestCmd) {
						tc.EXPECT().Run().Times(1).DoAndReturn(func() error {
							return tc.IsExpectedCommand()
						})
					}),
					mockexec.MockCommand(t, ctrl, []string{"sgdisk", "-d", "4", "/path/to/disk/device"}, func(tc *mockexec.TestCmd) {
						tc.EXPECT().Run().Times(1).DoAndReturn(func() error {
							return tc.IsExpectedCommand()
						})
					}),
					mockexec.MockCommand(t, ctrl, []string{"sgdisk", "-d", "3", "/path/to/disk/device"}, func(tc *mockexec.TestCmd) {
						tc.EXPECT().Run().Times(1).DoAndReturn(func() error {
							return tc.IsExpectedCommand()
						})
					}),
					mockexec.MockCommand(t, ctrl, []string{"partprobe", "/path/to/disk/device"}, func(tc *mockexec.TestCmd) {
						tc.EXPECT().Run().Times(1).DoAndReturn(func() error {
							return tc.IsExpectedCommand()
						})
					}),
				}
			},
			callsMakeONIEDefaultBootEntryAndCleanup: true,
			wantErr:                                 false,
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
			cmds: func(t *testing.T, ctrl *gomock.Controller) []exec.CommandFunc {
				return []exec.CommandFunc{
					mockexec.MockCommand(t, ctrl, []string{"sgdisk", "-d", "5", "/path/to/disk/device"}, func(tc *mockexec.TestCmd) {
						tc.EXPECT().Run().Times(1).DoAndReturn(func() error {
							if err := tc.IsExpectedCommand(); err != nil {
								return err
							}
							return errDeleteFailed
						})
					}),
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
		{
			name: "calling MakeONIEDefaultBootEntryAndCleanup fails",
			d: Devices{
				partEFI,
				partONIE,
				partDiag,
				partHHIdentity,
				partNOS,
			},
			cmds: func(t *testing.T, ctrl *gomock.Controller) []exec.CommandFunc {
				return []exec.CommandFunc{
					mockexec.MockCommand(t, ctrl, []string{"sgdisk", "-d", "5", "/path/to/disk/device"}, func(tc *mockexec.TestCmd) {
						tc.EXPECT().Run().Times(1).DoAndReturn(func() error {
							return tc.IsExpectedCommand()
						})
					}),
					mockexec.MockCommand(t, ctrl, []string{"partprobe", "/path/to/disk/device"}, func(tc *mockexec.TestCmd) {
						tc.EXPECT().Run().Times(1).DoAndReturn(func() error {
							return tc.IsExpectedCommand()
						})
					}),
				}
			},
			callsMakeONIEDefaultBootEntryAndCleanup:      true,
			callsMakeONIEDefaultBootEntryAndCleanupFails: true,
			wantErr:     true,
			wantErrToBe: errMakeONIEDefaultFailed,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			//// START - mock for MakeONIEDefaultBootEntryAndCleanup() call
			c := mockuefi.NewMockContext(ctrl)
			oldEfiCtx := efiCtx
			defer func() {
				efiCtx = oldEfiCtx
			}()
			efiCtx = c
			if tt.callsMakeONIEDefaultBootEntryAndCleanup {
				t.Skipf("Skipping %s until mockgen properyly supports generics for VariableNameIterator...", tt.name)
				if tt.callsMakeONIEDefaultBootEntryAndCleanupFails {
					c.EXPECT().GetSizeHint(gomock.Eq("BootCurrent"), gomock.Eq(efivars.GlobalVariable)).Times(1).
						Return(int64(0), nil)
					c.EXPECT().Get(gomock.Eq("BootCurrent"), gomock.Eq(efivars.GlobalVariable), gomock.Eq([]byte{})).Times(1).
						Return(efivario.Attributes(0), 0, errMakeONIEDefaultFailed)
				} else {
					c.EXPECT().GetSizeHint(gomock.Eq("BootCurrent"), gomock.Eq(efivars.GlobalVariable)).Times(1).
						Return(int64(2), nil)
					c.EXPECT().Get(gomock.Eq("BootCurrent"), gomock.Eq(efivars.GlobalVariable), gomock.Eq([]byte{0, 0})).Times(1).
						DoAndReturn(func(name string, guid efiguid.GUID, out []byte) (efivario.Attributes, int, error) {
							copy(out, []byte{0x07, 0x00})
							return efivario.BootServiceAccess | efivario.RuntimeAccess, 2, nil
						})
					c.EXPECT().GetSizeHint(gomock.Eq("Boot0007"), gomock.Eq(efivars.GlobalVariable)).Times(1).
						Return(int64(len(onieBootContents)-4), nil)
					c.EXPECT().Get(gomock.Eq("Boot0007"), gomock.Eq(efivars.GlobalVariable), gomock.Eq(make([]byte, len(onieBootContents)-4))).Times(1).
						DoAndReturn(func(name string, guid efiguid.GUID, out []byte) (efivario.Attributes, int, error) {
							copy(out, onieBootContents[4:])
							return efivario.BootServiceAccess | efivario.RuntimeAccess | efivario.NonVolatile, len(onieBootContents) - 4, nil
						})
					bootOrderContents := []byte{
						0x07, 0x00, 0x00, 0x00, 0x07, 0x00, 0x0b, 0x00, 0x00, 0x00, 0x01, 0x00, 0x06, 0x00, 0x02, 0x00,
						0x03, 0x00, 0x04, 0x00, 0x05, 0x00, 0x08, 0x00, 0x09, 0x00, 0x0a, 0x00,
					}
					c.EXPECT().GetSizeHint(gomock.Eq("BootOrder"), gomock.Eq(efivars.GlobalVariable)).Times(1).
						Return(int64(len(bootOrderContents)-4), nil)
					c.EXPECT().Get(gomock.Eq("BootOrder"), gomock.Eq(efivars.GlobalVariable), gomock.Eq(make([]byte, len(bootOrderContents)-4))).Times(1).
						DoAndReturn(func(name string, guid efiguid.GUID, out []byte) (efivario.Attributes, int, error) {
							copy(out, bootOrderContents[4:])
							return efivario.BootServiceAccess | efivario.RuntimeAccess | efivario.NonVolatile, len(bootOrderContents) - 4, nil
						})
				}
			}
			///// END - for MakeONIEDefaultBootEntryAndCleanup() call

			if tt.cmds != nil {
				oldCommand := exec.Command
				defer func() {
					exec.Command = oldCommand
				}()
				cmds := mockexec.NewMockCommands(tt.cmds(t, ctrl))
				defer cmds.Finish()
				exec.Command = cmds.Command()
			}
			err := tt.d.DeletePartitions(tt.args.platform)
			if (err != nil) != tt.wantErr {
				t.Errorf("Devices.DeletePartitions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr && tt.wantErrToBe != nil {
				if !errors.Is(err, tt.wantErrToBe) {
					t.Errorf("Devices.DeletePartitions() error = %v, wantErrToBe %v", err, tt.wantErrToBe)
					return
				}
			}
		})
	}
}

func TestDevices_CreateHedgehogIdentityPartition(t *testing.T) {
	// error test fixtures
	errCreateFailed := errors.New("sgdisk create failed")

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
	disk.Partitions = []*Device{partEFI, partONIE, partDiag}
	partEFI.Disk = disk
	partONIE.Disk = disk
	partDiag.Disk = disk

	// test data with existing identity partition
	partHHIdentity := &Device{
		Uevent: Uevent{
			UeventDevtype: UeventDevtypePartition,
			UeventPartn:   "4",
		},
		GPTPartType: GPTPartTypeHedgehogIdentity,
	}
	diskWithHHIdentity := &Device{
		Uevent: Uevent{
			UeventDevtype: UeventDevtypeDisk,
		},
		Path: "/path/to/disk/device",
	}
	diskWithHHIdentity.Partitions = []*Device{partEFI, partONIE, partDiag, partHHIdentity}
	partHHIdentity.Disk = diskWithHHIdentity

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

	diskNoDev := &Device{
		Uevent: Uevent{
			UeventDevtype: UeventDevtypeDisk,
		},
		Path: "",
	}
	partONIEBrokenDiskNoDev := &Device{
		Uevent: Uevent{
			UeventDevtype: UeventDevtypePartition,
			UeventPartn:   "2",
		},
		GPTPartType: GPTPartTypeONIE,
	}
	partONIEBrokenDiskNoDev.Disk = diskNoDev
	diskNoDev.Partitions = []*Device{partEFI, partONIEBrokenDiskNoDev, partDiag}

	type args struct {
		platform string
	}
	tests := []struct {
		name        string
		d           Devices
		args        args
		wantErr     bool
		wantErrToBe error
		cmds        func(t *testing.T, ctrl *gomock.Controller) []exec.CommandFunc
	}{
		{
			name: "success",
			d: Devices{
				partEFI,
				partONIE,
				partDiag,
			},
			cmds: func(t *testing.T, ctrl *gomock.Controller) []exec.CommandFunc {
				return []exec.CommandFunc{
					mockexec.MockCommand(t, ctrl,
						[]string{
							"sgdisk",
							"--new=4::+100MB",
							"--change-name=4:HEDGEHOG_IDENTITY",
							"--typecode=4:E982E2BD-867C-4D7A-89A2-9C5A9BC5DFDD",
							"/path/to/disk/device",
						},
						func(tc *mockexec.TestCmd) {
							tc.EXPECT().Run().Times(1).DoAndReturn(func() error {
								return tc.IsExpectedCommand()
							})
						},
					),
					mockexec.MockCommand(t, ctrl, []string{"partprobe", "/path/to/disk/device"}, func(tc *mockexec.TestCmd) {
						tc.EXPECT().Run().Times(1).DoAndReturn(func() error {
							return tc.IsExpectedCommand()
						})
					}),
				}
			},
			wantErr: false,
		},
		{
			name: "success but rereading partition table failed",
			d: Devices{
				partEFI,
				partONIE,
				partDiag,
			},
			cmds: func(t *testing.T, ctrl *gomock.Controller) []exec.CommandFunc {
				return []exec.CommandFunc{
					mockexec.MockCommand(t, ctrl,
						[]string{
							"sgdisk",
							"--new=4::+100MB",
							"--change-name=4:HEDGEHOG_IDENTITY",
							"--typecode=4:E982E2BD-867C-4D7A-89A2-9C5A9BC5DFDD",
							"/path/to/disk/device",
						},
						func(tc *mockexec.TestCmd) {
							tc.EXPECT().Run().Times(1).DoAndReturn(func() error {
								return tc.IsExpectedCommand()
							})
						},
					),
					mockexec.MockCommand(t, ctrl, []string{"partprobe", "/path/to/disk/device"}, func(tc *mockexec.TestCmd) {
						tc.EXPECT().Run().Times(1).DoAndReturn(func() error {
							if err := tc.IsExpectedCommand(); err != nil {
								panic(err)
							}
							return fmt.Errorf("rereading partition table failed which should be ignored")
						})
					}),
				}
			},
			wantErr: false,
		},
		{
			name: "create failed",
			d: Devices{
				partEFI,
				partONIE,
				partDiag,
			},
			cmds: func(t *testing.T, ctrl *gomock.Controller) []exec.CommandFunc {
				return []exec.CommandFunc{
					mockexec.MockCommand(t, ctrl,
						[]string{
							"sgdisk",
							"--new=4::+100MB",
							"--change-name=4:HEDGEHOG_IDENTITY",
							"--typecode=4:E982E2BD-867C-4D7A-89A2-9C5A9BC5DFDD",
							"/path/to/disk/device",
						},
						func(tc *mockexec.TestCmd) {
							tc.EXPECT().Run().Times(1).DoAndReturn(func() error {
								if err := tc.IsExpectedCommand(); err != nil {
									return err
								}
								return errCreateFailed
							})
						},
					),
				}
			},
			wantErr:     true,
			wantErrToBe: errCreateFailed,
		},
		{
			name: "partition already exists",
			d: Devices{
				partEFI,
				partONIE,
				partDiag,
				partHHIdentity,
			},
			wantErr:     true,
			wantErrToBe: ErrPartitionExists,
		},
		{
			name: "ONIE partition missing",
			d: Devices{
				partEFI,
				partDiag,
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
			},
			wantErr:     true,
			wantErrToBe: ErrBrokenDiscovery,
		},
		{
			name: "disk with no device node",
			d: Devices{
				partEFI,
				partONIEBrokenDiskNoDev,
				partDiag,
			},
			wantErr:     true,
			wantErrToBe: ErrNoDeviceNode,
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
			err := tt.d.CreateHedgehogIdentityPartition(tt.args.platform)
			if (err != nil) != tt.wantErr {
				t.Errorf("Devices.CreateHedgehogIdentityPartition() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr && tt.wantErrToBe != nil {
				if !errors.Is(err, tt.wantErrToBe) {
					t.Errorf("Devices.CreateHedgehogIdentityPartition() error = %v, wantErrToBe %v", err, tt.wantErrToBe)
					return
				}
			}
		})
	}
}
