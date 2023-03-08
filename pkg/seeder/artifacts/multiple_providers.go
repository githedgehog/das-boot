package artifacts

import "io"

type multipleProviders struct {
	providers []Provider
}

var _ Provider = &multipleProviders{}

func New(providers ...Provider) Provider {
	return &multipleProviders{providers: providers}
}

func (m *multipleProviders) Get(artifact string) io.ReadCloser {
	for _, p := range m.providers {
		if r := p.Get(artifact); r != nil {
			return r
		}
	}
	return nil
}
