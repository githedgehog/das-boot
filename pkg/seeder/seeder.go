package seeder

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Interface interacts with a seeder instance.
type Interface interface {
	// Start will start the seeder and its servers in the background. This function will return
	// probably even before listeners are start.
	Start()

	// Stop tells the seeder to stop all running servers. It is trying a graceful shutdown at first,
	// but will close the servers if the context timeouts or after 30 seconds if the context did not
	// timeout before that.
	Stop(context.Context)
}

type seeder struct {
	secureServer   *server
	insecureServer *server
}

var _ Interface = &seeder{}

var (
	ErrInvalidConfig = errors.New("seeder: invalid config")
)

func invalidConfigError(str string) error {
	return fmt.Errorf("%w: %s", ErrInvalidConfig, str)
}

func New(config *Config) (Interface, error) {
	if config == nil {
		return nil, invalidConfigError("empty config")
	}
	if config.InsecureServer == nil && config.SecureServer == nil {
		return nil, invalidConfigError("neither InsecureServer nor SecureServer are set")
	}

	ret := &seeder{}

	if config.InsecureServer != nil {
		var err error
		ret.insecureServer, err = newServer(config.InsecureServer, insecureHandler())
		if err != nil {
			return nil, err
		}
	}

	if config.SecureServer != nil {
		var err error
		ret.secureServer, err = newServer(config.SecureServer, secureHandler())
		if err != nil {
			return nil, err
		}
	}

	return ret, nil
}

func (s *seeder) Start() {
	// fire up our servers
	go s.insecureServer.Start()
	go s.secureServer.Start()
}

func (s *seeder) Stop(pctx context.Context) {
	// whatever context we get passed in, we will definitely cancel after 30 seconds
	ctx, cancel := context.WithTimeout(pctx, time.Second*30)
	defer cancel()

	// try graceful shutdown first
	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		if err := s.insecureServer.Shutdown(ctx); err != nil {
			l.Warn("insecure server: graceful shutdown failed", zap.Error(err))
		}
		wg.Done()
	}()
	go func() {
		if err := s.secureServer.Shutdown(ctx); err != nil {
			l.Warn("secure server: graceful shutdown failed", zap.Error(err))
		}
		wg.Done()
	}()
	go func() {
		wg.Wait()
		close(done)
	}()

	// if graceful shutdown fails, just tear it down
	select {
	case <-ctx.Done():
		if err := s.insecureServer.Close(); err != nil {
			l.Debug("insecure server: error on close", zap.Error(err))
		}
		if err := s.secureServer.Close(); err != nil {
			l.Debug("secure server: error on close", zap.Error(err))
		}
	case <-done:
		// graceful shutdown was successful
	}
}
