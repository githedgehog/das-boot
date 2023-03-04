package seeder

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

var ErrNoCertsAdded = errors.New("httpServer: no certs added to Client CA Pool")

type httpServer struct {
	done           chan struct{}
	err            error
	clientCAPath   string
	serverKeyPath  string
	serverCertPath string
	tlsCfg         *tls.Config
	tlsCfgLock     sync.RWMutex
	srv            *http.Server
}

func newHttpServer(addr, serverKeyPath, serverCertPath, clientCAPath string, handler http.Handler) *httpServer {
	return &httpServer{
		done:           make(chan struct{}),
		clientCAPath:   clientCAPath,
		serverKeyPath:  serverKeyPath,
		serverCertPath: serverCertPath,
		srv: &http.Server{
			Addr:              addr,
			ReadTimeout:       30 * time.Second,
			WriteTimeout:      30 * time.Second,
			IdleTimeout:       90 * time.Second,
			ReadHeaderTimeout: 5 * time.Second,
			Handler:           handler,
		},
	}
}

// tlsConfig will always return an up to date version of the TLS config. This allows us to reload/redo
// TLS configuration and we will serve those immediately to the next connection.
func (s *httpServer) tlsConfig(*tls.ClientHelloInfo) (*tls.Config, error) {
	s.tlsCfgLock.RLock()
	defer s.tlsCfgLock.RUnlock()
	return s.tlsCfg, nil
}

func (s *httpServer) ReloadTLSConfig() error {
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

func (s *httpServer) Done() <-chan struct{} {
	return s.done
}

func (s *httpServer) Err() error {
	return s.err
}

func (s *httpServer) Start() {
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

func (s *httpServer) listenAndServeTLS() {
	if err := s.srv.ListenAndServeTLS("", ""); err != nil {
		s.err = err
	}
	close(s.done)
}

func (s *httpServer) listenAndServe() {
	if err := s.srv.ListenAndServe(); err != nil {
		s.err = err
	}
	close(s.done)
}

func (s *httpServer) Shutdown(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}

func (s *httpServer) Close() error {
	return s.srv.Close()
}