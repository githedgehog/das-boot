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

package devid

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/golang/mock/gomock"
	"go.githedgehog.com/dasboot/pkg/exec"
	"go.githedgehog.com/dasboot/test/mock/mockexec"
)

func Test_idFromVendorIDAndSerial(t *testing.T) {
	errOnieSysinfoFailed := errors.New("onie-sysinfo failed")
	errOnieSyseepromFailed := errors.New("onie-syseeprom failed")

	tests := []struct {
		name        string
		want        string
		wantErr     bool
		wantErrToBe error
		cmds        func(t *testing.T, ctrl *gomock.Controller) []exec.CommandFunc
	}{
		{
			name: "success",
			cmds: func(t *testing.T, ctrl *gomock.Controller) []exec.CommandFunc {
				return []exec.CommandFunc{
					mockexec.MockCommand(t, ctrl, []string{"onie-sysinfo", "-i"}, func(tc *mockexec.TestCmd) {
						tc.EXPECT().Output().Times(1).DoAndReturn(func() ([]byte, error) {
							if err := tc.IsExpectedCommand(); err != nil {
								return nil, err
							}
							return []byte("42623"), nil
						})
					}),
					mockexec.MockCommand(t, ctrl, []string{"onie-syseeprom", "-g", "0x23"}, func(tc *mockexec.TestCmd) {
						tc.EXPECT().Output().Times(1).DoAndReturn(func() ([]byte, error) {
							if err := tc.IsExpectedCommand(); err != nil {
								return nil, err
							}
							return []byte("42135"), nil
						})
					}),
				}
			},
			want:    "bda28d62-b2e4-5eba-b490-19ffa25b68ac",
			wantErr: false,
		},
		{
			name: "onie-sysinfo fails",
			cmds: func(t *testing.T, ctrl *gomock.Controller) []exec.CommandFunc {
				return []exec.CommandFunc{
					mockexec.MockCommand(t, ctrl, []string{"onie-sysinfo", "-i"}, func(tc *mockexec.TestCmd) {
						tc.EXPECT().Output().Times(1).DoAndReturn(func() ([]byte, error) {
							if err := tc.IsExpectedCommand(); err != nil {
								return nil, err
							}
							return nil, errOnieSysinfoFailed
						})
					}),
				}
			},
			wantErr:     true,
			wantErrToBe: errOnieSysinfoFailed,
		},
		{
			name: "onie-syseeprom fails",
			cmds: func(t *testing.T, ctrl *gomock.Controller) []exec.CommandFunc {
				return []exec.CommandFunc{
					mockexec.MockCommand(t, ctrl, []string{"onie-sysinfo", "-i"}, func(tc *mockexec.TestCmd) {
						tc.EXPECT().Output().Times(1).DoAndReturn(func() ([]byte, error) {
							if err := tc.IsExpectedCommand(); err != nil {
								return nil, err
							}
							return []byte("42623"), nil
						})
					}),
					mockexec.MockCommand(t, ctrl, []string{"onie-syseeprom", "-g", "0x23"}, func(tc *mockexec.TestCmd) {
						tc.EXPECT().Output().Times(1).DoAndReturn(func() ([]byte, error) {
							if err := tc.IsExpectedCommand(); err != nil {
								return nil, err
							}
							return nil, errOnieSyseepromFailed
						})
					}),
				}
			},
			wantErr:     true,
			wantErrToBe: errOnieSyseepromFailed,
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
			got, err := idFromVendorIDAndSerial()
			if (err != nil) != tt.wantErr {
				t.Errorf("idFromVendorIDAndSerial() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr && tt.wantErrToBe != nil {
				if !errors.Is(err, tt.wantErrToBe) {
					t.Errorf("Device.Delete() error = %v, wantErrToBe %v", err, tt.wantErrToBe)
					return
				}
			}
			if got != tt.want {
				t.Errorf("idFromVendorIDAndSerial() = %v, want %v", got, tt.want)
				return
			}
		})
	}
}

func Test_idFromSystemUUID(t *testing.T) {
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	errIoReadAll := errors.New("io.ReadAll error")
	tests := []struct {
		name        string
		want        string
		wantErr     bool
		wantErrToBe error
		rootPath    string
		ioReadAll   func(r io.Reader) ([]byte, error)
	}{
		{
			name:     "success",
			rootPath: filepath.Join(pwd, "testdata", "idFromSystemUUID", "one"),
			want:     "a56aec4d-100e-4af0-8206-02a50f5e96f4",
			wantErr:  false,
		},
		{
			name:     "invalid uuid",
			rootPath: filepath.Join(pwd, "testdata", "idFromSystemUUID", "two"),
			wantErr:  true,
		},
		{
			name:     "reading from file fails",
			rootPath: filepath.Join(pwd, "testdata", "idFromSystemUUID", "one"),
			ioReadAll: func(r io.Reader) ([]byte, error) {
				return nil, errIoReadAll
			},
			wantErr:     true,
			wantErrToBe: errIoReadAll,
		},
		{
			name:        "path does not exist",
			rootPath:    filepath.Join(pwd, "testdata", "idFromSystemUUID", "does-not-exist"),
			wantErr:     true,
			wantErrToBe: os.ErrNotExist,
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
			if tt.ioReadAll != nil {
				oldIoReadAll := ioReadAll
				defer func() {
					ioReadAll = oldIoReadAll
				}()
				ioReadAll = tt.ioReadAll
			}
			got, err := idFromSystemUUID()
			if (err != nil) != tt.wantErr {
				t.Errorf("idFromSystemUUID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr && tt.wantErrToBe != nil {
				if !errors.Is(err, tt.wantErrToBe) {
					t.Errorf("idFromSystemUUID() error = %v, wantErrToBe %v", err, tt.wantErrToBe)
					return
				}
			}
			if got != tt.want {
				t.Errorf("idFromSystemUUID() = %v, want %v", got, tt.want)
				return
			}
		})
	}
}

