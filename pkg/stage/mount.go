package stage

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
)

var (
	ErrNotADirectory   = errors.New("not a directory")
	ErrNotASyscallStat = errors.New("not a syscall stat")
)

// IsMountPoint checks if path is a mount point by comparing if the underlying device between the directory
// and its parent differs.
func IsMountPoint(path string) (bool, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return false, err
	}

	if !fi.IsDir() {
		return false, ErrNotADirectory
	}

	stat, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return false, ErrNotASyscallStat
	}

	fiParent, err := os.Stat(filepath.Dir(path))
	if err != nil {
		return false, err
	}

	if !fiParent.IsDir() {
		return false, ErrNotADirectory
	}

	parentStat, ok := fiParent.Sys().(*syscall.Stat_t)
	if !ok {
		return false, ErrNotASyscallStat
	}

	// compare the device numbers of the directory and its parent
	// this will determine if they are on different filesystems and therefore
	// if this is a mountpoint
	return stat.Dev != parentStat.Dev, nil
}
