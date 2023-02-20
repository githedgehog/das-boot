package partitions

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
	"time"
)

func TestReadUevent(t *testing.T) {
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	type args struct {
		path string
	}
	tests := []struct {
		name    string
		args    args
		want    Uevent
		wantErr bool
	}{
		{
			name: "path does not exist",
			args: args{
				path: filepath.Join(pwd, "testdata", "ReadUevent", "does not exist"),
			},
			wantErr: true,
		},
		{
			name: "read nvme0n1",
			args: args{
				path: filepath.Join(pwd, "testdata", "ReadUevent", "nvme0n1"),
			},
			want: Uevent{
				"MAJOR":   "259",
				"MINOR":   "0",
				"DEVNAME": "nvme0n1",
				"DEVTYPE": "disk",
				"DISKSEQ": "1",
			},
		},
		{
			name: "read nvme0n1p1",
			args: args{
				path: filepath.Join(pwd, "testdata", "ReadUevent", "nvme0n1p1"),
			},
			want: Uevent{
				"MAJOR":    "259",
				"MINOR":    "1",
				"DEVNAME":  "nvme0n1p1",
				"DEVTYPE":  "partition",
				"DISKSEQ":  "1",
				"PARTN":    "1",
				"PARTNAME": "EFI system partition",
			},
		},
		{
			name: "read loop0",
			args: args{
				path: filepath.Join(pwd, "testdata", "ReadUevent", "loop0"),
			},
			want: Uevent{
				"MAJOR":   "7",
				"MINOR":   "0",
				"DEVNAME": "loop0",
				"DEVTYPE": "disk",
				"DISKSEQ": "2",
			},
		},
		{
			name: "unstable input",
			args: args{
				path: filepath.Join(pwd, "testdata", "ReadUevent", "unstable"),
			},
			want: Uevent{
				"MAJOR":   "7",
				"MINOR":   "0",
				"DEVNAME": "loop0",
				"DEVTYPE": "disk",
				"DISKSEQ": "2",
				"WEIRD":   "Value with = inside",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReadUevent(tt.args.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadUevent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ReadUevent() = %v, want %v", got, tt.want)
			}
		})
	}
}

var testOsStat = func(name string) (os.FileInfo, error) {
	return &testFileInfo{}, nil
}

var _ fs.FileInfo = &testFileInfo{}

type testFileInfo struct {
}

// IsDir implements fs.FileInfo
func (*testFileInfo) IsDir() bool {
	return false
}

// ModTime implements fs.FileInfo
func (*testFileInfo) ModTime() time.Time {
	return time.Now()
}

// Mode implements fs.FileInfo
func (*testFileInfo) Mode() fs.FileMode {
	panic("unimplemented")
}

// Name implements fs.FileInfo
func (*testFileInfo) Name() string {
	panic("unimplemented")
}

// Size implements fs.FileInfo
func (*testFileInfo) Size() int64 {
	return 0
}

// Sys implements fs.FileInfo
func (*testFileInfo) Sys() any {
	// return something outrageous
	return complex(1, 2)
}

