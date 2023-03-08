package artifacts

import "io"

// Provider is an interface for retrieving artifacts as they are going to be used by the seeder servers.
type Provider interface {
	Get(string) io.ReadCloser
}
