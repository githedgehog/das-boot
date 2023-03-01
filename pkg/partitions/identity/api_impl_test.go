package identity

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"errors"
	"io"
	"io/fs"
	"os"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"go.githedgehog.com/dasboot/pkg/partitions"
	"go.githedgehog.com/dasboot/pkg/partitions/location"
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
