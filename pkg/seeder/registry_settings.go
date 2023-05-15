package seeder

import (
	"context"
	"crypto/ecdsa"
	"crypto/x509"

	"go.githedgehog.com/dasboot/pkg/seeder/config"
	"go.githedgehog.com/dasboot/pkg/seeder/controlplane"
	"go.githedgehog.com/dasboot/pkg/seeder/errors"
	"go.githedgehog.com/dasboot/pkg/seeder/registration"
)

func (s *seeder) initializeRegistrySettings(ctx context.Context, cfg *config.RegistrySettings, cpc controlplane.Client) error {
	var key *ecdsa.PrivateKey
	var cert *x509.Certificate
	if cfg != nil {
		if (cfg.KeyPath != "" && cfg.CertPath == "") || (cfg.CertPath != "" && cfg.KeyPath == "") {
			return errors.InvalidConfigError("client signing key and client signing cert must always be set together")
		}
		if cfg.KeyPath != "" {
			var err error
			key, err = readKeyFromPath(cfg.KeyPath)
			if err != nil {
				return err
			}
		}
		if cfg.CertPath != "" {
			var err error
			cert, _, err = readCertFromPath(cfg.CertPath)
			if err != nil {
				return err
			}
		}
	}

	s.registry = registration.NewProcessor(ctx, cpc, key, cert)

	return nil
}
