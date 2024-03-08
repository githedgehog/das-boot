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

package file

import (
	"bufio"
	"io"
	"os"
	"path/filepath"

	"go.githedgehog.com/dasboot/pkg/log"
	"go.githedgehog.com/dasboot/pkg/seeder/artifacts"
	"go.uber.org/zap"
)

type fileProvider struct {
	base string
}

var _ artifacts.Provider = &fileProvider{}

// Provider will create a new file based artifacts provider
// which tries to serve artifacts from directory `path`.
// Every requested artifact must be a file within that
// directory.
func Provider(path string) artifacts.Provider {
	return &fileProvider{
		base: path,
	}
}

// Get implements artifacts.Provider
func (p *fileProvider) Get(artifact string) io.ReadCloser {
	path := filepath.Join(p.base, artifact)
	f, err := os.Open(path)
	if err != nil {
		log.L().Error("open failed", zap.String("provider", "file"), zap.String("artifact", artifact), zap.String("path", path), zap.Error(err))
		return nil
	}
	return newBufioReadCloser(f)
}

type bufioReadCloser struct {
	f *os.File
	b *bufio.Reader
}

// Read implements io.ReadCloser
func (rc *bufioReadCloser) Read(p []byte) (n int, err error) {
	return rc.b.Read(p)
}

// Close implements io.ReadCloser
func (rc *bufioReadCloser) Close() error {
	return rc.f.Close()
}

func newBufioReadCloser(f *os.File) io.ReadCloser {
	return &bufioReadCloser{
		f: f,
		b: bufio.NewReader(f),
	}
}
