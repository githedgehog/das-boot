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
	"io/fs"
	"os"

	"golang.org/x/sys/unix"
)

// for unit testing
var (
	rootPath = "/"

	osStat          func(name string) (fs.FileInfo, error)                                              = os.Stat
	osLstat         func(name string) (fs.FileInfo, error)                                              = os.Lstat //nolint: unused
	osRemove        func(name string) error                                                             = os.Remove
	osMkdirAll      func(path string, perm fs.FileMode) error                                           = os.MkdirAll
	unixIoctlGetInt func(fd int, req uint) (int, error)                                                 = unix.IoctlGetInt //nolint: unused
	unixMount       func(source string, target string, fstype string, flags uintptr, data string) error = unix.Mount
	unixUnmount     func(target string, flags int) error                                                = unix.Unmount
	unixMknod       func(path string, mode uint32, dev int) (err error)                                 = unix.Mknod
)
