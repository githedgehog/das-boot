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
