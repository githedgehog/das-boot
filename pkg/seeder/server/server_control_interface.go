package server

import "context"

// ControlInterface is a controller interface for the HTTP servers
// within the seeder
type ControlInterface interface {
	Done() <-chan struct{}
	Err() <-chan error
	Start()
	Shutdown(ctx context.Context) error
	Close() error
}
