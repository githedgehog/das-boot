package main

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"os"
)

func readKeyFromPath(path string) (*ecdsa.PrivateKey, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open '%s': %w", path, err)
	}
	b, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("reading '%s': %w", path, err)
	}
	p, _ := pem.Decode(b)
	key, err := x509.ParseECPrivateKey(p.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parsing key '%s': %w", path, err)
	}
	return key, nil
}

func readCertFromPath(path string) (*x509.Certificate, []byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("open '%s': %w", path, err)
	}
	b, err := io.ReadAll(f)
	if err != nil {
		return nil, nil, fmt.Errorf("reading '%s': %w", path, err)
	}
	p, _ := pem.Decode(b)
	cert, err := x509.ParseCertificate(p.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing certficate '%s': %w", path, err)
	}
	return cert, p.Bytes, nil
}