func TestUevent_DevicePath(t *testing.T) {
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	oldRootPath := rootPath
	rootPath = filepath.Join(pwd, "testdata", "DevicePath")
	defer func() {
		rootPath = oldRootPath
	}()

	// these must be created out of band as it requires root privileges to do so
	// we'll skip the test if they don't exist
	loop0Dev := filepath.Join(pwd, "testdata", "Devicepath", "dev", "loop0")
	urandomDev := filepath.Join(pwd, "testdata", "DevicePath", "dev", "urandom")
	if _, err := os.Stat(loop0Dev); err != nil {
		t.Skipf("SKIPPING: testdata must be initialized: loop0 missing: run 'sudo mknod %s b 7 0'", loop0Dev)
	}
	if _, err := os.Stat(urandomDev); err != nil {
		t.Skipf("SKIPPING: testdata must be initialized: urandom missing: run 'sudo mknod %s c 1 0'", urandomDev)
	}

	tests := []struct {
		name        string
		u           Uevent
		want        string
		wantErr     bool
		wantErrToBe error
		osStat      func(name string) (os.FileInfo, error)
	}{
		{
			name: "success",
			u: Uevent{
				UeventDevname: "loop0",
			},
			want:    filepath.Join(rootPath, "dev", "loop0"),
			wantErr: false,
		},
		{
			name: "is a character device",
			u: Uevent{
				UeventDevname: "urandom",
			},
			wantErr:     true,
			wantErrToBe: ErrNotABlockDevice,
		},
		{
			name: "is a regular file",
			u: Uevent{
				UeventDevname: "text.txt",
			},
			wantErr:     true,
			wantErrToBe: ErrNotABlockDevice,
		},
		{
			name: "device does not exist",
			u: Uevent{
				UeventDevname: "doesnotexist",
			},
			wantErr:     true,
			wantErrToBe: os.ErrNotExist,
		},
		{
			name: "invalid uevent",
			u: Uevent{
				"invalid": "loop0",
			},
			wantErr:     true,
			wantErrToBe: ErrInvalidUevent,
		},
		{
			name: "file not derived from system call",
			u: Uevent{
				UeventDevname: "loop0",
			},
			wantErr:     true,
			wantErrToBe: ErrStatNotFromSyscall,
			osStat:      testOsStat,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.osStat != nil {
				oldOsStat := osStat
				osStat = tt.osStat
				defer func() {
					osStat = oldOsStat
				}()
			}
			got, err := tt.u.DevicePath()
			if (err != nil) != tt.wantErr {
				t.Errorf("Uevent.DevicePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr && tt.wantErrToBe != nil {
				if !errors.Is(err, tt.wantErrToBe) {
					t.Errorf("Uevent.DevicePath() error = %v, wantErrToBe %v", err, tt.wantErrToBe)
				}
			}
			if got != tt.want {
				t.Errorf("Uevent.DevicePath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUevent_GetMajorMinor(t *testing.T) {
	tests := []struct {
		name        string
		u           Uevent
		want        uint32
		want1       uint32
		wantErr     bool
		wantErrToBe error
	}{
		{
			name: "success",
			u: Uevent{
				UeventMajor: "259",
				UeventMinor: "0",
			},
			want:  259,
			want1: 0,
		},
		{
			name: "major is a signed value",
			u: Uevent{
				UeventMajor: "-1",
				UeventMinor: "0",
			},
			wantErr:     true,
			wantErrToBe: strconv.ErrSyntax,
		},
		{
			name: "minor is out of range",
			u: Uevent{
				UeventMajor: "42",
				UeventMinor: "324987198324983214871432871983247983214983219841984329812341723984719843279812749821374981274398129843",
			},
			wantErr:     true,
			wantErrToBe: strconv.ErrRange,
		},
		{
			name: "invalid uevent with major missing",
			u: Uevent{
				UeventMinor: "0",
			},
			wantErr:     true,
			wantErrToBe: ErrInvalidUevent,
		},
		{
			name: "invalid uevent with minor missing",
			u: Uevent{
				UeventMajor: "260",
			},
			wantErr:     true,
			wantErrToBe: ErrInvalidUevent,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := tt.u.GetMajorMinor()
			if (err != nil) != tt.wantErr {
				t.Errorf("Uevent.GetMajorMinor() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr && tt.wantErrToBe != nil {
				if !errors.Is(err, tt.wantErrToBe) {
					t.Errorf("Uevent.DevicePath() error = %v, wantErrToBe %v", err, tt.wantErrToBe)
				}
			}
			if got != tt.want {
				t.Errorf("Uevent.GetMajorMinor() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("Uevent.GetMajorMinor() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestUevent_IsDisk(t *testing.T) {
	tests := []struct {
		name string
		u    Uevent
		want bool
	}{
		{
			name: "success",
			u: Uevent{
				UeventDevtype: "disk",
			},
			want: true,
		},
		{
			name: "not a disk",
			u: Uevent{
				UeventDevtype: "partition",
			},
			want: false,
		},
		{
			name: "invalid partname value",
			u: Uevent{
				UeventDevtype: "invalid",
			},
			want: false,
		},
		{
			name: "invalid uevent",
			u:    Uevent{},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.u.IsDisk(); got != tt.want {
				t.Errorf("Uevent.IsDisk() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUevent_IsPartition(t *testing.T) {
	tests := []struct {
		name string
		u    Uevent
		want bool
	}{
		{
			name: "success",
			u: Uevent{
				UeventDevtype: "partition",
			},
			want: true,
		},
		{
			name: "not a partition",
			u: Uevent{
				UeventDevtype: "disk",
			},
			want: false,
		},
		{
			name: "invalid partname value",
			u: Uevent{
				UeventDevtype: "invalid",
			},
			want: false,
		},
		{
			name: "invalid uevent",
			u:    Uevent{},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.u.IsPartition(); got != tt.want {
				t.Errorf("Uevent.IsDisk() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUevent_GetPartitionNumber(t *testing.T) {
	tests := []struct {
		name string
		u    Uevent
		want int
	}{
		{
			name: "success",
			u: Uevent{
				UeventPartn: "6",
			},
			want: 6,
		},
		{
			name: "invalid uevent",
			u:    Uevent{},
			want: -1,
		},
		{
			name: "not a partition",
			u: Uevent{
				UeventDevtype: "disk",
			},
			want: -1,
		},
		{
			name: "invalid partition number",
			u: Uevent{
				UeventPartn: "not a number",
			},
			want: -1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.u.GetPartitionNumber(); got != tt.want {
				t.Errorf("Uevent.GetPartitionNumber() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUevent_GetPartitionName(t *testing.T) {
	tests := []struct {
		name string
		u    Uevent
		want string
	}{
		{
			name: "success",
			u: Uevent{
				UeventPartname: "HH-DIAG",
			},
			want: "HH-DIAG",
		},
		{
			name: "invalid uevent",
			u:    Uevent{},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.u.GetPartitionName(); got != tt.want {
				t.Errorf("Uevent.GetPartitionName() = %v, want %v", got, tt.want)
			}
		})
	}
}
