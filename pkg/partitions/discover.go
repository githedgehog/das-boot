package partitions

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

var sysfsPath = filepath.Join(rootPath, "sys")

func Discover() (interface{}, error) {
	var ret []*Device
	// var walkFunc func(path string, d fs.FileInfo, err error) error
	walkFunc := func(path string, d fs.DirEntry, err error) error {
		// fmt.Printf("%s\n", path)
		if d.Name() == "uevent" {
			entry, err := ReadUevent(path)
			if err != nil {
				return nil
			}
			dev := &Device{
				Uevent: entry,
				Path:   filepath.Dir(path),
			}
			ret = append(ret, dev)
		}
		return nil
	}
	if err := WalkDir(filepath.Join(sysfsPath, "block"), walkFunc, 2); err != nil {
		return nil, fmt.Errorf("partitions: discover: %w", err)
	}

	// stupid, but I don't know right now what else to do
	for _, dev := range ret {
		if dev.IsDisk() {
			for _, dev2 := range ret {
				if dev2.IsPartition() && strings.HasPrefix(dev2.Path, dev.Path) {
					dev2.Disk = dev
					dev.Partitions = append(dev.Partitions, dev2)
				}
			}
		}
	}
	for _, dev := range ret {

		fmt.Printf("%#v\n", *dev)
	}
	return ret, nil
}

// WalkDir extends filepath.WalkDir to also follow symlinks but only until maxLevel depth
func WalkDir(path string, walkFn fs.WalkDirFunc, maxLevel uint) error {
	return walkDir(path, path, walkFn, maxLevel)
}

func walkDir(filename string, linkDirname string, walkFn fs.WalkDirFunc, maxLevel uint) error {
	symWalkFunc := func(path string, info fs.DirEntry, err error) error {

		if fname, err := filepath.Rel(filename, path); err == nil {
			path = filepath.Join(linkDirname, fname)
		} else {
			return err
		}
		if err == nil && info.Type()&os.ModeSymlink == os.ModeSymlink {
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
					return walkDir(finalPath, path, walkFn, maxLevel)
				}
			}
		}

		return walkFn(path, info, err)
	}
	return filepath.WalkDir(filename, symWalkFunc)
}
