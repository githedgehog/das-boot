package identity

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"go.githedgehog.com/dasboot/pkg/partitions"
	"go.githedgehog.com/dasboot/pkg/partitions/location"
	"go.githedgehog.com/dasboot/test/mock/mockio"
	"go.githedgehog.com/dasboot/test/mock/mockio/mockfs"
	"go.githedgehog.com/dasboot/test/mock/mockpartitions"
	"go.githedgehog.com/dasboot/test/mock/mockpartitions/mocklocation"
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

func Test_api_GenerateClientKeyPair(t *testing.T) {
	errGeneratePrivateKey := errors.New("GeneratePrivateKey() tragically failed")
	errMarshalPrivateKey := errors.New("MarshalECPrivateKey() tragically failed")
	tests := []struct {
		name                    string
		wantErr                 bool
		wantErrToBe             error
		pre                     func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS)
		ecdsaGenerateKey        func(c elliptic.Curve, rand io.Reader) (*ecdsa.PrivateKey, error)
		x509MarshalECPrivateKey func(key *ecdsa.PrivateKey) ([]byte, error)
	}{
		{
			name:    "success",
			wantErr: false,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				f := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().OpenFile(gomock.Eq(clientKeyPath), gomock.Eq(os.O_CREATE|os.O_TRUNC|os.O_WRONLY), gomock.Eq(fs.FileMode(0644))).Times(1).Return(f, nil)
				f.EXPECT().Close().Times(1).Return(nil)
				f.EXPECT().Write(gomock.Any()).Times(1).DoAndReturn(func(b []byte) (int, error) {
					return len(b), nil
				})
				mfs.EXPECT().Remove(gomock.Eq(clientCSRPath)).Times(1).Return(nil)
				mfs.EXPECT().Remove(gomock.Eq(clientCertPath)).Times(1).Return(os.ErrNotExist)
			},
		},
		{
			name:    "success, but deleting previous CSR fails",
			wantErr: true,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				f := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().OpenFile(gomock.Eq(clientKeyPath), gomock.Eq(os.O_CREATE|os.O_TRUNC|os.O_WRONLY), gomock.Eq(fs.FileMode(0644))).Times(1).Return(f, nil)
				f.EXPECT().Close().Times(1).Return(nil)
				f.EXPECT().Write(gomock.Any()).Times(1).DoAndReturn(func(b []byte) (int, error) {
					return len(b), nil
				})
				mfs.EXPECT().Remove(gomock.Eq(clientCSRPath)).Times(1).Return(os.ErrPermission)
			},
		},
		{
			name:    "success, but deleting previous certificate fails",
			wantErr: true,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				f := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().OpenFile(gomock.Eq(clientKeyPath), gomock.Eq(os.O_CREATE|os.O_TRUNC|os.O_WRONLY), gomock.Eq(fs.FileMode(0644))).Times(1).Return(f, nil)
				f.EXPECT().Close().Times(1).Return(nil)
				f.EXPECT().Write(gomock.Any()).Times(1).DoAndReturn(func(b []byte) (int, error) {
					return len(b), nil
				})
				mfs.EXPECT().Remove(gomock.Eq(clientCSRPath)).Times(1).Return(os.ErrNotExist)
				mfs.EXPECT().Remove(gomock.Eq(clientCertPath)).Times(1).Return(os.ErrPermission)
			},
		},
		{
			name:        "generating private key fails",
			wantErr:     true,
			wantErrToBe: errGeneratePrivateKey,
			ecdsaGenerateKey: func(c elliptic.Curve, rand io.Reader) (*ecdsa.PrivateKey, error) {
				return nil, errGeneratePrivateKey
			},
		},
		{
			name:        "DER encoding of private key fails",
			wantErr:     true,
			wantErrToBe: errMarshalPrivateKey,
			x509MarshalECPrivateKey: func(key *ecdsa.PrivateKey) ([]byte, error) {
				return nil, errMarshalPrivateKey
			},
		},
		{
			name:        "opening key file fails",
			wantErr:     true,
			wantErrToBe: os.ErrPermission,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				mfs.EXPECT().OpenFile(gomock.Eq(clientKeyPath), gomock.Eq(os.O_CREATE|os.O_TRUNC|os.O_WRONLY), gomock.Eq(fs.FileMode(0644))).Times(1).Return(nil, os.ErrPermission)
			},
		},
		{
			name:        "writing to key file fails",
			wantErr:     true,
			wantErrToBe: io.ErrUnexpectedEOF,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				f := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().OpenFile(gomock.Eq(clientKeyPath), gomock.Eq(os.O_CREATE|os.O_TRUNC|os.O_WRONLY), gomock.Eq(fs.FileMode(0644))).Times(1).Return(f, nil)
				f.EXPECT().Close().Times(1).Return(nil)
				f.EXPECT().Write(gomock.Any()).Times(1).DoAndReturn(func(b []byte) (int, error) {
					return 0, io.ErrUnexpectedEOF
				})
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockfs := mockpartitions.NewMockFS(ctrl)
			a := &api{
				dev: &partitions.Device{
					Uevent: partitions.Uevent{
						partitions.UeventDevtype: partitions.UeventDevtypePartition,
					},
					GPTPartType: partitions.GPTPartTypeHedgehogIdentity,
					FS:          mockfs,
				},
			}
			if tt.ecdsaGenerateKey != nil {
				oldEcdsaGenerateKey := ecdsaGenerateKey
				defer func() {
					ecdsaGenerateKey = oldEcdsaGenerateKey
				}()
				ecdsaGenerateKey = tt.ecdsaGenerateKey
			}
			if tt.x509MarshalECPrivateKey != nil {
				oldX509MarshalECPrivateKey := x509MarshalECPrivateKey
				defer func() {
					x509MarshalECPrivateKey = oldX509MarshalECPrivateKey
				}()
				x509MarshalECPrivateKey = tt.x509MarshalECPrivateKey
			}
			if tt.pre != nil {
				tt.pre(t, ctrl, mockfs)
			}
			err := a.GenerateClientKeyPair()
			if (err != nil) != tt.wantErr {
				t.Errorf("api.GenerateClientKeyPair() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr && tt.wantErrToBe != nil {
				if !errors.Is(err, tt.wantErrToBe) {
					t.Errorf("Open() error = %v, wantErrToBe %v", err, tt.wantErrToBe)
					return
				}
			}
		})
	}
}

