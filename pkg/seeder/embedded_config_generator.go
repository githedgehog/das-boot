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
	"crypto/ecdsa"
	"crypto/x509"
	"errors"

	config "go.githedgehog.com/dasboot/pkg/config"
	confighhagentprov "go.githedgehog.com/dasboot/pkg/hhagentprov/config"
	config0 "go.githedgehog.com/dasboot/pkg/stage0/config"
	config1 "go.githedgehog.com/dasboot/pkg/stage1/config"
	config2 "go.githedgehog.com/dasboot/pkg/stage2/config"

	seederconfig "go.githedgehog.com/dasboot/pkg/seeder/config"
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

// Stage0 will generate an executable from the provided stage0 artifact and stage0 configuration.
// The caller does not need to set the `Version` and `SignatureCert` fields as they are being
// overwritten by this function.
func (ecg *embeddedConfigGenerator) Stage0(artifact []byte, cfg *config0.Stage0) ([]byte, error) {
	cfg.Version = 1
	cfg.SignatureCert = ecg.certDER
	return config.GenerateExecutableWithEmbeddedConfig(artifact, cfg, ecg.key)
}

// Stage1 will generate an executable from the provided stage1 artifact and stage1 configuration.
// The caller does not need to set the `Version` and `SignatureCert` fields as they are being
// overwritten by this function.
func (ecg *embeddedConfigGenerator) Stage1(artifact []byte, cfg *config1.Stage1) ([]byte, error) {
	cfg.Version = 1
	cfg.SignatureCert = ecg.certDER
	return config.GenerateExecutableWithEmbeddedConfig(artifact, cfg, ecg.key)
}

// Stage2 will generate an executable from the provided stage2 artifact and stage2 configuration.
// The caller does not need to set the `Version` and `SignatureCert` fields as they are being
// overwritten by this function.
func (ecg *embeddedConfigGenerator) Stage2(artifact []byte, cfg *config2.Stage2) ([]byte, error) {
	cfg.Version = 1
	cfg.SignatureCert = ecg.certDER
	return config.GenerateExecutableWithEmbeddedConfig(artifact, cfg, ecg.key)
}

func (ecg *embeddedConfigGenerator) HedgehogAgentProvisioner(artifact []byte, cfg *confighhagentprov.HedgehogAgentProvisioner) ([]byte, error) {
	cfg.Version = 1
	cfg.SignatureCert = ecg.certDER
	return config.GenerateExecutableWithEmbeddedConfig(artifact, cfg, ecg.key)
}

func (s *seeder) intializeEmbeddedConfigGenerator(c *seederconfig.EmbeddedConfigGeneratorConfig) error {
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
