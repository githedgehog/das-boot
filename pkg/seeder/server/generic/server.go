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

package generic

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"go.githedgehog.com/dasboot/pkg/seeder/config"
	seedererrors "go.githedgehog.com/dasboot/pkg/seeder/errors"
	"go.githedgehog.com/dasboot/pkg/seeder/server"
)

type GenericServer struct {
	done        chan struct{}
	err         chan error
	HTTPServers []*HTTPServer
}

var _ server.ControlInterface = &GenericServer{}

func NewGenericServer(b *config.BindInfo, handler http.Handler) (*GenericServer, error) {
	if len(b.Address) == 0 {
		return nil, seedererrors.InvalidConfigError("no address in server config")
	}
	if (b.ServerKeyPath != "" && b.ServerCertPath == "") || (b.ServerCertPath != "" && b.ServerKeyPath == "") {
		return nil, seedererrors.InvalidConfigError("server key and server cert must always be set together")
	}

	ret := &GenericServer{
		done: make(chan struct{}),
		err:  make(chan error, len(b.Address)),
	}
	for _, addr := range b.Address {
		if addr == "" {
			return nil, seedererrors.InvalidConfigError("address must not be empty")
		}
		ret.HTTPServers = append(ret.HTTPServers, NewHttpServer(addr, b.ServerKeyPath, b.ServerCertPath, b.ClientCAPath, handler))
	}
	return ret, nil
}

func (s *GenericServer) Done() <-chan struct{} {
	return s.done
}

func (s *GenericServer) Err() <-chan error {
	return s.err
}

func (s *GenericServer) Start() {
	var wg sync.WaitGroup
	wg.Add(len(s.HTTPServers))

	for i, hs := range s.HTTPServers {
		go func(_ int, hs *HTTPServer) {
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

func (s *GenericServer) Shutdown(ctx context.Context) error {
	var wg sync.WaitGroup
	var errs []error
	errch := make(chan error, len(s.HTTPServers))
	wg.Add(len(s.HTTPServers))

	// fan out shutdown commands to all servers
	for _, hs := range s.HTTPServers {
		go func(hs *HTTPServer) {
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

func (s *GenericServer) Close() error {
	var wg sync.WaitGroup
	var errs []error
	errch := make(chan error, len(s.HTTPServers))
	wg.Add(len(s.HTTPServers))

	// fan out close commands to all servers
	for _, hs := range s.HTTPServers {
		go func(hs *HTTPServer) {
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