func Test_api_GetLocation(t *testing.T) {
	tests := []struct {
		name        string
		want        *location.Info
		wantErr     bool
		wantErrToBe error
		pre         func(t *testing.T, ctrl *gomock.Controller, fs *mockpartitions.MockFS)
	}{
		{
			name: "success",
			pre: func(t *testing.T, ctrl *gomock.Controller, fs *mockpartitions.MockFS) {
				f1 := mockio.NewMockReadWriteCloser(ctrl)
				f1.EXPECT().Read(gomock.Any()).Times(1).DoAndReturn(func(b []byte) (int, error) {
					ret := []byte("2a59c9f4-9966-4270-b6a2-2313f41d5ce1")
					copy(b, ret)
					return len(ret), io.EOF
				})
				f1.EXPECT().Close().Times(1)
				f2 := mockio.NewMockReadWriteCloser(ctrl)
				f2.EXPECT().Read(gomock.Any()).Times(1).DoAndReturn(func(b []byte) (int, error) {
					ret := []byte("uuid-sig")
					copy(b, ret)
					return len(ret), io.EOF
				})
				f2.EXPECT().Close().Times(1)
				f3 := mockio.NewMockReadWriteCloser(ctrl)
				f3.EXPECT().Read(gomock.Any()).Times(1).DoAndReturn(func(b []byte) (int, error) {
					ret := []byte(`{"a":"aa","b":"bb"}`)
					copy(b, ret)
					return len(ret), io.EOF
				})
				f3.EXPECT().Close().Times(1)
				f4 := mockio.NewMockReadWriteCloser(ctrl)
				f4.EXPECT().Read(gomock.Any()).Times(1).DoAndReturn(func(b []byte) (int, error) {
					ret := []byte("metadata-sig")
					copy(b, ret)
					return len(ret), io.EOF
				})
				f4.EXPECT().Close().Times(1)
				fs.EXPECT().Open(gomock.Eq(locationUUIDPath)).Times(1).Return(f1, nil)
				fs.EXPECT().Open(gomock.Eq(locationUUIDSigPath)).Times(1).Return(f2, nil)
				fs.EXPECT().Open(gomock.Eq(locationMetadataPath)).Times(1).Return(f3, nil)
				fs.EXPECT().Open(gomock.Eq(locationMetadataSigPath)).Times(1).Return(f4, nil)
			},
			want: &location.Info{
				UUID:        "2a59c9f4-9966-4270-b6a2-2313f41d5ce1",
				UUIDSig:     []byte("uuid-sig"),
				Metadata:    `{"a":"aa","b":"bb"}`,
				MetadataSig: []byte("metadata-sig"),
			},
		},
		{
			name: "f1 open failure",
			pre: func(t *testing.T, ctrl *gomock.Controller, fs *mockpartitions.MockFS) {
				fs.EXPECT().Open(gomock.Eq(locationUUIDPath)).Times(1).Return(nil, os.ErrNotExist)
			},
			wantErr:     true,
			wantErrToBe: os.ErrNotExist,
		},
		{
			name: "f1 reading failure",
			pre: func(t *testing.T, ctrl *gomock.Controller, fs *mockpartitions.MockFS) {
				f1 := mockio.NewMockReadWriteCloser(ctrl)
				f1.EXPECT().Read(gomock.Any()).Times(1).Return(0, io.ErrUnexpectedEOF)
				f1.EXPECT().Close().Times(1)
				fs.EXPECT().Open(gomock.Eq(locationUUIDPath)).Times(1).Return(f1, nil)
			},
			wantErr:     true,
			wantErrToBe: io.ErrUnexpectedEOF,
		},
		{
			name: "f1 returns invalid uuid",
			pre: func(t *testing.T, ctrl *gomock.Controller, fs *mockpartitions.MockFS) {
				f1 := mockio.NewMockReadWriteCloser(ctrl)
				f1.EXPECT().Read(gomock.Any()).Times(1).DoAndReturn(func(b []byte) (int, error) {
					ret := []byte("invalid uuid")
					copy(b, ret)
					return len(ret), io.EOF
				})
				f1.EXPECT().Close().Times(1)
				fs.EXPECT().Open(gomock.Eq(locationUUIDPath)).Times(1).Return(f1, nil)
			},
			wantErr: true,
		},
		{
			name: "f2 open failure",
			pre: func(t *testing.T, ctrl *gomock.Controller, fs *mockpartitions.MockFS) {
				f1 := mockio.NewMockReadWriteCloser(ctrl)
				f1.EXPECT().Read(gomock.Any()).Times(1).DoAndReturn(func(b []byte) (int, error) {
					ret := []byte("2a59c9f4-9966-4270-b6a2-2313f41d5ce1")
					copy(b, ret)
					return len(ret), io.EOF
				})
				f1.EXPECT().Close().Times(1)
				fs.EXPECT().Open(gomock.Eq(locationUUIDPath)).Times(1).Return(f1, nil)
				fs.EXPECT().Open(gomock.Eq(locationUUIDSigPath)).Times(1).Return(nil, os.ErrNotExist)
			},
			wantErr:     true,
			wantErrToBe: os.ErrNotExist,
		},
		{
			name: "f2 reading failure",
			pre: func(t *testing.T, ctrl *gomock.Controller, fs *mockpartitions.MockFS) {
				f1 := mockio.NewMockReadWriteCloser(ctrl)
				f1.EXPECT().Read(gomock.Any()).Times(1).DoAndReturn(func(b []byte) (int, error) {
					ret := []byte("2a59c9f4-9966-4270-b6a2-2313f41d5ce1")
					copy(b, ret)
					return len(ret), io.EOF
				})
				f1.EXPECT().Close().Times(1)
				f2 := mockio.NewMockReadWriteCloser(ctrl)
				f2.EXPECT().Read(gomock.Any()).Times(1).Return(0, io.ErrUnexpectedEOF)
				f2.EXPECT().Close().Times(1)
				fs.EXPECT().Open(gomock.Eq(locationUUIDPath)).Times(1).Return(f1, nil)
				fs.EXPECT().Open(gomock.Eq(locationUUIDSigPath)).Times(1).Return(f2, nil)
			},
			wantErr:     true,
			wantErrToBe: io.ErrUnexpectedEOF,
		},
		{
			name: "f3 open failure",
			pre: func(t *testing.T, ctrl *gomock.Controller, fs *mockpartitions.MockFS) {
				f1 := mockio.NewMockReadWriteCloser(ctrl)
				f1.EXPECT().Read(gomock.Any()).Times(1).DoAndReturn(func(b []byte) (int, error) {
					ret := []byte("2a59c9f4-9966-4270-b6a2-2313f41d5ce1")
					copy(b, ret)
					return len(ret), io.EOF
				})
				f1.EXPECT().Close().Times(1)
				f2 := mockio.NewMockReadWriteCloser(ctrl)
				f2.EXPECT().Read(gomock.Any()).Times(1).DoAndReturn(func(b []byte) (int, error) {
					ret := []byte("uuid-sig")
					copy(b, ret)
					return len(ret), io.EOF
				})
				f2.EXPECT().Close().Times(1)
				fs.EXPECT().Open(gomock.Eq(locationUUIDPath)).Times(1).Return(f1, nil)
				fs.EXPECT().Open(gomock.Eq(locationUUIDSigPath)).Times(1).Return(f2, nil)
				fs.EXPECT().Open(gomock.Eq(locationMetadataPath)).Times(1).Return(nil, os.ErrNotExist)
			},
			wantErr:     true,
			wantErrToBe: os.ErrNotExist,
		},
		{
			name: "f3 reading failure",
			pre: func(t *testing.T, ctrl *gomock.Controller, fs *mockpartitions.MockFS) {
				f1 := mockio.NewMockReadWriteCloser(ctrl)
				f1.EXPECT().Read(gomock.Any()).Times(1).DoAndReturn(func(b []byte) (int, error) {
					ret := []byte("2a59c9f4-9966-4270-b6a2-2313f41d5ce1")
					copy(b, ret)
					return len(ret), io.EOF
				})
				f1.EXPECT().Close().Times(1)
				f2 := mockio.NewMockReadWriteCloser(ctrl)
				f2.EXPECT().Read(gomock.Any()).Times(1).DoAndReturn(func(b []byte) (int, error) {
					ret := []byte("uuid-sig")
					copy(b, ret)
					return len(ret), io.EOF
				})
				f2.EXPECT().Close().Times(1)
				f3 := mockio.NewMockReadWriteCloser(ctrl)
				f3.EXPECT().Read(gomock.Any()).Times(1).Return(0, io.ErrUnexpectedEOF)
				f3.EXPECT().Close().Times(1)
				fs.EXPECT().Open(gomock.Eq(locationUUIDPath)).Times(1).Return(f1, nil)
				fs.EXPECT().Open(gomock.Eq(locationUUIDSigPath)).Times(1).Return(f2, nil)
				fs.EXPECT().Open(gomock.Eq(locationMetadataPath)).Times(1).Return(f3, nil)
			},
			wantErr:     true,
			wantErrToBe: io.ErrUnexpectedEOF,
		},
		{
			name: "f3 returns invalid JSON",
			pre: func(t *testing.T, ctrl *gomock.Controller, fs *mockpartitions.MockFS) {
				f1 := mockio.NewMockReadWriteCloser(ctrl)
				f1.EXPECT().Read(gomock.Any()).Times(1).DoAndReturn(func(b []byte) (int, error) {
					ret := []byte("2a59c9f4-9966-4270-b6a2-2313f41d5ce1")
					copy(b, ret)
					return len(ret), io.EOF
				})
				f1.EXPECT().Close().Times(1)
				f2 := mockio.NewMockReadWriteCloser(ctrl)
				f2.EXPECT().Read(gomock.Any()).Times(1).DoAndReturn(func(b []byte) (int, error) {
					ret := []byte("uuid-sig")
					copy(b, ret)
					return len(ret), io.EOF
				})
				f2.EXPECT().Close().Times(1)
				f3 := mockio.NewMockReadWriteCloser(ctrl)
				f3.EXPECT().Read(gomock.Any()).Times(1).DoAndReturn(func(b []byte) (int, error) {
					ret := []byte(`{"a":"aa","b":`)
					copy(b, ret)
					return len(ret), io.EOF
				})
				f3.EXPECT().Close().Times(1)
				fs.EXPECT().Open(gomock.Eq(locationUUIDPath)).Times(1).Return(f1, nil)
				fs.EXPECT().Open(gomock.Eq(locationUUIDSigPath)).Times(1).Return(f2, nil)
				fs.EXPECT().Open(gomock.Eq(locationMetadataPath)).Times(1).Return(f3, nil)
			},
			wantErr: true,
		},
		{
			name: "f4 open failure",
			pre: func(t *testing.T, ctrl *gomock.Controller, fs *mockpartitions.MockFS) {
				f1 := mockio.NewMockReadWriteCloser(ctrl)
				f1.EXPECT().Read(gomock.Any()).Times(1).DoAndReturn(func(b []byte) (int, error) {
					ret := []byte("2a59c9f4-9966-4270-b6a2-2313f41d5ce1")
					copy(b, ret)
					return len(ret), io.EOF
				})
				f1.EXPECT().Close().Times(1)
				f2 := mockio.NewMockReadWriteCloser(ctrl)
				f2.EXPECT().Read(gomock.Any()).Times(1).DoAndReturn(func(b []byte) (int, error) {
					ret := []byte("uuid-sig")
					copy(b, ret)
					return len(ret), io.EOF
				})
				f2.EXPECT().Close().Times(1)
				f3 := mockio.NewMockReadWriteCloser(ctrl)
				f3.EXPECT().Read(gomock.Any()).Times(1).DoAndReturn(func(b []byte) (int, error) {
					ret := []byte(`{"a":"aa","b":"bb"}`)
					copy(b, ret)
					return len(ret), io.EOF
				})
				f3.EXPECT().Close().Times(1)
				fs.EXPECT().Open(gomock.Eq(locationUUIDPath)).Times(1).Return(f1, nil)
				fs.EXPECT().Open(gomock.Eq(locationUUIDSigPath)).Times(1).Return(f2, nil)
				fs.EXPECT().Open(gomock.Eq(locationMetadataPath)).Times(1).Return(f3, nil)
				fs.EXPECT().Open(gomock.Eq(locationMetadataSigPath)).Times(1).Return(nil, os.ErrNotExist)
			},
			wantErr:     true,
			wantErrToBe: os.ErrNotExist,
		},
		{
			name: "f4 reading failure",
			pre: func(t *testing.T, ctrl *gomock.Controller, fs *mockpartitions.MockFS) {
				f1 := mockio.NewMockReadWriteCloser(ctrl)
				f1.EXPECT().Read(gomock.Any()).Times(1).DoAndReturn(func(b []byte) (int, error) {
					ret := []byte("2a59c9f4-9966-4270-b6a2-2313f41d5ce1")
					copy(b, ret)
					return len(ret), io.EOF
				})
				f1.EXPECT().Close().Times(1)
				f2 := mockio.NewMockReadWriteCloser(ctrl)
				f2.EXPECT().Read(gomock.Any()).Times(1).DoAndReturn(func(b []byte) (int, error) {
					ret := []byte("uuid-sig")
					copy(b, ret)
					return len(ret), io.EOF
				})
				f2.EXPECT().Close().Times(1)
				f3 := mockio.NewMockReadWriteCloser(ctrl)
				f3.EXPECT().Read(gomock.Any()).Times(1).DoAndReturn(func(b []byte) (int, error) {
					ret := []byte(`{"a":"aa","b":"bb"}`)
					copy(b, ret)
					return len(ret), io.EOF
				})
				f3.EXPECT().Close().Times(1)
				f4 := mockio.NewMockReadWriteCloser(ctrl)
				f4.EXPECT().Read(gomock.Any()).Times(1).Return(0, io.ErrUnexpectedEOF)
				f4.EXPECT().Close().Times(1)
				fs.EXPECT().Open(gomock.Eq(locationUUIDPath)).Times(1).Return(f1, nil)
				fs.EXPECT().Open(gomock.Eq(locationUUIDSigPath)).Times(1).Return(f2, nil)
				fs.EXPECT().Open(gomock.Eq(locationMetadataPath)).Times(1).Return(f3, nil)
				fs.EXPECT().Open(gomock.Eq(locationMetadataSigPath)).Times(1).Return(f4, nil)
			},
			wantErr:     true,
			wantErrToBe: io.ErrUnexpectedEOF,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockfs := mockpartitions.NewMockFS(ctrl)
			a := &api{
				dev: &partitions.Device{
					Uevent: partitions.Uevent{
						partitions.UeventDevtype: partitions.UeventDevtypePartition,
					},
					GPTPartType: partitions.GPTPartTypeHedgehogIdentity,
					FS:          mockfs,
				},
			}
			if tt.pre != nil {
				tt.pre(t, ctrl, mockfs)
			}
			got, err := a.GetLocation()
			if (err != nil) != tt.wantErr {
				t.Errorf("api.GetLocation() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr && tt.wantErrToBe != nil {
				if !errors.Is(err, tt.wantErrToBe) {
					t.Errorf("Open() error = %v, wantErrToBe %v", err, tt.wantErrToBe)
					return
				}
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("api.GetLocation() = %v, want %v", got, tt.want)
				return
			}
		})
	}
}

