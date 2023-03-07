package seeder

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"os"
)

var (
	ErrNotAnECDSAKey            = errors.New("certificate is not an ECDSA based cert")
	ErrPublicPrivateKeyMismatch = errors.New("private key does not match public key from certificate")
)

type embeddedConfigGenerator struct {
	key     *ecdsa.PrivateKey
	cert    *x509.Certificate
	certDER []byte
}

func (s *seeder) intializeEmbeddedConfigGenerator(c *EmbeddedConfigGeneratorConfig) error {
	// read key - expecting PEM format
	key, err := readKeyFromPath(c.KeyPath)
	if err != nil {
		return err
	}

	// read cert - expecting PEM format
	cert, certDER, err := readCertFromPath(c.CertPath)
	if err != nil {
		return err
	}

	// ensure the public keys match
	// if !reflect.DeepEqual()
	certPub, ok := cert.PublicKey.(*ecdsa.PublicKey)
	if !ok {
		return ErrNotAnECDSAKey
	}
	if certPub.X.Cmp(key.X) != 0 || certPub.Y.Cmp(key.Y) != 0 {
		return ErrPublicPrivateKeyMismatch
	}

	s.ecg = &embeddedConfigGenerator{
		key:     key,
		cert:    cert,
		certDER: certDER,
	}

	return nil
}

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
