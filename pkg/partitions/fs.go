package partitions

import (
	"io"
	"io/fs"
)

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