func Test_api_StoreLocation(t *testing.T) {
	info := &location.Info{
		UUID:        "2a59c9f4-9966-4270-b6a2-2313f41d5ce1",
		UUIDSig:     []byte("uuid-sig"),
		Metadata:    `{"a":"aa","b":"bb"}`,
		MetadataSig: []byte("metadata-sig"),
	}
	errF1Open := errors.New("Open() failed tragically")
	errF1Write := errors.New("Write() failed tragically")
	errF2Open := errors.New("Open() failed tragically")
	errF2Write := errors.New("Write() failed tragically")
	errF3Open := errors.New("Open() failed tragically")
	errF3Write := errors.New("Write() failed tragically")
	errF4Open := errors.New("Open() failed tragically")
	errF4Write := errors.New("Write() failed tragically")
	type args struct {
		info *location.Info
	}
	tests := []struct {
		name        string
		args        args
		wantErr     bool
		wantErrToBe error
		pre         func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS)
	}{
		{
			name:    "success",
			args:    args{info: info},
			wantErr: false,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				// uuid
				f1 := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().OpenFile(gomock.Eq(locationUUIDPath), gomock.Eq(os.O_CREATE|os.O_TRUNC|os.O_WRONLY), gomock.Eq(fs.FileMode(0644))).Times(1).Return(f1, nil)
				f1.EXPECT().Write(gomock.Eq([]byte("2a59c9f4-9966-4270-b6a2-2313f41d5ce1"))).Times(1).DoAndReturn(func(b []byte) (int, error) {
					return len(b), nil
				})
				f1.EXPECT().Close().Times(1).Return(nil)

				// uuid.sig
				f2 := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().OpenFile(gomock.Eq(locationUUIDSigPath), gomock.Eq(os.O_CREATE|os.O_TRUNC|os.O_WRONLY), gomock.Eq(fs.FileMode(0644))).Times(1).Return(f2, nil)
				f2.EXPECT().Write(gomock.Eq([]byte("uuid-sig"))).Times(1).DoAndReturn(func(b []byte) (int, error) {
					return len(b), nil
				})
				f2.EXPECT().Close().Times(1).Return(nil)

				// metadata
				f3 := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().OpenFile(gomock.Eq(locationMetadataPath), gomock.Eq(os.O_CREATE|os.O_TRUNC|os.O_WRONLY), gomock.Eq(fs.FileMode(0644))).Times(1).Return(f3, nil)
				f3.EXPECT().Write(gomock.Eq([]byte(`{"a":"aa","b":"bb"}`))).Times(1).DoAndReturn(func(b []byte) (int, error) {
					return len(b), nil
				})
				f3.EXPECT().Close().Times(1).Return(nil)

				// metadata.sig
				f4 := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().OpenFile(gomock.Eq(locationMetadataSigPath), gomock.Eq(os.O_CREATE|os.O_TRUNC|os.O_WRONLY), gomock.Eq(fs.FileMode(0644))).Times(1).Return(f4, nil)
				f4.EXPECT().Write(gomock.Eq([]byte("metadata-sig"))).Times(1).DoAndReturn(func(b []byte) (int, error) {
					return len(b), nil
				})
				f4.EXPECT().Close().Times(1).Return(nil)
			},
		},
		{
			name:        "f4 fails writing",
			args:        args{info: info},
			wantErr:     true,
			wantErrToBe: errF4Write,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				// uuid
				f1 := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().OpenFile(gomock.Eq(locationUUIDPath), gomock.Eq(os.O_CREATE|os.O_TRUNC|os.O_WRONLY), gomock.Eq(fs.FileMode(0644))).Times(1).Return(f1, nil)
				f1.EXPECT().Write(gomock.Eq([]byte("2a59c9f4-9966-4270-b6a2-2313f41d5ce1"))).Times(1).DoAndReturn(func(b []byte) (int, error) {
					return len(b), nil
				})
				f1.EXPECT().Close().Times(1).Return(nil)

				// uuid.sig
				f2 := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().OpenFile(gomock.Eq(locationUUIDSigPath), gomock.Eq(os.O_CREATE|os.O_TRUNC|os.O_WRONLY), gomock.Eq(fs.FileMode(0644))).Times(1).Return(f2, nil)
				f2.EXPECT().Write(gomock.Eq([]byte("uuid-sig"))).Times(1).DoAndReturn(func(b []byte) (int, error) {
					return len(b), nil
				})
				f2.EXPECT().Close().Times(1).Return(nil)

				// metadata
				f3 := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().OpenFile(gomock.Eq(locationMetadataPath), gomock.Eq(os.O_CREATE|os.O_TRUNC|os.O_WRONLY), gomock.Eq(fs.FileMode(0644))).Times(1).Return(f3, nil)
				f3.EXPECT().Write(gomock.Eq([]byte(`{"a":"aa","b":"bb"}`))).Times(1).DoAndReturn(func(b []byte) (int, error) {
					return len(b), nil
				})
				f3.EXPECT().Close().Times(1).Return(nil)

				// metadata.sig
				f4 := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().OpenFile(gomock.Eq(locationMetadataSigPath), gomock.Eq(os.O_CREATE|os.O_TRUNC|os.O_WRONLY), gomock.Eq(fs.FileMode(0644))).Times(1).Return(f4, nil)
				f4.EXPECT().Write(gomock.Eq([]byte("metadata-sig"))).Times(1).Return(0, errF4Write)
				f4.EXPECT().Close().Times(1).Return(nil)
			},
		},
		{
			name:        "f4 fails open",
			args:        args{info: info},
			wantErr:     true,
			wantErrToBe: errF4Open,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				// uuid
				f1 := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().OpenFile(gomock.Eq(locationUUIDPath), gomock.Eq(os.O_CREATE|os.O_TRUNC|os.O_WRONLY), gomock.Eq(fs.FileMode(0644))).Times(1).Return(f1, nil)
				f1.EXPECT().Write(gomock.Eq([]byte("2a59c9f4-9966-4270-b6a2-2313f41d5ce1"))).Times(1).DoAndReturn(func(b []byte) (int, error) {
					return len(b), nil
				})
				f1.EXPECT().Close().Times(1).Return(nil)

				// uuid.sig
				f2 := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().OpenFile(gomock.Eq(locationUUIDSigPath), gomock.Eq(os.O_CREATE|os.O_TRUNC|os.O_WRONLY), gomock.Eq(fs.FileMode(0644))).Times(1).Return(f2, nil)
				f2.EXPECT().Write(gomock.Eq([]byte("uuid-sig"))).Times(1).DoAndReturn(func(b []byte) (int, error) {
					return len(b), nil
				})
				f2.EXPECT().Close().Times(1).Return(nil)

				// metadata
				f3 := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().OpenFile(gomock.Eq(locationMetadataPath), gomock.Eq(os.O_CREATE|os.O_TRUNC|os.O_WRONLY), gomock.Eq(fs.FileMode(0644))).Times(1).Return(f3, nil)
				f3.EXPECT().Write(gomock.Eq([]byte(`{"a":"aa","b":"bb"}`))).Times(1).DoAndReturn(func(b []byte) (int, error) {
					return len(b), nil
				})
				f3.EXPECT().Close().Times(1).Return(nil)

				// metadata.sig
				mfs.EXPECT().OpenFile(gomock.Eq(locationMetadataSigPath), gomock.Eq(os.O_CREATE|os.O_TRUNC|os.O_WRONLY), gomock.Eq(fs.FileMode(0644))).Times(1).Return(nil, errF4Open)
			},
		},
		{
			name:        "f3 fails writing",
			args:        args{info: info},
			wantErr:     true,
			wantErrToBe: errF3Write,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				// uuid
				f1 := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().OpenFile(gomock.Eq(locationUUIDPath), gomock.Eq(os.O_CREATE|os.O_TRUNC|os.O_WRONLY), gomock.Eq(fs.FileMode(0644))).Times(1).Return(f1, nil)
				f1.EXPECT().Write(gomock.Eq([]byte("2a59c9f4-9966-4270-b6a2-2313f41d5ce1"))).Times(1).DoAndReturn(func(b []byte) (int, error) {
					return len(b), nil
				})
				f1.EXPECT().Close().Times(1).Return(nil)

				// uuid.sig
				f2 := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().OpenFile(gomock.Eq(locationUUIDSigPath), gomock.Eq(os.O_CREATE|os.O_TRUNC|os.O_WRONLY), gomock.Eq(fs.FileMode(0644))).Times(1).Return(f2, nil)
				f2.EXPECT().Write(gomock.Eq([]byte("uuid-sig"))).Times(1).DoAndReturn(func(b []byte) (int, error) {
					return len(b), nil
				})
				f2.EXPECT().Close().Times(1).Return(nil)

				// metadata
				f3 := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().OpenFile(gomock.Eq(locationMetadataPath), gomock.Eq(os.O_CREATE|os.O_TRUNC|os.O_WRONLY), gomock.Eq(fs.FileMode(0644))).Times(1).Return(f3, nil)
				f3.EXPECT().Write(gomock.Eq([]byte(`{"a":"aa","b":"bb"}`))).Times(1).Return(0, errF3Write)
				f3.EXPECT().Close().Times(1).Return(nil)
			},
		},
		{
			name:        "f3 fails open",
			args:        args{info: info},
			wantErr:     true,
			wantErrToBe: errF3Open,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				// uuid
				f1 := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().OpenFile(gomock.Eq(locationUUIDPath), gomock.Eq(os.O_CREATE|os.O_TRUNC|os.O_WRONLY), gomock.Eq(fs.FileMode(0644))).Times(1).Return(f1, nil)
				f1.EXPECT().Write(gomock.Eq([]byte("2a59c9f4-9966-4270-b6a2-2313f41d5ce1"))).Times(1).DoAndReturn(func(b []byte) (int, error) {
					return len(b), nil
				})
				f1.EXPECT().Close().Times(1).Return(nil)

				// uuid.sig
				f2 := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().OpenFile(gomock.Eq(locationUUIDSigPath), gomock.Eq(os.O_CREATE|os.O_TRUNC|os.O_WRONLY), gomock.Eq(fs.FileMode(0644))).Times(1).Return(f2, nil)
				f2.EXPECT().Write(gomock.Eq([]byte("uuid-sig"))).Times(1).DoAndReturn(func(b []byte) (int, error) {
					return len(b), nil
				})
				f2.EXPECT().Close().Times(1).Return(nil)

				// metadata
				mfs.EXPECT().OpenFile(gomock.Eq(locationMetadataPath), gomock.Eq(os.O_CREATE|os.O_TRUNC|os.O_WRONLY), gomock.Eq(fs.FileMode(0644))).Times(1).Return(nil, errF3Open)
			},
		},
		{
			name:        "f2 fails writing",
			args:        args{info: info},
			wantErr:     true,
			wantErrToBe: errF2Write,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				// uuid
				f1 := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().OpenFile(gomock.Eq(locationUUIDPath), gomock.Eq(os.O_CREATE|os.O_TRUNC|os.O_WRONLY), gomock.Eq(fs.FileMode(0644))).Times(1).Return(f1, nil)
				f1.EXPECT().Write(gomock.Eq([]byte("2a59c9f4-9966-4270-b6a2-2313f41d5ce1"))).Times(1).DoAndReturn(func(b []byte) (int, error) {
					return len(b), nil
				})
				f1.EXPECT().Close().Times(1).Return(nil)

				// uuid.sig
				f2 := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().OpenFile(gomock.Eq(locationUUIDSigPath), gomock.Eq(os.O_CREATE|os.O_TRUNC|os.O_WRONLY), gomock.Eq(fs.FileMode(0644))).Times(1).Return(f2, nil)
				f2.EXPECT().Write(gomock.Eq([]byte("uuid-sig"))).Times(1).Return(0, errF2Write)
				f2.EXPECT().Close().Times(1).Return(nil)
			},
		},
		{
			name:        "f2 fails open",
			args:        args{info: info},
			wantErr:     true,
			wantErrToBe: errF2Open,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				// uuid
				f1 := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().OpenFile(gomock.Eq(locationUUIDPath), gomock.Eq(os.O_CREATE|os.O_TRUNC|os.O_WRONLY), gomock.Eq(fs.FileMode(0644))).Times(1).Return(f1, nil)
				f1.EXPECT().Write(gomock.Eq([]byte("2a59c9f4-9966-4270-b6a2-2313f41d5ce1"))).Times(1).DoAndReturn(func(b []byte) (int, error) {
					return len(b), nil
				})
				f1.EXPECT().Close().Times(1).Return(nil)

				// uuid.sig
				mfs.EXPECT().OpenFile(gomock.Eq(locationUUIDSigPath), gomock.Eq(os.O_CREATE|os.O_TRUNC|os.O_WRONLY), gomock.Eq(fs.FileMode(0644))).Times(1).Return(nil, errF2Open)
			},
		},
		{
			name:        "f1 fails writing",
			args:        args{info: info},
			wantErr:     true,
			wantErrToBe: errF1Write,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				// uuid
				f1 := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().OpenFile(gomock.Eq(locationUUIDPath), gomock.Eq(os.O_CREATE|os.O_TRUNC|os.O_WRONLY), gomock.Eq(fs.FileMode(0644))).Times(1).Return(f1, nil)
				f1.EXPECT().Write(gomock.Eq([]byte("2a59c9f4-9966-4270-b6a2-2313f41d5ce1"))).Times(1).Return(0, errF1Write)
				f1.EXPECT().Close().Times(1).Return(nil)
			},
		},
		{
			name:        "f1 fails open",
			args:        args{info: info},
			wantErr:     true,
			wantErrToBe: errF1Open,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				// uuid
				mfs.EXPECT().OpenFile(gomock.Eq(locationUUIDPath), gomock.Eq(os.O_CREATE|os.O_TRUNC|os.O_WRONLY), gomock.Eq(fs.FileMode(0644))).Times(1).Return(nil, errF1Open)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockfs := mockpartitions.NewMockFS(ctrl)
			a := &api{
				dev: &partitions.Device{
					Uevent: partitions.Uevent{
						partitions.UeventDevtype: partitions.UeventDevtypePartition,
					},
					GPTPartType: partitions.GPTPartTypeHedgehogIdentity,
					FS:          mockfs,
				},
			}
			if tt.pre != nil {
				tt.pre(t, ctrl, mockfs)
			}
			err := a.StoreLocation(tt.args.info)
			if (err != nil) != tt.wantErr {
				t.Errorf("api.StoreLocation() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr && tt.wantErrToBe != nil {
				if !errors.Is(err, tt.wantErrToBe) {
					t.Errorf("Open() error = %v, wantErrToBe %v", err, tt.wantErrToBe)
					return
				}
			}
		})
	}
}

