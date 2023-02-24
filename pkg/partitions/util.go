package partitions

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"

	"go.uber.org/zap"
	"golang.org/x/sys/unix"
)

// for unit testing
var (
	osStat     func(name string) (fs.FileInfo, error)    = os.Stat
	osLstat    func(name string) (fs.FileInfo, error)    = os.Lstat
	osRemove   func(name string) error                   = os.Remove
	osMkdirAll func(path string, perm fs.FileMode) error = os.MkdirAll

	rootPath                                         = "/"
	execCommand func(name string, arg ...string) Cmd = func(name string, arg ...string) Cmd {
		return exec.Command(name, arg...)
	}
	unixIoctlGetInt      func(fd int, req uint) (int, error)                                                 = unix.IoctlGetInt
	unixMount            func(source string, target string, fstype string, flags uintptr, data string) error = unix.Mount
	unixUnmount          func(target string, flags int) error                                                = unix.Unmount
	unixMknod            func(path string, mode uint32, dev int) (err error)                                 = unix.Mknod
	filepathRel          func(basepath string, targpath string) (string, error)                              = filepath.Rel
	filepathEvalSymlinks func(path string) (string, error)                                                   = filepath.EvalSymlinks
)

var Logger = zap.L().With(zap.String("logger", "pkg/partitions"))

// Cmd interface is representing an os/exec Cmd struct.
// This makes it usable to replace for testing.
//
//go:generate mockgen -destination util_mock_cmd_test.go -package partitions . Cmd
type Cmd interface {
	Run() error
	Output() ([]byte, error)
	Start() error
	Wait() error
}

var _ Cmd = &exec.Cmd{}
