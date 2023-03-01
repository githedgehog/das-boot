package identity

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/x509"
	"io"

	"go.uber.org/zap"
)

var Logger = zap.L().With(zap.String("logger", "pkg/partitions/identity"))

var (
	ecdsaGenerateKey        func(c elliptic.Curve, rand io.Reader) (*ecdsa.PrivateKey, error) = ecdsa.GenerateKey
	x509MarshalECPrivateKey func(key *ecdsa.PrivateKey) ([]byte, error)                       = x509.MarshalECPrivateKey
)