func Test_api_CopyLocation(t *testing.T) {
	errFailure := errors.New("GetLocation() failed tragically")
	tests := []struct {
		name        string
		wantErr     bool
		wantErrToBe error
		pre         func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS, mlp *mocklocation.MockLocationPartition)
	}{
		{
			name:    "success",
			wantErr: false,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS, mlp *mocklocation.MockLocationPartition) {
				mlp.EXPECT().GetLocation().Times(1).Return(&location.Info{
					UUID:        "2a59c9f4-9966-4270-b6a2-2313f41d5ce1",
					UUIDSig:     []byte("uuid-sig"),
					Metadata:    `{"a":"aa","b":"bb"}`,
					MetadataSig: []byte("metadata-sig"),
				}, nil)
				// uuid
				f1 := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().OpenFile(gomock.Eq(locationUUIDPath), gomock.Eq(os.O_CREATE|os.O_TRUNC|os.O_WRONLY), gomock.Eq(fs.FileMode(0644))).Times(1).Return(f1, nil)
				f1.EXPECT().Write(gomock.Eq([]byte("2a59c9f4-9966-4270-b6a2-2313f41d5ce1"))).Times(1).DoAndReturn(func(b []byte) (int, error) {
					return len(b), nil
				})
				f1.EXPECT().Close().Times(1).Return(nil)

				// uuid.sig
				f2 := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().OpenFile(gomock.Eq(locationUUIDSigPath), gomock.Eq(os.O_CREATE|os.O_TRUNC|os.O_WRONLY), gomock.Eq(fs.FileMode(0644))).Times(1).Return(f2, nil)
				f2.EXPECT().Write(gomock.Eq([]byte("uuid-sig"))).Times(1).DoAndReturn(func(b []byte) (int, error) {
					return len(b), nil
				})
				f2.EXPECT().Close().Times(1).Return(nil)

				// metadata
				f3 := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().OpenFile(gomock.Eq(locationMetadataPath), gomock.Eq(os.O_CREATE|os.O_TRUNC|os.O_WRONLY), gomock.Eq(fs.FileMode(0644))).Times(1).Return(f3, nil)
				f3.EXPECT().Write(gomock.Eq([]byte(`{"a":"aa","b":"bb"}`))).Times(1).DoAndReturn(func(b []byte) (int, error) {
					return len(b), nil
				})
				f3.EXPECT().Close().Times(1).Return(nil)

				// metadata.sig
				f4 := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().OpenFile(gomock.Eq(locationMetadataSigPath), gomock.Eq(os.O_CREATE|os.O_TRUNC|os.O_WRONLY), gomock.Eq(fs.FileMode(0644))).Times(1).Return(f4, nil)
				f4.EXPECT().Write(gomock.Eq([]byte("metadata-sig"))).Times(1).DoAndReturn(func(b []byte) (int, error) {
					return len(b), nil
				})
				f4.EXPECT().Close().Times(1).Return(nil)
			},
		},
		{
			name:        "failure",
			wantErr:     true,
			wantErrToBe: errFailure,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS, mlp *mocklocation.MockLocationPartition) {
				mlp.EXPECT().GetLocation().Times(1).Return(nil, errFailure)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockfsTarget := mockpartitions.NewMockFS(ctrl)
			a := &api{
				dev: &partitions.Device{
					Uevent: partitions.Uevent{
						partitions.UeventDevtype: partitions.UeventDevtypePartition,
					},
					GPTPartType: partitions.GPTPartTypeHedgehogIdentity,
					FS:          mockfsTarget,
				},
			}
			mocklp := mocklocation.NewMockLocationPartition(ctrl)
			if tt.pre != nil {
				tt.pre(t, ctrl, mockfsTarget, mocklp)
			}
			err := a.CopyLocation(mocklp)
			if (err != nil) != tt.wantErr {
				t.Errorf("api.CopyLocation() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr && tt.wantErrToBe != nil {
				if !errors.Is(err, tt.wantErrToBe) {
					t.Errorf("Open() error = %v, wantErrToBe %v", err, tt.wantErrToBe)
					return
				}
			}
		})
	}
}

