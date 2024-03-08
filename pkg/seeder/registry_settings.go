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
