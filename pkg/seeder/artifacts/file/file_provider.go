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