func readFile(path string) []byte {
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	file, err := os.Open(filepath.Join(pwd, "testdata", path))
	if err != nil {
		panic(err)
	}
	b, err := io.ReadAll(file)
	if err != nil {
		panic(err)
	}
	return b
}

func Test_api_HasClientCSR(t *testing.T) {
	csrValid := readFile("csr-valid.pem")
	csrInvalid := readFile("csr-invalid.pem")
	csrNotACSR := readFile("csr-not-a-csr.pem")
	csrNotAPEM := readFile("csr-not-pem.pem")
	tests := []struct {
		name string
		want bool
		pre  func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS)
	}{
		{
			name: "success",
			want: true,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				f := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().Open(clientCSRPath).Times(1).Return(f, nil)
				mockio.ReadAllBytesMock(f, csrValid, 2)
				f.EXPECT().Close().Times(1).Return(nil)
			},
		},
		{
			name: "invalid CSR",
			want: false,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				f := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().Open(clientCSRPath).Times(1).Return(f, nil)
				mockio.ReadAllBytesMock(f, csrInvalid, 1)
				f.EXPECT().Close().Times(1).Return(nil)
			},
		},
		{
			name: "PEM not a CSR",
			want: false,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				f := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().Open(clientCSRPath).Times(1).Return(f, nil)
				mockio.ReadAllBytesMock(f, csrNotACSR, 1)
				f.EXPECT().Close().Times(1).Return(nil)
			},
		},
		{
			name: "not a PEM file",
			want: false,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				f := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().Open(clientCSRPath).Times(1).Return(f, nil)
				mockio.ReadAllBytesMock(f, csrNotAPEM, 1)
				f.EXPECT().Close().Times(1).Return(nil)
			},
		},
		{
			name: "reading PEM file fails",
			want: false,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				f := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().Open(clientCSRPath).Times(1).Return(f, nil)
				f.EXPECT().Read(gomock.Any()).Times(1).Return(0, io.ErrUnexpectedEOF)
				f.EXPECT().Close().Times(1).Return(nil)
			},
		},
		{
			name: "opening PEM file fails",
			want: false,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				mfs.EXPECT().Open(clientCSRPath).Times(1).Return(nil, os.ErrNotExist)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockfs := mockpartitions.NewMockFS(ctrl)
			a := &api{
				dev: &partitions.Device{
					Uevent: partitions.Uevent{
						partitions.UeventDevtype: partitions.UeventDevtypePartition,
					},
					GPTPartType: partitions.GPTPartTypeHedgehogIdentity,
					FS:          mockfs,
				},
			}
			if tt.pre != nil {
				tt.pre(t, ctrl, mockfs)
			}
			if got := a.HasClientCSR(); got != tt.want {
				t.Errorf("api.HasClientCSR() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_api_HasClientCert(t *testing.T) {
	certValid := readFile("cert-valid.pem")
	certInvalid := readFile("cert-invalid.pem")
	certNotACSR := readFile("cert-not-a-cert.pem")
	certNotAPEM := readFile("cert-not-pem.pem")
	tests := []struct {
		name string
		want bool
		pre  func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS)
	}{
		{
			name: "success",
			want: true,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				f := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().Open(clientCertPath).Times(1).Return(f, nil)
				mockio.ReadAllBytesMock(f, certValid, 3)
				f.EXPECT().Close().Times(1).Return(nil)
			},
		},
		{
			name: "invalid certificate",
			want: false,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				f := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().Open(clientCertPath).Times(1).Return(f, nil)
				mockio.ReadAllBytesMock(f, certInvalid, 1)
				f.EXPECT().Close().Times(1).Return(nil)
			},
		},
		{
			name: "PEM not a certificate",
			want: false,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				f := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().Open(clientCertPath).Times(1).Return(f, nil)
				mockio.ReadAllBytesMock(f, certNotACSR, 2)
				f.EXPECT().Close().Times(1).Return(nil)
			},
		},
		{
			name: "not a PEM file",
			want: false,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				f := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().Open(clientCertPath).Times(1).Return(f, nil)
				mockio.ReadAllBytesMock(f, certNotAPEM, 1)
				f.EXPECT().Close().Times(1).Return(nil)
			},
		},
		{
			name: "reading PEM file fails",
			want: false,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				f := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().Open(clientCertPath).Times(1).Return(f, nil)
				f.EXPECT().Read(gomock.Any()).Times(1).Return(0, io.ErrUnexpectedEOF)
				f.EXPECT().Close().Times(1).Return(nil)
			},
		},
		{
			name: "opening PEM file fails",
			want: false,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				mfs.EXPECT().Open(clientCertPath).Times(1).Return(nil, os.ErrNotExist)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockfs := mockpartitions.NewMockFS(ctrl)
			a := &api{
				dev: &partitions.Device{
					Uevent: partitions.Uevent{
						partitions.UeventDevtype: partitions.UeventDevtypePartition,
					},
					GPTPartType: partitions.GPTPartTypeHedgehogIdentity,
					FS:          mockfs,
				},
			}
			if tt.pre != nil {
				tt.pre(t, ctrl, mockfs)
			}
			if got := a.HasClientCert(); got != tt.want {
				t.Errorf("api.HasClientCert() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_api_HasValidClientCert(t *testing.T) {
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	certValid := readFile("cert-valid.pem")
	certInvalid := readFile("cert-invalid.pem")
	certExpired := readFile("cert-expired.pem")
	certNotACSR := readFile("cert-not-a-cert.pem")
	certNotAPEM := readFile("cert-not-pem.pem")
	tests := []struct {
		name string
		want bool
		pre  func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS)
	}{
		{
			name: "success",
			want: true,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				f := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().Open(clientCertPath).Times(1).Return(f, nil)
				mockio.ReadAllBytesMock(f, certValid, 3)
				f.EXPECT().Close().Times(1).Return(nil)

				// for the cert loading check, we'll have some more calls
				mfs.EXPECT().Path(gomock.Eq(clientKeyPath)).Times(1).Return(filepath.Join(pwd, "testdata", "key-valid.pem"))
				mfs.EXPECT().Path(gomock.Eq(clientCertPath)).Times(1).Return(filepath.Join(pwd, "testdata", "cert-valid.pem"))
			},
		},
		{
			name: "invalid certificate",
			want: false,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				f := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().Open(clientCertPath).Times(1).Return(f, nil)
				mockio.ReadAllBytesMock(f, certInvalid, 1)
				f.EXPECT().Close().Times(1).Return(nil)
			},
		},
		{
			name: "expired certificate",
			want: false,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				f := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().Open(clientCertPath).Times(1).Return(f, nil)
				mockio.ReadAllBytesMock(f, certExpired, 3)
				f.EXPECT().Close().Times(1).Return(nil)
			},
		},
		{
			name: "PEM not a certificate",
			want: false,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				f := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().Open(clientCertPath).Times(1).Return(f, nil)
				mockio.ReadAllBytesMock(f, certNotACSR, 2)
				f.EXPECT().Close().Times(1).Return(nil)
			},
		},
		{
			name: "not a PEM file",
			want: false,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				f := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().Open(clientCertPath).Times(1).Return(f, nil)
				mockio.ReadAllBytesMock(f, certNotAPEM, 1)
				f.EXPECT().Close().Times(1).Return(nil)
			},
		},
		{
			name: "reading PEM file fails",
			want: false,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				f := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().Open(clientCertPath).Times(1).Return(f, nil)
				f.EXPECT().Read(gomock.Any()).Times(1).Return(0, io.ErrUnexpectedEOF)
				f.EXPECT().Close().Times(1).Return(nil)
			},
		},
		{
			name: "opening PEM file fails",
			want: false,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				mfs.EXPECT().Open(clientCertPath).Times(1).Return(nil, os.ErrNotExist)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockfs := mockpartitions.NewMockFS(ctrl)
			a := &api{
				dev: &partitions.Device{
					Uevent: partitions.Uevent{
						partitions.UeventDevtype: partitions.UeventDevtypePartition,
					},
					GPTPartType: partitions.GPTPartTypeHedgehogIdentity,
					FS:          mockfs,
				},
			}
			if tt.pre != nil {
				tt.pre(t, ctrl, mockfs)
			}
			if got := a.HasValidClientCert(); got != tt.want {
				t.Errorf("api.HasValidClientCert() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_api_MatchesClientCertificate(t *testing.T) {
	type args struct {
		cert *x509.Certificate
	}
	certValid := readFile("cert-valid.pem")
	p, _ := pem.Decode(certValid)
	if p == nil {
		panic("certValid is not a PEM file")
	}
	x509CertValid, err := x509.ParseCertificate(p.Bytes)
	if err != nil {
		panic(err)
	}
	certNoMatch := readFile("cert-expired.pem")
	p, _ = pem.Decode(certNoMatch)
	if p == nil {
		panic("certNoMatch is not a PEM file")
	}
	x509CertNoMatch, err := x509.ParseCertificate(p.Bytes)
	certInvalid := readFile("cert-invalid.pem")
	certNotACSR := readFile("cert-not-a-cert.pem")
	certNotAPEM := readFile("cert-not-pem.pem")
	tests := []struct {
		name string
		want bool
		args args
		pre  func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS)
	}{
		{
			name: "success",
			want: true,
			args: args{
				cert: x509CertValid,
			},
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				f := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().Open(clientCertPath).Times(1).Return(f, nil)
				mockio.ReadAllBytesMock(f, certValid, 3)
				f.EXPECT().Close().Times(1).Return(nil)
			},
		},
		{
			name: "certificates do not match",
			want: false,
			args: args{
				cert: x509CertNoMatch,
			},
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				f := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().Open(clientCertPath).Times(1).Return(f, nil)
				mockio.ReadAllBytesMock(f, certValid, 3)
				f.EXPECT().Close().Times(1).Return(nil)
			},
		},
		{
			name: "invalid certificate",
			want: false,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				f := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().Open(clientCertPath).Times(1).Return(f, nil)
				mockio.ReadAllBytesMock(f, certInvalid, 1)
				f.EXPECT().Close().Times(1).Return(nil)
			},
		},
		{
			name: "PEM not a certificate",
			want: false,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				f := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().Open(clientCertPath).Times(1).Return(f, nil)
				mockio.ReadAllBytesMock(f, certNotACSR, 2)
				f.EXPECT().Close().Times(1).Return(nil)
			},
		},
		{
			name: "not a PEM file",
			want: false,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				f := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().Open(clientCertPath).Times(1).Return(f, nil)
				mockio.ReadAllBytesMock(f, certNotAPEM, 1)
				f.EXPECT().Close().Times(1).Return(nil)
			},
		},
		{
			name: "reading PEM file fails",
			want: false,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				f := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().Open(clientCertPath).Times(1).Return(f, nil)
				f.EXPECT().Read(gomock.Any()).Times(1).Return(0, io.ErrUnexpectedEOF)
				f.EXPECT().Close().Times(1).Return(nil)
			},
		},
		{
			name: "opening PEM file fails",
			want: false,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				mfs.EXPECT().Open(clientCertPath).Times(1).Return(nil, os.ErrNotExist)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockfs := mockpartitions.NewMockFS(ctrl)
			a := &api{
				dev: &partitions.Device{
					Uevent: partitions.Uevent{
						partitions.UeventDevtype: partitions.UeventDevtypePartition,
					},
					GPTPartType: partitions.GPTPartTypeHedgehogIdentity,
					FS:          mockfs,
				},
			}
			if tt.pre != nil {
				tt.pre(t, ctrl, mockfs)
			}
			if got := a.MatchesClientCertificate(tt.args.cert); got != tt.want {
				t.Errorf("api.MatchesClientCertificate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_api_HasClientKey(t *testing.T) {
	keyValid := readFile("key-valid.pem")
	keyInvalid := readFile("key-invalid.pem")
	keyNotAKey := readFile("key-not-a-key.pem")
	keyNotAPEM := readFile("key-not-pem.pem")
	tests := []struct {
		name string
		want bool
		pre  func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS)
	}{
		{
			name: "success",
			want: true,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				f := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().Open(clientKeyPath).Times(1).Return(f, nil)
				mockio.ReadAllBytesMock(f, keyValid, 1)
				f.EXPECT().Close().Times(1).Return(nil)
			},
		},
		{
			name: "invalid key",
			want: false,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				f := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().Open(clientKeyPath).Times(1).Return(f, nil)
				mockio.ReadAllBytesMock(f, keyInvalid, 1)
				f.EXPECT().Close().Times(1).Return(nil)
			},
		},
		{
			name: "PEM not a key",
			want: false,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				f := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().Open(clientKeyPath).Times(1).Return(f, nil)
				mockio.ReadAllBytesMock(f, keyNotAKey, 2)
				f.EXPECT().Close().Times(1).Return(nil)
			},
		},
		{
			name: "not a PEM file",
			want: false,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				f := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().Open(clientKeyPath).Times(1).Return(f, nil)
				mockio.ReadAllBytesMock(f, keyNotAPEM, 1)
				f.EXPECT().Close().Times(1).Return(nil)
			},
		},
		{
			name: "reading PEM file fails",
			want: false,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				f := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().Open(clientKeyPath).Times(1).Return(f, nil)
				f.EXPECT().Read(gomock.Any()).Times(1).Return(0, io.ErrUnexpectedEOF)
				f.EXPECT().Close().Times(1).Return(nil)
			},
		},
		{
			name: "opening PEM file fails",
			want: false,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				mfs.EXPECT().Open(clientKeyPath).Times(1).Return(nil, os.ErrNotExist)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockfs := mockpartitions.NewMockFS(ctrl)
			a := &api{
				dev: &partitions.Device{
					Uevent: partitions.Uevent{
						partitions.UeventDevtype: partitions.UeventDevtypePartition,
					},
					GPTPartType: partitions.GPTPartTypeHedgehogIdentity,
					FS:          mockfs,
				},
			}
			if tt.pre != nil {
				tt.pre(t, ctrl, mockfs)
			}
			if got := a.HasClientKey(); got != tt.want {
				t.Errorf("api.HasClientKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_api_LoadX509KeyPair(t *testing.T) {
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	tests := []struct {
		name        string
		wantErr     bool
		wantErrToBe error
		pre         func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS)
	}{
		{
			name:    "files success",
			wantErr: false,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				mfs.EXPECT().Path(gomock.Eq(clientKeyPath)).Times(1).Return(filepath.Join(pwd, "testdata", "key-valid.pem"))
				mfs.EXPECT().Path(gomock.Eq(clientCertPath)).Times(1).Return(filepath.Join(pwd, "testdata", "cert-valid.pem"))
			},
		},
		{
			name:    "files failure",
			wantErr: true,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				mfs.EXPECT().Path(gomock.Eq(clientKeyPath)).Times(1).Return(filepath.Join(pwd, "testdata", "key-invalid.pem"))
				mfs.EXPECT().Path(gomock.Eq(clientCertPath)).Times(1).Return(filepath.Join(pwd, "testdata", "cert-invalid.pem"))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockfs := mockpartitions.NewMockFS(ctrl)
			a := &api{
				dev: &partitions.Device{
					Uevent: partitions.Uevent{
						partitions.UeventDevtype: partitions.UeventDevtypePartition,
					},
					GPTPartType: partitions.GPTPartTypeHedgehogIdentity,
					FS:          mockfs,
				},
			}
			if tt.pre != nil {
				tt.pre(t, ctrl, mockfs)
			}
			_, err := a.LoadX509KeyPair()
			if (err != nil) != tt.wantErr {
				t.Errorf("api.LoadX509KeyPair() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr && tt.wantErrToBe != nil {
				if !errors.Is(err, tt.wantErrToBe) {
					t.Errorf("Open() error = %v, wantErrToBe %v", err, tt.wantErrToBe)
					return
				}
			}
		})
	}
}

