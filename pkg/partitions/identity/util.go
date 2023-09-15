package identity

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/x509"
	"io"

	"go.githedgehog.com/dasboot/pkg/devid"
)

var (
	ecdsaGenerateKey             func(c elliptic.Curve, rand io.Reader) (*ecdsa.PrivateKey, error)                         = ecdsa.GenerateKey
	x509MarshalECPrivateKey      func(key *ecdsa.PrivateKey) ([]byte, error)                                               = x509.MarshalECPrivateKey
	x509CreateCertificateRequest func(rand io.Reader, template *x509.CertificateRequest, priv any) (csr []byte, err error) = x509.CreateCertificateRequest
	devidID                      func() string                                                                             = devid.ID
)
