package location

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
		GPTPartType: partitions.GPTPartTypeHedgehogLocation,
	}
	errOpenFailed := errors.New("open failed tragically")
	tests := []struct {
		name        string
		args        args
		want        LocationPartition
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
					return len(versionBytes), nil
				})
				versionFile.EXPECT().Close().Times(1)
				fs.EXPECT().Open(gomock.Eq(versionFilePath)).Times(1).Return(versionFile, nil)
			},
		},
		{
			name: "not a location partition",
			args: args{d: &partitions.Device{
				Uevent: partitions.Uevent{
					partitions.UeventDevtype: partitions.UeventDevtypePartition,
				},
				GPTPartType: partitions.GPTPartTypeHedgehogIdentity,
			}},
			wantErr:     true,
			wantErrToBe: ErrWrongDevice,
		},
		{
			name: "version file does not exist",
			args: args{d: d},
			pre: func(t *testing.T, ctrl *gomock.Controller, fs *mockpartitions.MockFS) {
				fs.EXPECT().Open(gomock.Eq(versionFilePath)).Times(1).Return(nil, os.ErrNotExist)
			},
			wantErr:     true,
			wantErrToBe: ErrUninitializedPartition,
		},
		{
			name: "version file cannot be opened",
			args: args{d: d},
			pre: func(t *testing.T, ctrl *gomock.Controller, fs *mockpartitions.MockFS) {
				fs.EXPECT().Open(gomock.Eq(versionFilePath)).Times(1).Return(nil, errOpenFailed)
			},
			wantErr:     true,
			wantErrToBe: errOpenFailed,
		},
		{
			name: "version file is not valid JSON",
			args: args{d: d},
			pre: func(t *testing.T, ctrl *gomock.Controller, fs *mockpartitions.MockFS) {
				versionFile := mockio.NewMockReadWriteCloser(ctrl)
				versionFile.EXPECT().Read(gomock.Any()).AnyTimes().DoAndReturn(func(b []byte) (int, error) {
					versionBytes := `{"versio`
					copy(b, []byte(versionBytes))
					return len(versionBytes), nil
				})
				versionFile.EXPECT().Close().Times(1)
				fs.EXPECT().Open(gomock.Eq(versionFilePath)).Times(1).Return(versionFile, nil)
			},
			wantErr: true,
		},
		{
			name: "unsupported version",
			args: args{d: d},
			pre: func(t *testing.T, ctrl *gomock.Controller, fs *mockpartitions.MockFS) {
				versionFile := mockio.NewMockReadWriteCloser(ctrl)
				versionFile.EXPECT().Read(gomock.Any()).Times(1).DoAndReturn(func(b []byte) (int, error) {
					versionBytes := `{"version":2}`
					copy(b, []byte(versionBytes))
					return len(versionBytes), nil
				})
				versionFile.EXPECT().Close().Times(1)
				fs.EXPECT().Open(gomock.Eq(versionFilePath)).Times(1).Return(versionFile, nil)
			},
			wantErr:     true,
			wantErrToBe: ErrUnsupportedVersion,
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

func Test_api_GetLocation(t *testing.T) {
	tests := []struct {
		name        string
		want        *Info
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
			want: &Info{
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
					GPTPartType: partitions.GPTPartTypeHedgehogLocation,
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
