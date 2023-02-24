package partitions

import (
	"io/fs"
	"os"

	"go.uber.org/zap"
	"golang.org/x/sys/unix"
)

// for unit testing
var (
	rootPath = "/"

	osStat          func(name string) (fs.FileInfo, error)                                              = os.Stat
	osLstat         func(name string) (fs.FileInfo, error)                                              = os.Lstat
	osRemove        func(name string) error                                                             = os.Remove
	osMkdirAll      func(path string, perm fs.FileMode) error                                           = os.MkdirAll
	unixIoctlGetInt func(fd int, req uint) (int, error)                                                 = unix.IoctlGetInt
	unixMount       func(source string, target string, fstype string, flags uintptr, data string) error = unix.Mount
	unixUnmount     func(target string, flags int) error                                                = unix.Unmount
	unixMknod       func(path string, mode uint32, dev int) (err error)                                 = unix.Mknod
)

var Logger = zap.L().With(zap.String("logger", "pkg/partitions"))
