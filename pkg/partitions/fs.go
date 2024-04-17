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
)

//go:generate mockgen -destination ../../test/mock/mockpartitions/fs_mock.go -package mockpartitions go.githedgehog.com/dasboot/pkg/partitions FS
//go:generate mockgen -destination ../../test/mock/mockio/io_readwritecloser.go -package mockio io ReadWriteCloser
//go:generate mockgen -destination ../../test/mock/mockio/mockfs/fs_fileinfo.go -package mockfs "io/fs" FileInfo
//go:generate mockgen -destination ../../test/mock/mockio/mockfs/fs_direntry.go -package mockfs "io/fs" DirEntry
type FS interface {
	SetBase(basePath string)
	Path(name string) string
	Open(name string) (io.ReadWriteCloser, error)
	OpenFile(name string, flag int, perm fs.FileMode) (io.ReadWriteCloser, error)
	Stat(name string) (fs.FileInfo, error)
	ReadDir(name string) ([]fs.DirEntry, error)
	Remove(path string) error
	RemoveAll(path string) error
	Mkdir(name string, perm fs.FileMode) error
}