func Test_idFromCPUInfo(t *testing.T) {
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	// this only works on ARM
	// only has an effect on the generated UUID
	oldArch := arch
	defer func() {
		arch = oldArch
	}()
	arch = "arm64"

	tests := []struct {
		name        string
		want        string
		wantErr     bool
		wantErrToBe error
		rootPath    string
	}{
		{
			name:     "success",
			rootPath: filepath.Join(pwd, "testdata", "idFromCPUInfo", "two"),
			want:     "677b8b78-f321-5e46-b4f8-e8569a025a20",
		},
		{
			name:        "bogus CPU serial",
			rootPath:    filepath.Join(pwd, "testdata", "idFromCPUInfo", "three"),
			wantErr:     true,
			wantErrToBe: ErrBogusCPUSerial,
		},
		{
			name:        "CPU serial not found",
			rootPath:    filepath.Join(pwd, "testdata", "idFromCPUInfo", "one"),
			wantErr:     true,
			wantErrToBe: ErrCPUSerialNotFound,
		},
		{
			name:        "opening proc cpuinfo fails",
			rootPath:    filepath.Join(pwd, "testdata", "idFromCPUInfo", "does-not-exist"),
			wantErr:     true,
			wantErrToBe: os.ErrNotExist,
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
			got, err := idFromCPUInfo()
			if (err != nil) != tt.wantErr {
				t.Errorf("idFromCPUInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr && tt.wantErrToBe != nil {
				if !errors.Is(err, tt.wantErrToBe) {
					t.Errorf("idFromCPUInfo() error = %v, wantErrToBe %v", err, tt.wantErrToBe)
					return
				}
			}
			if got != tt.want {
				t.Errorf("idFromCPUInfo() = %v, want %v", got, tt.want)
				return
			}
		})
	}
}

func Test_idFromMACAddresses(t *testing.T) {
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	errIoReadAll := errors.New("io.ReadAll failed")
	tests := []struct {
		name        string
		want        string
		wantErr     bool
		wantErrToBe error
		rootPath    string
		ioReadAll   func(r io.Reader) ([]byte, error)
	}{
		{
			name:     "success",
			rootPath: filepath.Join(pwd, "testdata", "idFromMACAddresses", "one"),
			want:     "90286cbb-a0d5-5e4b-9c97-12bb2869389b",
		},
		{
			name:        "sysfs not mounted",
			rootPath:    filepath.Join(pwd, "testdata", "idFromMACAddresses", "does-not-exist"),
			wantErr:     true,
			wantErrToBe: ErrNoNetdevs,
		},
		{
			name:        "no MAC addresses for netdevs",
			rootPath:    filepath.Join(pwd, "testdata", "idFromMACAddresses", "two"),
			wantErr:     true,
			wantErrToBe: ErrNoMACAddressesForNetdevs,
		},
		{
			name:     "reading from address file fails",
			rootPath: filepath.Join(pwd, "testdata", "idFromMACAddresses", "one"),
			ioReadAll: func(r io.Reader) ([]byte, error) {
				return nil, errIoReadAll
			},
			wantErr:     true,
			wantErrToBe: errIoReadAll,
		},
		{
			name:        "netdev has no address file",
			rootPath:    filepath.Join(pwd, "testdata", "idFromMACAddresses", "three"),
			wantErr:     true,
			wantErrToBe: os.ErrNotExist,
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
			if tt.ioReadAll != nil {
				oldIoReadAll := ioReadAll
				defer func() {
					ioReadAll = oldIoReadAll
				}()
				ioReadAll = tt.ioReadAll
			}
			got, err := idFromMACAddresses()
			if (err != nil) != tt.wantErr {
				t.Errorf("idFromMACAddresses() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr && tt.wantErrToBe != nil {
				if !errors.Is(err, tt.wantErrToBe) {
					t.Errorf("idFromCPUInfo() error = %v, wantErrToBe %v", err, tt.wantErrToBe)
					return
				}
			}
			if got != tt.want {
				t.Errorf("idFromMACAddresses() = %v, want %v", got, tt.want)
				return
			}
		})
	}
}

