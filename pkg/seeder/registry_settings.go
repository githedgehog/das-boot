package seeder

import (
	"context"
	"crypto/ecdsa"
	"crypto/x509"

	"go.githedgehog.com/dasboot/pkg/seeder/registration"
)

func (s *seeder) initializeRegistrySettings(ctx context.Context, config *RegistrySettings) error {
	var key *ecdsa.PrivateKey
	var cert *x509.Certificate
	if config != nil {
		if (config.KeyPath != "" && config.CertPath == "") || (config.CertPath != "" && config.KeyPath == "") {
			return invalidConfigError("client signing key and client signing cert must always be set together")
		}
		if config.KeyPath != "" {
			var err error
			key, err = readKeyFromPath(config.KeyPath)
			if err != nil {
				return err
			}
		}
		if config.CertPath != "" {
			var err error
			cert, _, err = readCertFromPath(config.CertPath)
			if err != nil {
				return err
			}
		}
	}

	s.registry = registration.NewProcessor(ctx, nil, key, cert)

	return nil
}