func Test_api_ReadClientCSR(t *testing.T) {
	csrValid := readFile("csr-valid.pem")
	csrInvalid := readFile("csr-invalid.pem")
	csrNotAPEM := readFile("csr-not-pem.pem")
	tests := []struct {
		name        string
		wantErr     bool
		wantErrToBe error
		pre         func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS)
	}{
		{
			name:    "success",
			wantErr: false,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				f := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().Open(clientCSRPath).Times(1).Return(f, nil)
				f.EXPECT().Close().Times(1).Return(nil)
				mockio.ReadAllBytesMock(f, csrValid, 2)
			},
		},
		{
			name:    "invalid CSR",
			wantErr: true,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				f := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().Open(clientCSRPath).Times(1).Return(f, nil)
				f.EXPECT().Close().Times(1).Return(nil)
				mockio.ReadAllBytesMock(f, csrInvalid, 1)
			},
		},
		{
			name:        "PEM decoding fails",
			wantErr:     true,
			wantErrToBe: ErrNoPEMData,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				f := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().Open(clientCSRPath).Times(1).Return(f, nil)
				f.EXPECT().Close().Times(1).Return(nil)
				mockio.ReadAllBytesMock(f, csrNotAPEM, 1)
			},
		},
		{
			name:        "reading CSR file fails",
			wantErr:     true,
			wantErrToBe: io.ErrUnexpectedEOF,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				f := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().Open(clientCSRPath).Times(1).Return(f, nil)
				f.EXPECT().Read(gomock.Any()).Times(1).Return(0, io.ErrUnexpectedEOF)
				f.EXPECT().Close().Times(1).Return(nil)
			},
		},
		{
			name:        "opening CSR file fails",
			wantErr:     true,
			wantErrToBe: os.ErrNotExist,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				mfs.EXPECT().Open(clientCSRPath).Times(1).Return(nil, os.ErrNotExist)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockfs := mockpartitions.NewMockFS(ctrl)
			a := &api{
				dev: &partitions.Device{
					Uevent: partitions.Uevent{
						partitions.UeventDevtype: partitions.UeventDevtypePartition,
					},
					GPTPartType: partitions.GPTPartTypeHedgehogIdentity,
					FS:          mockfs,
				},
			}
			if tt.pre != nil {
				tt.pre(t, ctrl, mockfs)
			}
			_, err := a.ReadClientCSR()
			if (err != nil) != tt.wantErr {
				t.Errorf("api.ReadClientCSR() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr && tt.wantErrToBe != nil {
				if !errors.Is(err, tt.wantErrToBe) {
					t.Errorf("Open() error = %v, wantErrToBe %v", err, tt.wantErrToBe)
					return
				}
			}
		})
	}
}