func TestID(t *testing.T) {
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	tests := []struct {
		name     string
		want     string
		rootPath string
		arch     string
		cmds     func(t *testing.T, ctrl *gomock.Controller) []exec.CommandFunc
	}{
		{
			name: "success with ONIE builtins",
			cmds: func(t *testing.T, ctrl *gomock.Controller) []exec.CommandFunc {
				return []exec.CommandFunc{
					mockexec.MockCommand(t, ctrl, []string{"onie-sysinfo", "-i"}, func(tc *mockexec.TestCmd) {
						tc.EXPECT().Output().Times(1).DoAndReturn(func() ([]byte, error) {
							return []byte("42623"), nil
						})
					}),
					mockexec.MockCommand(t, ctrl, []string{"onie-syseeprom", "-g", "0x23"}, func(tc *mockexec.TestCmd) {
						tc.EXPECT().Output().Times(1).DoAndReturn(func() ([]byte, error) {
							return []byte("42135"), nil
						})
					}),
				}
			},
			want: "bda28d62-b2e4-5eba-b490-19ffa25b68ac",
		},
		{
			name: "ONIE builtins fail but success with DMI fallback",
			cmds: func(t *testing.T, ctrl *gomock.Controller) []exec.CommandFunc {
				return []exec.CommandFunc{
					mockexec.MockCommand(t, ctrl, []string{"onie-sysinfo", "-i"}, func(tc *mockexec.TestCmd) {
						tc.EXPECT().Output().Times(1).DoAndReturn(func() ([]byte, error) {
							return nil, fmt.Errorf("utter failure")
						})
					}),
				}
			},
			arch:     "amd64",
			rootPath: filepath.Join(pwd, "testdata", "ID", "one"),
			want:     "a56aec4d-100e-4af0-8206-02a50f5e96f4",
		},
		{
			name: "ONIE builtins fail but success with CPU Serial number fallback",
			cmds: func(t *testing.T, ctrl *gomock.Controller) []exec.CommandFunc {
				return []exec.CommandFunc{
					mockexec.MockCommand(t, ctrl, []string{"onie-sysinfo", "-i"}, func(tc *mockexec.TestCmd) {
						tc.EXPECT().Output().Times(1).DoAndReturn(func() ([]byte, error) {
							return nil, fmt.Errorf("utter failure")
						})
					}),
				}
			},
			arch:     "arm64",
			rootPath: filepath.Join(pwd, "testdata", "ID", "two"),
			want:     "677b8b78-f321-5e46-b4f8-e8569a025a20",
		},
		{
			name: "ONIE builtins fail, secondary fallback fails, but success with MAC addresses on amd64",
			cmds: func(t *testing.T, ctrl *gomock.Controller) []exec.CommandFunc {
				return []exec.CommandFunc{
					mockexec.MockCommand(t, ctrl, []string{"onie-sysinfo", "-i"}, func(tc *mockexec.TestCmd) {
						tc.EXPECT().Output().Times(1).DoAndReturn(func() ([]byte, error) {
							return nil, fmt.Errorf("utter failure")
						})
					}),
				}
			},
			arch:     "amd64",
			rootPath: filepath.Join(pwd, "testdata", "ID", "three"),
			want:     "90286cbb-a0d5-5e4b-9c97-12bb2869389b",
		},
		{
			name: "ONIE builtins fail, secondary fallback fails, but success with MAC addresses on arm64",
			cmds: func(t *testing.T, ctrl *gomock.Controller) []exec.CommandFunc {
				return []exec.CommandFunc{
					mockexec.MockCommand(t, ctrl, []string{"onie-sysinfo", "-i"}, func(tc *mockexec.TestCmd) {
						tc.EXPECT().Output().Times(1).DoAndReturn(func() ([]byte, error) {
							return nil, fmt.Errorf("utter failure")
						})
					}),
				}
			},
			arch:     "arm64",
			rootPath: filepath.Join(pwd, "testdata", "ID", "three"),
			want:     "90286cbb-a0d5-5e4b-9c97-12bb2869389b",
		},
		{
			name: "everything fails",
			cmds: func(t *testing.T, ctrl *gomock.Controller) []exec.CommandFunc {
				return []exec.CommandFunc{
					mockexec.MockCommand(t, ctrl, []string{"onie-sysinfo", "-i"}, func(tc *mockexec.TestCmd) {
						tc.EXPECT().Output().Times(1).DoAndReturn(func() ([]byte, error) {
							return nil, fmt.Errorf("utter failure")
						})
					}),
				}
			},
			arch:     "ppc64",
			rootPath: filepath.Join(pwd, "testdata", "ID", "one"),
			want:     "",
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
			if tt.rootPath != "" {
				oldRootPath := rootPath
				defer func() {
					rootPath = oldRootPath
				}()
				rootPath = tt.rootPath
			}
			if tt.arch != "" {
				oldArch := arch
				defer func() {
					arch = oldArch
				}()
				arch = tt.arch
			}
			if got := ID(); got != tt.want {
				t.Errorf("ID() = %v, want %v", got, tt.want)
			}
		})
	}
}
