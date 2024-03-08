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
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"net/http"
	"os"
	"sync"
	"time"
)

var ErrNoCertsAdded = errors.New("HTTPServer: no certs added to Client CA Pool")

type HTTPServer struct {
	done           chan struct{}
	err            error
	clientCAPath   string
	serverKeyPath  string
	serverCertPath string
	tlsCfg         *tls.Config
	tlsCfgLock     sync.RWMutex
	srv            *http.Server
}

func (s *HTTPServer) Srv() *http.Server {
	return s.srv
}

func NewHttpServer(addr, serverKeyPath, serverCertPath, clientCAPath string, handler http.Handler) *HTTPServer {
	return &HTTPServer{
		done:           make(chan struct{}),
		clientCAPath:   clientCAPath,
		serverKeyPath:  serverKeyPath,
		serverCertPath: serverCertPath,
		srv: &http.Server{
			Addr: addr,
			// if a header cannot be read within 10s, then this should abort for sure
			ReadHeaderTimeout: 10 * time.Second,
			// the system is not using large POST bodies at this point, so this is a safe approach
			ReadTimeout: 30 * time.Second,
			// However, the system *is* writing out large request bodies because we serve installer artifacts
			// which can easily be >1GB.
			// That said, they also should be served within 5min, and not block the server for too long.
			WriteTimeout: 300 * time.Second,
			// Keep-Alives should not hold up a connection for more than 1.5 minutes
			IdleTimeout: 90 * time.Second,
			Handler:     handler,
		},
	}
}

// tlsConfig will always return an up to date version of the TLS config. This allows us to reload/redo
// TLS configuration and we will serve those immediately to the next connection.
func (s *HTTPServer) tlsConfig(*tls.ClientHelloInfo) (*tls.Config, error) {
	s.tlsCfgLock.RLock()
	defer s.tlsCfgLock.RUnlock()
	return s.tlsCfg, nil
}

func (s *HTTPServer) ReloadTLSConfig() error {
	// nothing to do if this is not a TLS server
	if s.serverKeyPath == "" {
		return nil
	}

	// load new cert and key
	cert, err := tls.LoadX509KeyPair(s.serverCertPath, s.serverKeyPath)
	if err != nil {
		return err
	}

	// and try to load a new client CA pool
	var clientCAPool *x509.CertPool

	if s.clientCAPath != "" {
		f, err := os.Open(s.clientCAPath)
		if err != nil {
			return err
		}
		defer f.Close()
		b, err := io.ReadAll(f)
		if err != nil {
			return err
		}
		clientCAPool = x509.NewCertPool()
		if !clientCAPool.AppendCertsFromPEM(b) {
			return ErrNoCertsAdded
		}
	}

	// and update the TLS config
	// this requires the write lock
	s.tlsCfgLock.Lock()
	defer s.tlsCfgLock.Unlock()
	s.tlsCfg = &tls.Config{
		MinVersion:         tls.VersionTLS12,
		ClientCAs:          clientCAPool,
		ClientAuth:         tls.VerifyClientCertIfGiven,
		Certificates:       []tls.Certificate{cert},
		GetConfigForClient: s.tlsConfig,
	}

	return nil
}

func (s *HTTPServer) Done() <-chan struct{} {
	return s.done
}

func (s *HTTPServer) Err() error {
	return s.err
}

func (s *HTTPServer) Start() {
	// make a TLS config
	// if we cannot make one at all, we need to abort on startup
	if err := s.ReloadTLSConfig(); err != nil {
		s.err = err
		close(s.done)
		return
	}

	// if this has a TLS config set at this point
	// it means that this needs to run an HTTPS server
	if s.tlsCfg != nil {
		s.srv.TLSConfig = s.tlsCfg
		go s.listenAndServeTLS()
		return
	}

	// otherwise we run a plain HTTP server
	go s.listenAndServe()
}

func (s *HTTPServer) listenAndServeTLS() {
	if err := s.srv.ListenAndServeTLS("", ""); err != nil {
		s.err = err
	}
	close(s.done)
}

func (s *HTTPServer) listenAndServe() {
	if err := s.srv.ListenAndServe(); err != nil {
		s.err = err
	}
	close(s.done)
}

func (s *HTTPServer) Shutdown(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}

func (s *HTTPServer) Close() error {
	return s.srv.Close()
}
