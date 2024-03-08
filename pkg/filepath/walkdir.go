// Copyright 2023 Hedgehog
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package filepath

import (
	"io/fs"
	"os"
	"path/filepath"
)

var (
	osLstat              func(name string) (fs.FileInfo, error)                 = os.Lstat
	filepathRel          func(basepath string, targpath string) (string, error) = filepath.Rel
	filepathEvalSymlinks func(path string) (string, error)                      = filepath.EvalSymlinks
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
		if err == nil {
			for _, entry := range exclusions {
				if info.Name() == entry {
					excluded = true
					break
				}
			}
		}

		if err == nil && info.Type()&os.ModeSymlink == os.ModeSymlink && !excluded {
			finalPath, err := filepathEvalSymlinks(path)
			if err != nil {
				return walkFn(path, info, err)
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
