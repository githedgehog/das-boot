package seeder

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
)

type server struct {
	done        chan struct{}
	err         chan error
	httpServers []*httpServer
}

func newServer(b *BindInfo, handler http.Handler) (*server, error) {
	if len(b.Address) == 0 {
		return nil, invalidConfigError("no address in server config")
	}
	if (b.ServerKeyPath != "" && b.ServerCertPath == "") || (b.ServerCertPath != "" && b.ServerKeyPath == "") {
		return nil, invalidConfigError("server key and server cert must always be set together")
	}

	ret := &server{
		done: make(chan struct{}),
		err:  make(chan error, len(b.Address)),
	}
	for _, addr := range b.Address {
		if addr == "" {
			return nil, invalidConfigError("address must not be empty")
		}
		ret.httpServers = append(ret.httpServers, newHttpServer(addr, b.ServerKeyPath, b.ServerCertPath, b.ClientCAPath, handler))
	}
	return ret, nil
}

func (s *server) Done() <-chan struct{} {
	return s.done
}

func (s *server) Err() <-chan error {
	return s.err
}

func (s *server) Start() {
	var wg sync.WaitGroup
	wg.Add(len(s.httpServers))

	for i, hs := range s.httpServers {
		go func(_ int, hs *httpServer) {
			hs.Start()
			<-hs.Done()
			// we filter out all ErrServerClosed which are generated by Shutdown or Closed calls
			if err := hs.Err(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				s.err <- fmt.Errorf("server on '%s': %w", hs.srv.Addr, err)
			}
			wg.Done()
		}(i, hs)
	}

	go func() {
		wg.Wait()
		close(s.done)
		close(s.err)
	}()
}

func (s *server) Shutdown(ctx context.Context) error {
	var wg sync.WaitGroup
	var errs []error
	errch := make(chan error, len(s.httpServers))
	wg.Add(len(s.httpServers))

	// fan out shutdown commands to all servers
	for _, hs := range s.httpServers {
		go func(hs *httpServer) {
			if err := hs.Shutdown(ctx); err != nil {
				errch <- fmt.Errorf("server on '%s': %w", hs.srv.Addr, err)
			}
			wg.Done()
		}(hs)
	}

	// collect errors
	done := make(chan struct{})
	go func() {
		for {
			select {
			case err := <-errch:
				errs = append(errs, err)
			case <-done:
				return
			}
		}
	}()

	// wait until all shutdowns have run
	wg.Wait()
	close(done)

	// return accordingly
	if len(errs) == 0 {
		return nil
	} else if len(errs) == 1 {
		return fmt.Errorf("shutdown error: %w", errs[0])
	} else {
		return fmt.Errorf("multiple shutdown errors:\n%w", errors.Join(errs...))
	}
}

func (s *server) Close() error {
	var wg sync.WaitGroup
	var errs []error
	errch := make(chan error, len(s.httpServers))
	wg.Add(len(s.httpServers))

	// fan out close commands to all servers
	for _, hs := range s.httpServers {
		go func(hs *httpServer) {
			if err := hs.Close(); err != nil {
				errch <- fmt.Errorf("server on '%s': %w", hs.srv.Addr, err)
			}
			wg.Done()
		}(hs)
	}

	// collect errors
	done := make(chan struct{})
	go func() {
		for {
			select {
			case err := <-errch:
				errs = append(errs, err)
			case <-done:
				return
			}
		}
	}()

	wg.Wait()
	close(done)

	// return accordingly
	if len(errs) == 0 {
		return nil
	} else if len(errs) == 1 {
		return fmt.Errorf("close error: %w", errs[0])
	} else {
		return fmt.Errorf("multiple close errors:\n%w", errors.Join(errs...))
	}
}
