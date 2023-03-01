package identity

import (
	"errors"
	"io"
	"os"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"go.githedgehog.com/dasboot/pkg/partitions"
	"go.githedgehog.com/dasboot/test/mock/mockio"
	"go.githedgehog.com/dasboot/test/mock/mockpartitions"
)

func TestOpen(t *testing.T) {
	type args struct {
		d *partitions.Device
	}
	d := &partitions.Device{
		Uevent: partitions.Uevent{
			partitions.UeventDevtype: partitions.UeventDevtypePartition,
		},
		GPTPartType: partitions.GPTPartTypeHedgehogIdentity,
	}
	tests := []struct {
		name        string
		args        args
		want        IdentityPartition
		wantErr     bool
		wantErrToBe error
		pre         func(t *testing.T, ctrl *gomock.Controller, fs *mockpartitions.MockFS)
	}{
		{
			name:    "success",
			args:    args{d: d},
			want:    &api{dev: d},
			wantErr: false,
			pre: func(t *testing.T, ctrl *gomock.Controller, fs *mockpartitions.MockFS) {
				versionFile := mockio.NewMockReadWriteCloser(ctrl)
				versionFile.EXPECT().Read(gomock.Any()).Times(1).DoAndReturn(func(b []byte) (int, error) {
					versionBytes := `{"version":1}`
					copy(b, []byte(versionBytes))
					return len(versionBytes), io.EOF
				})
				versionFile.EXPECT().Close().Times(1)
				fs.EXPECT().Open(gomock.Eq(versionFilePath)).Times(1).Return(versionFile, nil)
			},
		},
		{
			name: "not an identity partition",
			args: args{d: &partitions.Device{
				Uevent: partitions.Uevent{
					partitions.UeventDevtype: partitions.UeventDevtypePartition,
				},
				GPTPartType: partitions.GPTPartTypeHedgehogLocation,
			}},
			wantErr:     true,
			wantErrToBe: ErrWrongDevice,
		},
		{
			name:        "partition is not initialized",
			args:        args{d: d},
			wantErr:     true,
			wantErrToBe: ErrUninitializedPartition,
			pre: func(t *testing.T, ctrl *gomock.Controller, fs *mockpartitions.MockFS) {
				fs.EXPECT().Open(gomock.Eq(versionFilePath)).Times(1).Return(nil, os.ErrNotExist)
			},
		},
		{
			name:        "opening version file fails",
			args:        args{d: d},
			wantErr:     true,
			wantErrToBe: io.ErrUnexpectedEOF,
			pre: func(t *testing.T, ctrl *gomock.Controller, fs *mockpartitions.MockFS) {
				fs.EXPECT().Open(gomock.Eq(versionFilePath)).Times(1).Return(nil, io.ErrUnexpectedEOF)
			},
		},
		{
			name:    "version file contains invalid JSON",
			args:    args{d: d},
			wantErr: true,
			pre: func(t *testing.T, ctrl *gomock.Controller, fs *mockpartitions.MockFS) {
				versionFile := mockio.NewMockReadWriteCloser(ctrl)
				versionFile.EXPECT().Read(gomock.Any()).Times(1).DoAndReturn(func(b []byte) (int, error) {
					versionBytes := `{"versio`
					copy(b, []byte(versionBytes))
					return len(versionBytes), io.EOF
				})
				versionFile.EXPECT().Close().Times(1)
				fs.EXPECT().Open(gomock.Eq(versionFilePath)).Times(1).Return(versionFile, nil)
			},
		},
		{
			name:        "unsupported version",
			args:        args{d: d},
			wantErr:     true,
			wantErrToBe: ErrUnsupportedVersion,
			pre: func(t *testing.T, ctrl *gomock.Controller, fs *mockpartitions.MockFS) {
				versionFile := mockio.NewMockReadWriteCloser(ctrl)
				versionFile.EXPECT().Read(gomock.Any()).Times(1).DoAndReturn(func(b []byte) (int, error) {
					versionBytes := `{"version":2}`
					copy(b, []byte(versionBytes))
					return len(versionBytes), io.EOF
				})
				versionFile.EXPECT().Close().Times(1)
				fs.EXPECT().Open(gomock.Eq(versionFilePath)).Times(1).Return(versionFile, nil)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockfs := mockpartitions.NewMockFS(ctrl)
			if tt.args.d != nil {
				tt.args.d.FS = mockfs
			}
			if tt.pre != nil {
				tt.pre(t, ctrl, mockfs)
			}
			got, err := Open(tt.args.d)
			if (err != nil) != tt.wantErr {
				t.Errorf("Open() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr && tt.wantErrToBe != nil {
				if !errors.Is(err, tt.wantErrToBe) {
					t.Errorf("Open() error = %v, wantErrToBe %v", err, tt.wantErrToBe)
					return
				}
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Open() = %v, want %v", got, tt.want)
			}
		})
	}
}
