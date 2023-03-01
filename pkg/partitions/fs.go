package partitions

import (
	"io"
	"io/fs"
)

//go:generate mockgen -destination ../../test/mock/mockpartitions/fs_mock.go -package mockpartitions . FS
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
	RemoveAll(path string) error
	Mkdir(name string, perm fs.FileMode) error
}
