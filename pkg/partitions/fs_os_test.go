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
	"errors"
	"os"
	"testing"
)

func Test_fsOs(t *testing.T) {
	t.Run("SetBase", func(t *testing.T) {
		fs := &fsOs{}
		fs.SetBase("test")
		if fs.base != "test" {
			t.Errorf("SetBase() not working as expected")
		}
	})
	t.Run("Path", func(t *testing.T) {
		var panicHappened bool
		func() {
			defer func() {
				r := recover()
				panicHappened = r != nil
			}()
			fs := &fsOs{}
			fs.Path("/call/should/panic")
		}()
		if !panicHappened {
			t.Errorf("Path() expected panic to happen")
			return
		}
		fs := &fsOs{
			base: "/base",
		}
		if fs.Path("path") != "/base/path" {
			t.Errorf("Path() not returning path as expected")
		}
	})
	t.Run("Mkdir", func(t *testing.T) {
		fs, path := prepFsOsTest()
		defer cleanFsOsTest(path)

		if err := fs.Mkdir("mkdir", 0755); err != nil {
			t.Errorf("Mkdir() failed: %v", err)
			return
		}
		info, err := os.Stat(fs.Path("mkdir"))
		if err != nil {
			t.Errorf("Mkdir() stat failed: %v", err)
		}
		if !info.IsDir() {
			t.Errorf("MKdir() did not create dir")
		}

		fs.base = ""
		if err := fs.Mkdir("mkdir", 0755); err == nil || !errors.Is(err, ErrNotMounted) {
			t.Errorf("Mkdir() no error on empty base or wrong error")
		}
	})
	t.Run("Open", func(t *testing.T) {
		fs, path := prepFsOsTest()
		defer cleanFsOsTest(path)
		prepFsOsFile(fs, "open")

		f, err := fs.Open("open")
		if err != nil {
			t.Errorf("Open() failed unexpectedly: %v", err)
			return
		}
		f.Close()

		fs.base = ""
		if _, err := fs.Open("open"); err == nil || !errors.Is(err, ErrNotMounted) {
			t.Errorf("Open() no error on empty base or wrong error")
		}
	})
	t.Run("OpenFile", func(t *testing.T) {
		fs, path := prepFsOsTest()
		defer cleanFsOsTest(path)
		prepFsOsFile(fs, "openfile")

		f, err := fs.OpenFile("openfile", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
		if err != nil {
			t.Errorf("OpenFile() failed unexpectedly: %v", err)
			return
		}
		f.Close()

		fs.base = ""
		if _, err := fs.OpenFile("openfile", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644); err == nil || !errors.Is(err, ErrNotMounted) {
			t.Errorf("OpenFile() no error on empty base or wrong error")
		}
	})
	t.Run("ReadDir", func(t *testing.T) {
		fs, path := prepFsOsTest()
		defer cleanFsOsTest(path)
		prepFsOsFile(fs, "readdir")

		entries, err := fs.ReadDir("")
		if err != nil {
			t.Errorf("ReadDir() failed unexpectedly: %v", err)
			return
		}
		if len(entries) != 1 {
			t.Errorf("ReadDir() returned unexpected number of entries: %d", len(entries))
		}

		fs.base = ""
		if _, err := fs.ReadDir("openfile"); err == nil || !errors.Is(err, ErrNotMounted) {
			t.Errorf("ReadDir() no error on empty base or wrong error")
		}
	})
	t.Run("RemoveAll", func(t *testing.T) {
		fs, path := prepFsOsTest()
		defer cleanFsOsTest(path)
		prepFsOsDir(fs, "/a/b/c")
		prepFsOsFile(fs, "/a/b/c/removeall")

		err := fs.RemoveAll("a")
		if err != nil {
			t.Errorf("RemoveAll() failed unexpectedly: %v", err)
			return
		}

		fs.base = ""
		if err := fs.RemoveAll("a"); err == nil || !errors.Is(err, ErrNotMounted) {
			t.Errorf("RemoveAll() no error on empty base or wrong error")
		}
	})
	t.Run("Stat", func(t *testing.T) {
		fs, path := prepFsOsTest()
		defer cleanFsOsTest(path)
		prepFsOsDir(fs, "/a/b/c")

		info, err := fs.Stat("a")
		if err != nil {
			t.Errorf("Stat() failed unexpectedly: %v", err)
			return
		}
		if !info.IsDir() {
			t.Errorf("Stat() for wrong file apparently")
			return
		}
		if info.Name() != "a" {
			t.Errorf("Stat() for wrong file apparently")
			return
		}

		fs.base = ""
		if _, err := fs.Stat("a"); err == nil || !errors.Is(err, ErrNotMounted) {
			t.Errorf("Stat() no error on empty base or wrong error")
		}
	})
}

func prepFsOsTest() (*fsOs, string) {
	path, err := os.MkdirTemp("", "das-boot-test-fsOs-")
	if err != nil {
		panic(err)
	}
	return &fsOs{base: path}, path
}

func cleanFsOsTest(path string) {
	os.RemoveAll(path)
}

func prepFsOsFile(fs *fsOs, path string) {
	f, err := os.Create(fs.Path(path))
	if err != nil {
		panic(err)
	}
	defer f.Close()
}

func prepFsOsDir(fs *fsOs, path string) {
	err := os.MkdirAll(fs.Path(path), 0755)
	if err != nil {
		panic(err)
	}
}
