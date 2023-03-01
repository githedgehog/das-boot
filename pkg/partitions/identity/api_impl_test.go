package identity

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"go.githedgehog.com/dasboot/pkg/partitions"
	"go.githedgehog.com/dasboot/test/mock/mockio"
	"go.githedgehog.com/dasboot/test/mock/mockio/mockfs"
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
				return
			}
		})
	}
}

func TestInit(t *testing.T) {
	type args struct {
		d *partitions.Device
	}
	d := &partitions.Device{
		Uevent: partitions.Uevent{
			partitions.UeventDevtype: partitions.UeventDevtypePartition,
		},
		GPTPartType: partitions.GPTPartTypeHedgehogIdentity,
	}
	errRemoveAllFailed := errors.New("RemoveAll() failed tragically")
	errOpenVersionFileForWriting := errors.New("OpenFile() for version file for writing failed")
	errMkdirIdentityDir := errors.New("Mkdir() for identity dir failed")
	errMkdirLocationDir := errors.New("Mkdir() for location dir failed")
	errWritingJSONToVersionFileFailed := errors.New("Write() to version file during JSON encoding failedP")
	tests := []struct {
		name        string
		args        args
		want        IdentityPartition
		wantErr     bool
		wantErrToBe error
		pre         func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS)
	}{
		{
			name: "success",
			args: args{d: d},
			want: &api{dev: d},
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				// version file check
				mfs.EXPECT().Stat(gomock.Eq(versionFilePath)).Times(1).Return(nil, os.ErrNotExist)

				// cleaning up directory
				e1 := mockfs.NewMockDirEntry(ctrl)
				e1.EXPECT().Name().Times(1).Return("lost+found")
				e2 := mockfs.NewMockDirEntry(ctrl)
				e2.EXPECT().Name().Times(2).Return("removeme")
				mfs.EXPECT().ReadDir(gomock.Eq("")).Times(1).Return([]fs.DirEntry{e1, e2}, nil)
				mfs.EXPECT().RemoveAll(gomock.Eq("removeme")).Times(1).Return(nil)

				// writing version file
				f := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().OpenFile(gomock.Eq(versionFilePath), gomock.Eq(os.O_CREATE|os.O_TRUNC|os.O_WRONLY), gomock.Eq(fs.FileMode(0644))).Times(1).Return(f, nil)
				f.EXPECT().Close().Times(1)
				versString := `{"version":1}` + "\n"
				f.EXPECT().Write(gomock.Eq([]byte(versString))).Times(1).Return(len(versString), nil)

				// creating directories
				mfs.EXPECT().Mkdir(gomock.Eq(identityDirPath), gomock.Eq(fs.FileMode(0755))).Times(1).Return(nil)
				mfs.EXPECT().Mkdir(gomock.Eq(locationDirPath), gomock.Eq(fs.FileMode(0755))).Times(1).Return(nil)
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
			name:        "stat on version file fails",
			args:        args{d: d},
			wantErr:     true,
			wantErrToBe: os.ErrPermission,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				// version file check
				mfs.EXPECT().Stat(gomock.Eq(versionFilePath)).Times(1).Return(nil, os.ErrPermission)
			},
		},
		{
			name:        "partition already initialized",
			args:        args{d: d},
			wantErr:     true,
			wantErrToBe: ErrAlreadyInitialized,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				// version file check
				mfs.EXPECT().Stat(gomock.Eq(versionFilePath)).Times(1).Return(nil, nil)
			},
		},
		{
			name:        "reading partition entries fails",
			args:        args{d: d},
			wantErr:     true,
			wantErrToBe: os.ErrPermission,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				// version file check
				mfs.EXPECT().Stat(gomock.Eq(versionFilePath)).Times(1).Return(nil, os.ErrNotExist)

				// cleaning up directory
				mfs.EXPECT().ReadDir(gomock.Eq("")).Times(1).Return(nil, os.ErrPermission)

			},
		},
		{
			name:        "removing partition entries fails",
			args:        args{d: d},
			wantErr:     true,
			wantErrToBe: errRemoveAllFailed,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				// version file check
				mfs.EXPECT().Stat(gomock.Eq(versionFilePath)).Times(1).Return(nil, os.ErrNotExist)

				// cleaning up directory
				e1 := mockfs.NewMockDirEntry(ctrl)
				e1.EXPECT().Name().Times(1).Return("lost+found")
				e2 := mockfs.NewMockDirEntry(ctrl)
				e2.EXPECT().Name().Times(3).Return("removeme")
				mfs.EXPECT().ReadDir(gomock.Eq("")).Times(1).Return([]fs.DirEntry{e1, e2}, nil)
				mfs.EXPECT().RemoveAll(gomock.Eq("removeme")).Times(1).Return(errRemoveAllFailed)
			},
		},
		{
			name:        "opening vesion file for writing fails",
			args:        args{d: d},
			wantErr:     true,
			wantErrToBe: errOpenVersionFileForWriting,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				// version file check
				mfs.EXPECT().Stat(gomock.Eq(versionFilePath)).Times(1).Return(nil, os.ErrNotExist)

				// cleaning up directory
				e1 := mockfs.NewMockDirEntry(ctrl)
				e1.EXPECT().Name().Times(1).Return("lost+found")
				e2 := mockfs.NewMockDirEntry(ctrl)
				e2.EXPECT().Name().Times(2).Return("removeme")
				mfs.EXPECT().ReadDir(gomock.Eq("")).Times(1).Return([]fs.DirEntry{e1, e2}, nil)
				mfs.EXPECT().RemoveAll(gomock.Eq("removeme")).Times(1).Return(nil)

				// writing version file
				mfs.EXPECT().OpenFile(gomock.Eq(versionFilePath), gomock.Eq(os.O_CREATE|os.O_TRUNC|os.O_WRONLY), gomock.Eq(fs.FileMode(0644))).Times(1).Return(nil, errOpenVersionFileForWriting)
			},
		},
		{
			name:        "writing JSON to version file fails",
			args:        args{d: d},
			wantErr:     true,
			wantErrToBe: errWritingJSONToVersionFileFailed,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				// version file check
				mfs.EXPECT().Stat(gomock.Eq(versionFilePath)).Times(1).Return(nil, os.ErrNotExist)

				// cleaning up directory
				e1 := mockfs.NewMockDirEntry(ctrl)
				e1.EXPECT().Name().Times(1).Return("lost+found")
				e2 := mockfs.NewMockDirEntry(ctrl)
				e2.EXPECT().Name().Times(2).Return("removeme")
				mfs.EXPECT().ReadDir(gomock.Eq("")).Times(1).Return([]fs.DirEntry{e1, e2}, nil)
				mfs.EXPECT().RemoveAll(gomock.Eq("removeme")).Times(1).Return(nil)

				// writing version file
				f := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().OpenFile(gomock.Eq(versionFilePath), gomock.Eq(os.O_CREATE|os.O_TRUNC|os.O_WRONLY), gomock.Eq(fs.FileMode(0644))).Times(1).Return(f, nil)
				f.EXPECT().Close().Times(1)
				versString := `{"version":1}` + "\n"
				f.EXPECT().Write(gomock.Eq([]byte(versString))).Times(1).Return(0, errWritingJSONToVersionFileFailed)
			},
		},
		{
			name:        "making identity directory fails",
			args:        args{d: d},
			wantErr:     true,
			wantErrToBe: errMkdirIdentityDir,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				// version file check
				mfs.EXPECT().Stat(gomock.Eq(versionFilePath)).Times(1).Return(nil, os.ErrNotExist)

				// cleaning up directory
				e1 := mockfs.NewMockDirEntry(ctrl)
				e1.EXPECT().Name().Times(1).Return("lost+found")
				e2 := mockfs.NewMockDirEntry(ctrl)
				e2.EXPECT().Name().Times(2).Return("removeme")
				mfs.EXPECT().ReadDir(gomock.Eq("")).Times(1).Return([]fs.DirEntry{e1, e2}, nil)
				mfs.EXPECT().RemoveAll(gomock.Eq("removeme")).Times(1).Return(nil)

				// writing version file
				f := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().OpenFile(gomock.Eq(versionFilePath), gomock.Eq(os.O_CREATE|os.O_TRUNC|os.O_WRONLY), gomock.Eq(fs.FileMode(0644))).Times(1).Return(f, nil)
				f.EXPECT().Close().Times(1)
				versString := `{"version":1}` + "\n"
				f.EXPECT().Write(gomock.Eq([]byte(versString))).Times(1).Return(len(versString), nil)

				// creating directories
				mfs.EXPECT().Mkdir(gomock.Eq(identityDirPath), gomock.Eq(fs.FileMode(0755))).Times(1).Return(errMkdirIdentityDir)
			},
		},
		{
			name:        "making location directory fails",
			args:        args{d: d},
			wantErr:     true,
			wantErrToBe: errMkdirLocationDir,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				// version file check
				mfs.EXPECT().Stat(gomock.Eq(versionFilePath)).Times(1).Return(nil, os.ErrNotExist)

				// cleaning up directory
				e1 := mockfs.NewMockDirEntry(ctrl)
				e1.EXPECT().Name().Times(1).Return("lost+found")
				e2 := mockfs.NewMockDirEntry(ctrl)
				e2.EXPECT().Name().Times(2).Return("removeme")
				mfs.EXPECT().ReadDir(gomock.Eq("")).Times(1).Return([]fs.DirEntry{e1, e2}, nil)
				mfs.EXPECT().RemoveAll(gomock.Eq("removeme")).Times(1).Return(nil)

				// writing version file
				f := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().OpenFile(gomock.Eq(versionFilePath), gomock.Eq(os.O_CREATE|os.O_TRUNC|os.O_WRONLY), gomock.Eq(fs.FileMode(0644))).Times(1).Return(f, nil)
				f.EXPECT().Close().Times(1)
				versString := `{"version":1}` + "\n"
				f.EXPECT().Write(gomock.Eq([]byte(versString))).Times(1).Return(len(versString), nil)

				// creating directories
				mfs.EXPECT().Mkdir(gomock.Eq(identityDirPath), gomock.Eq(fs.FileMode(0755))).Times(1).Return(nil)
				mfs.EXPECT().Mkdir(gomock.Eq(locationDirPath), gomock.Eq(fs.FileMode(0755))).Times(1).Return(errMkdirLocationDir)
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
			got, err := Init(tt.args.d)
			if (err != nil) != tt.wantErr {
				t.Errorf("Init() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr && tt.wantErrToBe != nil {
				if !errors.Is(err, tt.wantErrToBe) {
					t.Errorf("Open() error = %v, wantErrToBe %v", err, tt.wantErrToBe)
					return
				}
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Init() = %v, want %v", got, tt.want)
				return
			}
		})
	}
}
