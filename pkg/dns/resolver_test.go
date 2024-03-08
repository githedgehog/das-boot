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

package dns

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"go.githedgehog.com/dasboot/test/mock/mockio"
)

func TestSetSystemResolvers(t *testing.T) {
	t.Run("osOpenFile", func(t *testing.T) {
		tf, err := os.CreateTemp("", "TestSetSystemResolvers-osOpenFile-")
		if err != nil {
			panic(err)
		}
		path := tf.Name()
		tf.Close()
		defer os.Remove(path)
		f, err := osOpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
		if err != nil {
			t.Errorf("os.OpenFile failed: %v", err)
			return
		}
		defer f.Close()
		if f == nil {
			t.Errorf("os.OpenFile opened with nil return")
			return
		}
	})
	writeError := errors.New("write error")
	type args struct {
		servers []string
	}
	tests := []struct {
		name        string
		args        args
		want        []byte
		wantErr     bool
		wantErrToBe error
		osOpenFile  func(t *testing.T, ctrl *gomock.Controller, want *[]byte) func(name string, flag int, perm fs.FileMode) (io.ReadWriteCloser, error)
	}{
		{
			name: "success",
			args: args{servers: []string{"127.0.0.1", "127.0.0.2"}},
			osOpenFile: func(t *testing.T, ctrl *gomock.Controller, want *[]byte) func(name string, flag int, perm fs.FileMode) (io.ReadWriteCloser, error) {
				f := mockio.NewMockReadWriteCloser(ctrl)
				f.EXPECT().Close().Times(1)
				f.EXPECT().Write(gomock.Any()).Times(6).DoAndReturn(func(b []byte) (int, error) {
					*want = append(*want, b...)
					return len(b), nil
				})
				return func(name string, flag int, perm fs.FileMode) (io.ReadWriteCloser, error) {
					if name != etcResolvConfPath {
						panic("unexpected file path")
					}
					return f, nil
				}
			},
			want: []byte(`# Hedgehog DAS BOOT
# This DNS resolver configuration was being derived by the stage0 installer.

nameserver 127.0.0.1
nameserver 127.0.0.2

options edns0 trust-ad timeout:5 attempts:2 rotate
search .
`),
		},
		{
			name:        "open failed",
			args:        args{servers: []string{"127.0.0.1", "127.0.0.2"}},
			wantErr:     true,
			wantErrToBe: os.ErrNotExist,
			osOpenFile: func(t *testing.T, ctrl *gomock.Controller, want *[]byte) func(name string, flag int, perm fs.FileMode) (io.ReadWriteCloser, error) {
				return func(name string, flag int, perm fs.FileMode) (io.ReadWriteCloser, error) {
					if name != etcResolvConfPath {
						panic("unexpected file path")
					}
					return nil, os.ErrNotExist
				}
			},
		},
		{
			name:        "empty servers list",
			args:        args{servers: nil},
			wantErr:     true,
			wantErrToBe: ErrNoServers,
			// not necessary, but just a precaution
			osOpenFile: func(t *testing.T, ctrl *gomock.Controller, want *[]byte) func(name string, flag int, perm fs.FileMode) (io.ReadWriteCloser, error) {
				return func(name string, flag int, perm fs.FileMode) (io.ReadWriteCloser, error) {
					return nil, os.ErrNotExist
				}
			},
		},
		{
			name:        "invalid IP in list",
			args:        args{servers: []string{"not an IP"}},
			wantErr:     true,
			wantErrToBe: ErrInvalidIPAddress,
			// not necessary, but just a precaution
			osOpenFile: func(t *testing.T, ctrl *gomock.Controller, want *[]byte) func(name string, flag int, perm fs.FileMode) (io.ReadWriteCloser, error) {
				return func(name string, flag int, perm fs.FileMode) (io.ReadWriteCloser, error) {
					return nil, os.ErrNotExist
				}
			},
		},
		{
			name:        "write failed",
			args:        args{servers: []string{"127.0.0.1", "127.0.0.2"}},
			wantErr:     true,
			wantErrToBe: writeError,
			osOpenFile: func(t *testing.T, ctrl *gomock.Controller, want *[]byte) func(name string, flag int, perm fs.FileMode) (io.ReadWriteCloser, error) {
				f := mockio.NewMockReadWriteCloser(ctrl)
				f.EXPECT().Close().Times(1)
				f.EXPECT().Write(gomock.Any()).Times(1).Return(0, writeError)
				return func(name string, flag int, perm fs.FileMode) (io.ReadWriteCloser, error) {
					if name != etcResolvConfPath {
						panic("unexpected file path")
					}
					return f, nil
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			var want []byte
			if tt.osOpenFile != nil {
				oldOsOpenFile := osOpenFile
				defer func() {
					osOpenFile = oldOsOpenFile
				}()
				osOpenFile = tt.osOpenFile(t, ctrl, &want)
			}
			err := SetSystemResolvers(tt.args.servers)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetSystemResolvers() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr && tt.wantErrToBe != nil {
				if !errors.Is(err, tt.wantErrToBe) {
					t.Errorf("SetSystemResolvers() error = %v, wantErrToBe %v", err, tt.wantErrToBe)
					return
				}
			}
			if len(want) > 0 && !reflect.DeepEqual(want, tt.want) {
				t.Errorf("SetSystemResolvers()\n\twant = \n'%v'\n\tgot = \n'%v'\n", string(tt.want), string(want))
			}
		})
	}
}