func Test_api_StoreClientCert(t *testing.T) {
	type args struct {
		certBytes []byte
	}
	csrValid := readFile("csr-valid.pem")
	certValid := readFile("cert-valid.der")
	certValidPEM := readFile("cert-valid.pem")
	certInvalid := readFile("csr-valid.der")
	errWriteFailed := errors.New("Write() failed tragically")
	errOpenFailed := errors.New("Open() failed tragically")
	tests := []struct {
		name        string
		args        args
		wantErr     bool
		wantErrToBe error
		pre         func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS)
	}{
		{
			name:    "success",
			args:    args{certBytes: certValid},
			wantErr: false,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				f := mockio.NewMockReadWriteCloser(ctrl)
				// CSR
				mfs.EXPECT().Open(clientCSRPath).Times(1).Return(f, nil)
				f.EXPECT().Close().Times(1).Return(nil)
				mockio.ReadAllBytesMock(f, csrValid, 2)

				// Cert
				mfs.EXPECT().OpenFile(gomock.Eq(clientCertPath), gomock.Eq(os.O_CREATE|os.O_TRUNC|os.O_WRONLY), gomock.Eq(fs.FileMode(0644))).Times(1).Return(f, nil)
				f.EXPECT().Close().Times(1).Return(nil)
				f.EXPECT().Write(gomock.Eq(certValidPEM)).Times(1).Return(len(certValidPEM), nil)
			},
		},
		{
			name:        "write to file fails",
			args:        args{certBytes: certValid},
			wantErr:     true,
			wantErrToBe: errWriteFailed,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				f := mockio.NewMockReadWriteCloser(ctrl)
				// CSR
				mfs.EXPECT().Open(clientCSRPath).Times(1).Return(f, nil)
				f.EXPECT().Close().Times(1).Return(nil)
				mockio.ReadAllBytesMock(f, csrValid, 2)

				// Cert
				mfs.EXPECT().OpenFile(gomock.Eq(clientCertPath), gomock.Eq(os.O_CREATE|os.O_TRUNC|os.O_WRONLY), gomock.Eq(fs.FileMode(0644))).Times(1).Return(f, nil)
				f.EXPECT().Close().Times(1).Return(nil)
				f.EXPECT().Write(gomock.Eq(certValidPEM)).Times(1).Return(0, errWriteFailed)
			},
		},
		{
			name:        "opening file fails",
			args:        args{certBytes: certValid},
			wantErr:     true,
			wantErrToBe: errOpenFailed,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				f := mockio.NewMockReadWriteCloser(ctrl)
				// CSR
				mfs.EXPECT().Open(clientCSRPath).Times(1).Return(f, nil)
				f.EXPECT().Close().Times(1).Return(nil)
				mockio.ReadAllBytesMock(f, csrValid, 2)

				// Cert
				mfs.EXPECT().OpenFile(gomock.Eq(clientCertPath), gomock.Eq(os.O_CREATE|os.O_TRUNC|os.O_WRONLY), gomock.Eq(fs.FileMode(0644))).Times(1).Return(nil, errOpenFailed)
			},
		},
		{
			name:    "parsing certificate fails",
			args:    args{certBytes: certInvalid},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockfs := mockpartitions.NewMockFS(ctrl)
			a := &api{
				dev: &partitions.Device{
					Uevent: partitions.Uevent{
						partitions.UeventDevtype: partitions.UeventDevtypePartition,
					},
					GPTPartType: partitions.GPTPartTypeHedgehogIdentity,
					FS:          mockfs,
				},
			}
			if tt.pre != nil {
				tt.pre(t, ctrl, mockfs)
			}
			err := a.StoreClientCert(tt.args.certBytes)
			if (err != nil) != tt.wantErr {
				t.Errorf("api.StoreClientCert() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr && tt.wantErrToBe != nil {
				if !errors.Is(err, tt.wantErrToBe) {
					t.Errorf("Open() error = %v, wantErrToBe %v", err, tt.wantErrToBe)
					return
				}
			}
		})
	}
}

