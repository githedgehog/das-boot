package seeder

import (
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
	clientCAPath   string
	serverKeyPath  string
	serverCertPath string
	tlsCfg         *tls.Config
	tlsCfgLock     sync.RWMutex
	srv            *http.Server
}

func newHttpServer(addr, serverKeyPath, serverCertPath, clientCAPath string, handler http.Handler) *httpServer {
	ret := &httpServer{
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
	ret.srv.TLSConfig = &tls.Config{
		// MinVersion is of no consequence here, but the gosec linter complains, so whatever
		MinVersion:         tls.VersionTLS12,
		GetConfigForClient: ret.tlsConfig,
	}
	return ret
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
