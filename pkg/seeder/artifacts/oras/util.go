package oras

import (
	"crypto/tls"
	"crypto/x509"
	"io"
	"os"
)

func caPool(path string) *x509.CertPool {
	if path != "" {
		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()
		b, err := io.ReadAll(f)
		if err != nil {
			return nil
		}
		ret := x509.NewCertPool()
		if !ret.AppendCertsFromPEM(b) {
			return nil
		}
		return ret
	}
	return nil
}

func clientCertificates(certPath, keyPath string) []tls.Certificate {
	tlsCert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil
	}
	return []tls.Certificate{tlsCert}
}
