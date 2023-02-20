package partitions

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
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

		if fname, err := filepath.Rel(filename, path); err == nil {
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
			finalPath, err := filepath.EvalSymlinks(path)
			if err != nil {
				return err
			}
			info, err := os.Lstat(finalPath)
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
