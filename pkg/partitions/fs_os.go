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

package partitions

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

type fsOs struct {
	base string
}

var _ FS = &fsOs{}

// SetBase implements FS
func (fs *fsOs) SetBase(basePath string) {
	fs.base = basePath
}

func (fs *fsOs) Path(name string) string {
	if fs.base == "" {
		panic(ErrNotMounted)
	}
	return filepath.Join(fs.base, name)
}

// Mkdir implements FS
func (fs *fsOs) Mkdir(name string, perm fs.FileMode) error {
	if fs.base == "" {
		return ErrNotMounted
	}
	return os.Mkdir(filepath.Join(fs.base, name), perm)
}

// Open implements FS
func (fs *fsOs) Open(name string) (io.ReadWriteCloser, error) {
	if fs.base == "" {
		return nil, ErrNotMounted
	}
	return os.Open(filepath.Join(fs.base, name))
}

// OpenFile implements FS
func (fs *fsOs) OpenFile(name string, flag int, perm fs.FileMode) (io.ReadWriteCloser, error) {
	if fs.base == "" {
		return nil, ErrNotMounted
	}
	return os.OpenFile(filepath.Join(fs.base, name), flag, perm)
}

// ReadDir implements FS
func (fs *fsOs) ReadDir(name string) ([]fs.DirEntry, error) {
	if fs.base == "" {
		return nil, ErrNotMounted
	}
	return os.ReadDir(filepath.Join(fs.base, name))
}

// Remove implements FS
func (fs *fsOs) Remove(path string) error {
	if fs.base == "" {
		return ErrNotMounted
	}
	return os.Remove(filepath.Join(fs.base, path))
}

// RemoveAll implements FS
func (fs *fsOs) RemoveAll(path string) error {
	if fs.base == "" {
		return ErrNotMounted
	}
	return os.RemoveAll(filepath.Join(fs.base, path))
}

// Stat implements FS
func (fs *fsOs) Stat(name string) (fs.FileInfo, error) {
	if fs.base == "" {
		return nil, ErrNotMounted
	}
	return os.Stat(filepath.Join(fs.base, name))
}
