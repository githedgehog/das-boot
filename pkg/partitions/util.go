package partitions

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"

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

// WalkDir extends filepath.WalkDir to also follow symlinks but only until maxLevel depth
// You can specify symlinks that you do not want to follow in the exclusions list.
// These entries are not full paths, but only file names.
//
// taken and adjusted from `github.com/facebookgo/symwalk`
func WalkDir(path string, walkFn fs.WalkDirFunc, maxLevel uint, exclusions ...string) error {
	return walkDir(path, path, walkFn, maxLevel+1, exclusions)
}

func walkDir(filename string, linkDirname string, walkFn fs.WalkDirFunc, maxLevel uint, exclusions []string) error {
	symWalkFunc := func(path string, info fs.DirEntry, err error) error {

		if fname, err := filepathRel(filename, path); err == nil {
			path = filepath.Join(linkDirname, fname)
		} else {
			return err
		}
		var excluded bool
		for _, entry := range exclusions {
			if info.Name() == entry {
				excluded = true
				break
			}
		}
		if err == nil && info.Type()&os.ModeSymlink == os.ModeSymlink && !excluded {
			finalPath, err := filepathEvalSymlinks(path)
			if err != nil {
				return err
			}
			info, err := osLstat(finalPath)
			if err != nil {
				return walkFn(path, fs.FileInfoToDirEntry(info), err)
			}
			if info.IsDir() {
				maxLevel := maxLevel - 1
				if maxLevel > 0 {
					return walkDir(finalPath, path, walkFn, maxLevel, exclusions)
				}
			}
		}

		return walkFn(path, info, err)
	}
	return filepath.WalkDir(filename, symWalkFunc)
}

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