func Test_api_GenerateClientCSR(t *testing.T) {
	keyValid := readFile("key-valid.pem")
	keyValidDER := readFile("key-valid.der")
	keyInvalid := readFile("key-invalid.pem")
	errCreateCSR := errors.New("x509.CreateCertificateRequest() failed tragically")
	errReadFailed := errors.New("Read() failed tragically")
	errOpenFailed := errors.New("Open() failed tragically")
	errWriteFailed := errors.New("Write() failed tragically")
	tests := []struct {
		name                         string
		wantErr                      bool
		wantErrToBe                  error
		pre                          func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS)
		x509CreateCertificateRequest func(rand io.Reader, template *x509.CertificateRequest, priv any) (csr []byte, err error)
		devidID                      func() string
	}{
		{
			name:    "success",
			wantErr: false,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				f := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().Open(gomock.Eq(clientKeyPath)).Times(1).Return(f, nil)
				f.EXPECT().Close().Times(1).Return(nil)
				mockio.ReadAllBytesMock(f, keyValid, 1)
				f2 := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().OpenFile(gomock.Eq(clientCSRPath), gomock.Eq(os.O_CREATE|os.O_TRUNC|os.O_WRONLY), gomock.Eq(fs.FileMode(0644))).Times(1).Return(f2, nil)
				f2.EXPECT().Close().Times(1).Return(nil)
				f2.EXPECT().Write(gomock.Any()).Times(1).DoAndReturn(func(b []byte) (int, error) {
					return len(b), nil
				})
				mfs.EXPECT().Remove(gomock.Eq(clientCertPath)).Times(1).Return(os.ErrNotExist)
			},
		},
		{
			name:    "success, but deleting previous cert fails",
			wantErr: true,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				f := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().Open(gomock.Eq(clientKeyPath)).Times(1).Return(f, nil)
				f.EXPECT().Close().Times(1).Return(nil)
				mockio.ReadAllBytesMock(f, keyValid, 1)
				f2 := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().OpenFile(gomock.Eq(clientCSRPath), gomock.Eq(os.O_CREATE|os.O_TRUNC|os.O_WRONLY), gomock.Eq(fs.FileMode(0644))).Times(1).Return(f2, nil)
				f2.EXPECT().Close().Times(1).Return(nil)
				f2.EXPECT().Write(gomock.Any()).Times(1).DoAndReturn(func(b []byte) (int, error) {
					return len(b), nil
				})
				mfs.EXPECT().Remove(gomock.Eq(clientCertPath)).Times(1).Return(os.ErrPermission)
			},
		},
		{
			name:        "writing CSR to disk fails",
			wantErr:     true,
			wantErrToBe: errWriteFailed,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				f := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().Open(gomock.Eq(clientKeyPath)).Times(1).Return(f, nil)
				f.EXPECT().Close().Times(1).Return(nil)
				mockio.ReadAllBytesMock(f, keyValid, 1)
				f2 := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().OpenFile(gomock.Eq(clientCSRPath), gomock.Eq(os.O_CREATE|os.O_TRUNC|os.O_WRONLY), gomock.Eq(fs.FileMode(0644))).Times(1).Return(f2, nil)
				f2.EXPECT().Close().Times(1).Return(nil)
				f2.EXPECT().Write(gomock.Any()).Times(1).Return(0, errWriteFailed)
			},
		},
		{
			name:        "opening CSR file fails",
			wantErr:     true,
			wantErrToBe: errOpenFailed,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				f := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().Open(gomock.Eq(clientKeyPath)).Times(1).Return(f, nil)
				f.EXPECT().Close().Times(1).Return(nil)
				mockio.ReadAllBytesMock(f, keyValid, 1)
				mfs.EXPECT().OpenFile(gomock.Eq(clientCSRPath), gomock.Eq(os.O_CREATE|os.O_TRUNC|os.O_WRONLY), gomock.Eq(fs.FileMode(0644))).Times(1).Return(nil, errOpenFailed)
			},
		},
		{
			name:        "creating CSR fails",
			wantErr:     true,
			wantErrToBe: errCreateCSR,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				f := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().Open(clientKeyPath).Times(1).Return(f, nil)
				f.EXPECT().Close().Times(1).Return(nil)
				mockio.ReadAllBytesMock(f, keyValid, 1)
			},
			x509CreateCertificateRequest: func(rand io.Reader, template *x509.CertificateRequest, priv any) (csr []byte, err error) {
				return nil, errCreateCSR
			},
		},
		{
			name:        "no device ID",
			wantErr:     true,
			wantErrToBe: ErrNoDevID,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				f := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().Open(clientKeyPath).Times(1).Return(f, nil)
				f.EXPECT().Close().Times(1).Return(nil)
				mockio.ReadAllBytesMock(f, keyValid, 1)
			},
			devidID: func() string {
				return ""
			},
		},
		{
			name:    "invalid key file",
			wantErr: true,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				f := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().Open(clientKeyPath).Times(1).Return(f, nil)
				f.EXPECT().Close().Times(1).Return(nil)
				mockio.ReadAllBytesMock(f, keyInvalid, 1)
			},
		},
		{
			name:        "key file not PEM",
			wantErr:     true,
			wantErrToBe: ErrNoPEMData,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				f := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().Open(clientKeyPath).Times(1).Return(f, nil)
				f.EXPECT().Close().Times(1).Return(nil)
				mockio.ReadAllBytesMock(f, keyValidDER, 1)
			},
		},
		{
			name:        "reading key file fails",
			wantErr:     true,
			wantErrToBe: errReadFailed,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				f := mockio.NewMockReadWriteCloser(ctrl)
				mfs.EXPECT().Open(clientKeyPath).Times(1).Return(f, nil)
				f.EXPECT().Close().Times(1).Return(nil)
				f.EXPECT().Read(gomock.Any()).Times(1).Return(0, errReadFailed)
			},
		},
		{
			name:        "opening key file fails",
			wantErr:     true,
			wantErrToBe: errOpenFailed,
			pre: func(t *testing.T, ctrl *gomock.Controller, mfs *mockpartitions.MockFS) {
				mfs.EXPECT().Open(clientKeyPath).Times(1).Return(nil, errOpenFailed)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockfs := mockpartitions.NewMockFS(ctrl)
			a := &api{
				dev: &partitions.Device{
					Uevent: partitions.Uevent{
						partitions.UeventDevtype: partitions.UeventDevtypePartition,
					},
					GPTPartType: partitions.GPTPartTypeHedgehogIdentity,
					FS:          mockfs,
				},
			}
			if tt.x509CreateCertificateRequest != nil {
				oldX509CreateCertificateRequest := x509CreateCertificateRequest
				defer func() {
					x509CreateCertificateRequest = oldX509CreateCertificateRequest
				}()
				x509CreateCertificateRequest = tt.x509CreateCertificateRequest
			}
			if tt.devidID != nil {
				oldDevidID := devidID
				defer func() {
					devidID = oldDevidID
				}()
				devidID = tt.devidID
			}
			if tt.pre != nil {
				tt.pre(t, ctrl, mockfs)
			}
			_, err := a.GenerateClientCSR()
			if (err != nil) != tt.wantErr {
				t.Errorf("api.GenerateClientCSR() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr && tt.wantErrToBe != nil {
				if !errors.Is(err, tt.wantErrToBe) {
					t.Errorf("api.GenerateClientCSR() error = %v, wantErrToBe %v", err, tt.wantErrToBe)
					return
				}
			}
		})
	}
}
