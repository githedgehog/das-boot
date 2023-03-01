package mockio

import (
	"io"

	gomock "github.com/golang/mock/gomock"
)

func ReadAllBytesMock(f *MockReadWriteCloser, b []byte, times int) {
	var index int
	file := b
	f.EXPECT().Read(gomock.Any()).Times(times).DoAndReturn(func(b []byte) (int, error) {
		l := len(file) - index
		if len(b) < l {
			l = len(b)
		}
		var err error
		if index+l >= len(file) {
			err = io.EOF
		}
		copy(b, file[index:index+l])
		read := len(file[index : index+l])
		index += l
		return read, err
	})
}
