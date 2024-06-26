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

package config

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"

	// not used for security
	"crypto/sha1" //nolint: gosec
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	mathrand "math/rand"
	"reflect"
	"strings"
	"testing"
	"time"
)

func generateTestKeyMaterial(curve elliptic.Curve) (key *ecdsa.PrivateKey, cert []byte, caPool *x509.CertPool, caKey *ecdsa.PrivateKey, caCert *x509.Certificate) { //nolint: unparam
	var err error

	// create CA
	caKey, err = ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		panic(err)
	}
	// not used for security purposes
	caPubKey, err := caKey.PublicKey.ECDH()
	if err != nil {
		panic(err)
	}
	caKeyID := sha1.Sum(caPubKey.Bytes()) //nolint: gosec
	caTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2019),
		Subject: pkix.Name{
			CommonName: "Installer Signing Root CA",
		},
		SubjectKeyId:          caKeyID[:],
		AuthorityKeyId:        caKeyID[:],
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour * 24 * 360),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	caCertDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		panic(fmt.Errorf("failed to generate CA certificate: %w", err))
	}
	caCert, err = x509.ParseCertificate(caCertDER)
	if err != nil {
		panic(fmt.Errorf("failed to parse CA certificate: %w", err))
	}
	caPool = x509.NewCertPool()
	caPool.AddCert(caCert)

	// create cert for signing which is signed by CA
	key, err = ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		panic(err)
	}

	csrTemplate := &x509.CertificateRequest{
		PublicKey: key.PublicKey,
		Subject: pkix.Name{
			CommonName: "installer signing cert",
		},
	}
	csrBytes, err := x509.CreateCertificateRequest(rand.Reader, csrTemplate, key)
	if err != nil {
		panic(fmt.Errorf("failed to create CSR: %w", err))
	}
	csr, err := x509.ParseCertificateRequest(csrBytes)
	if err != nil {
		panic(fmt.Errorf("failed to parse CSR: %w", err))
	}
	csrPub := csr.PublicKey.(*ecdsa.PublicKey)
	ecdhCsrPub, err := csrPub.ECDH()
	if err != nil {
		panic(fmt.Errorf("failed to convert ECDSA public key to ECDH public key: %w", err))
	}

	// not used for security purposes
	subjectKeyId := sha1.Sum(ecdhCsrPub.Bytes()) //nolint: gosec
	certTemplate := &x509.Certificate{
		// not used for security purposes
		SerialNumber: big.NewInt(mathrand.Int63()), //nolint: gosec
		Subject:      csr.Subject,
		SubjectKeyId: subjectKeyId[:],
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour * 24 * 360),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	cert, err = x509.CreateCertificate(rand.Reader, certTemplate, caCert, csr.PublicKey, caKey)
	if err != nil {
		panic(fmt.Errorf("certificate signing failed: %w", err))
	}

	// sanity check to see that this initialization works
	tmpCert, err := x509.ParseCertificate(cert)
	if err != nil {
		panic(fmt.Errorf("failed to parse signing certificate: %w", err))
	}
	chains, err := tmpCert.Verify(x509.VerifyOptions{
		Intermediates: caPool,
		Roots:         caPool,
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	})
	if err != nil {
		panic(fmt.Errorf("failed to verify signing certificate: %w", err))
	}
	if len(chains) != 1 {
		panic(fmt.Errorf("verification chain has an unexpected length: %d != 1", len(chains)))
	}

	return
}

func generateRSAKeyAndCertAndAddToPool(caPool *x509.CertPool) []byte {
	// generate new CA
	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	type pkcs1PublicKey struct {
		N *big.Int
		E int
	}

	caPublicKeyBytes, err := asn1.Marshal(pkcs1PublicKey{
		N: caKey.PublicKey.N,
		E: caKey.PublicKey.E,
	})
	if err != nil {
		panic(err)
	}
	// not used for security purposes
	caKeyID := sha1.Sum(caPublicKeyBytes) //nolint: gosec
	caTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2019),
		Subject: pkix.Name{
			CommonName: "Installer Signing Root CA",
		},
		SubjectKeyId:          caKeyID[:],
		AuthorityKeyId:        caKeyID[:],
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour * 24 * 360),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	caCertDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		panic(fmt.Errorf("failed to generate CA certificate: %w", err))
	}
	caCert, err := x509.ParseCertificate(caCertDER)
	if err != nil {
		panic(fmt.Errorf("failed to parse CA certificate: %w", err))
	}
	caPool.AddCert(caCert)

	// cert for signing
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}

	csrTemplate := &x509.CertificateRequest{
		PublicKey: key.PublicKey,
		Subject: pkix.Name{
			CommonName: "installer signing cert",
		},
	}
	csrBytes, err := x509.CreateCertificateRequest(rand.Reader, csrTemplate, key)
	if err != nil {
		panic(fmt.Errorf("failed to create CSR: %w", err))
	}
	csr, err := x509.ParseCertificateRequest(csrBytes)
	if err != nil {
		panic(fmt.Errorf("failed to parse CSR: %w", err))
	}
	csrPub := csr.PublicKey.(*rsa.PublicKey)

	publicKeyBytes, err := asn1.Marshal(pkcs1PublicKey{
		N: csrPub.N,
		E: csrPub.E,
	})
	if err != nil {
		panic(err)
	}

	// not used for security purposes
	subjectKeyId := sha1.Sum(publicKeyBytes) //nolint: gosec
	certTemplate := &x509.Certificate{
		// not used for security purposes
		SerialNumber: big.NewInt(mathrand.Int63()), //nolint: gosec
		Subject:      csr.Subject,
		SubjectKeyId: subjectKeyId[:],
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour * 24 * 360),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	cert, err := x509.CreateCertificate(rand.Reader, certTemplate, caCert, csr.PublicKey, caKey)
	if err != nil {
		panic(fmt.Errorf("certificate signing failed: %w", err))
	}

	// sanity check to see that this initialization works
	tmpCert, err := x509.ParseCertificate(cert)
	if err != nil {
		panic(fmt.Errorf("failed to parse signing certificate: %w", err))
	}
	chains, err := tmpCert.Verify(x509.VerifyOptions{
		Intermediates: caPool,
		Roots:         caPool,
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	})
	if err != nil {
		panic(fmt.Errorf("failed to verify signing certificate: %w", err))
	}
	if len(chains) != 1 {
		panic(fmt.Errorf("verification chain has an unexpected length: %d != 1", len(chains)))
	}
	return cert
}

var _ EmbeddedConfig = &configTest{}

type configTest struct {
	Field1        string        `json:",omitempty"`
	Field2        int           `json:",omitempty"`
	SignatureCert []byte        `json:",omitempty"`
	Version       ConfigVersion `json:",omitempty"`
}

// IsSupportedConfigVersion implements EmbeddedConfig
func (*configTest) IsSupportedConfigVersion(v ConfigVersion) bool {
	return v == 1
}

// Version implements EmbeddedConfig
func (c *configTest) ConfigVersion() ConfigVersion {
	return c.Version
}

// Cert implements EmbeddedConfig
func (c *configTest) Cert() []byte {
	return c.SignatureCert
}

// Validate implements EmbeddedConfig
func (c *configTest) Validate() error {
	if c.Field1 == "" {
		return fmt.Errorf("Field1 is empty")
	}

	if c.Field2 <= 0 || c.Field2 > 16 {
		return fmt.Errorf("invalid value for Field2: %d", c.Field2)
	}
	return nil
}

var _ EmbeddedConfig = &configTestFailValidate{}

type configTestFailValidate struct {
	Field1        string        `json:",omitempty"`
	Field2        int           `json:",omitempty"`
	SignatureCert []byte        `json:",omitempty"`
	Version       ConfigVersion `json:",omitempty"`
}

// IsSupportedConfigVersion implements EmbeddedConfig
func (*configTestFailValidate) IsSupportedConfigVersion(v ConfigVersion) bool {
	return v == 1
}

// Version implements EmbeddedConfig
func (c *configTestFailValidate) ConfigVersion() ConfigVersion {
	return c.Version
}

// Cert implements EmbeddedConfig
func (c *configTestFailValidate) Cert() []byte {
	return c.SignatureCert
}

// Validate implements EmbeddedConfig
func (c *configTestFailValidate) Validate() error {
	return fmt.Errorf("always fail validate")
}

var _ EmbeddedConfig = &configTestCert{}

type configTestCert struct {
	Field1       string        `json:",omitempty"`
	Field2       int           `json:",omitempty"`
	OverrideCert []byte        `json:"-"`
	Version      ConfigVersion `json:",omitempty"`
}

// IsSupportedConfigVersion implements EmbeddedConfig
func (*configTestCert) IsSupportedConfigVersion(v ConfigVersion) bool {
	return v == 1
}

// Version implements EmbeddedConfig
func (c *configTestCert) ConfigVersion() ConfigVersion {
	return c.Version
}

// Cert implements EmbeddedConfig
func (c *configTestCert) Cert() []byte {
	return c.OverrideCert
}

// Validate implements EmbeddedConfig
func (c *configTestCert) Validate() error {
	return nil
}

var _ EmbeddedConfig = &invalidConfigTest{}

type invalidConfigTest struct {
	Field1 chan struct{} `json:",omitempty"`
}

// IsSupportedConfigVersion implements EmbeddedConfig
func (*invalidConfigTest) IsSupportedConfigVersion(v ConfigVersion) bool {
	return true
}

// Version implements EmbeddedConfig
func (c *invalidConfigTest) ConfigVersion() ConfigVersion {
	return 1
}

// Cert implements EmbeddedConfig
func (*invalidConfigTest) Cert() []byte {
	panic("should never reach Cert() function")
}

// Validate implements EmbeddedConfig
func (*invalidConfigTest) Validate() error {
	// yeah, looks just great
	return nil
}

var _ io.Reader = &failReader{}

type failReader struct{}

var errCSRNGReadFailure = errors.New("CRSNG Read Failure")

func (*failReader) Read(p []byte) (n int, err error) {
	return 0, errCSRNGReadFailure
}

func TestGenerateExecutableWithEmbeddedConfig(t *testing.T) {
	key, _, _, _, _ := generateTestKeyMaterial(elliptic.P256())
	if key == nil {
		panic("generateTestKeyMaterial is broken")
	}

	invalidKey, _, _, _, _ := generateTestKeyMaterial(elliptic.P384())

	var exe = []byte("I'm a binary")

	type args struct {
		exe []byte
		c   EmbeddedConfig
		key *ecdsa.PrivateKey
	}
	tests := []struct {
		name             string
		args             args
		wantErr          bool
		wantErrToBe      error
		skip             bool
		cryptoRandReader io.Reader
		ecdsaSignASN1    func(rand io.Reader, priv *ecdsa.PrivateKey, hash []byte) ([]byte, error)
	}{
		{
			name: "success",
			args: args{
				key: key,
				exe: exe,
				c: &configTest{
					Field1:  "I'm not empty",
					Field2:  8,
					Version: 1,
				},
			},
		},
		{
			name: "invalid configuration version",
			args: args{
				key: key,
				exe: exe,
				c: &configTest{
					Field1:  "still valid",
					Field2:  17,
					Version: 0,
				},
			},
			wantErr:     true,
			wantErrToBe: ErrInvalidConfigVersion,
		},
		{
			name: "validation fails",
			args: args{
				key: key,
				exe: exe,
				c: &configTest{
					Field1:  "still valid",
					Field2:  17,
					Version: 1,
				},
			},
			wantErr:     true,
			wantErrToBe: &ValidationError{},
		},
		{
			name: "config invalid for JSON marshaling",
			args: args{
				key: key,
				exe: exe,
				c:   &invalidConfigTest{},
			},
			wantErr: true,
		},
		{
			// don't rename, or fix the hack below in the execution
			name: "config too large in size",
			args: args{
				key: key,
				exe: exe,
				// c is being set in the hack below
			},
			// skipping this test if the race detector is enabled,
			// as this is taking too long otherwise (and has no race obviously)
			// skip:        version.RaceEnabled,
			skip:        true,
			wantErr:     true,
			wantErrToBe: ErrConfigTooLarge,
		},
		{
			name: "signing error",
			args: args{
				key: key,
				exe: exe,
				c: &configTest{
					Field1:  "valid",
					Field2:  2,
					Version: 1,
				},
			},
			cryptoRandReader: &failReader{},
			wantErr:          true,
		},
		{
			name: "invalid key error",
			args: args{
				key: invalidKey,
				exe: exe,
				c: &configTest{
					Field1:  "I'm not empty",
					Field2:  8,
					Version: 1,
				},
			},
			wantErr:     true,
			wantErrToBe: ErrInvalidKey,
		},
		{
			name: "invalid signature size",
			args: args{
				key: key,
				exe: exe,
				c: &configTest{
					Field1:  "I'm not empty",
					Field2:  8,
					Version: 1,
				},
			},
			ecdsaSignASN1: func(rand io.Reader, priv *ecdsa.PrivateKey, hash []byte) ([]byte, error) {
				b, err := ecdsa.SignASN1(rand, priv, hash)
				if err != nil {
					return nil, err
				}
				// add some bytes to mess with it
				b = append(b, []byte{0x1, 0x2, 0x3, 0x4, 0x5}...)
				return b, nil
			},
			wantErr:     true,
			wantErrToBe: ErrSignatureSize,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skip {
				t.Skipf("Skipping test %q as requested...", tt.name)
				return
			}
			// ugly hack, but otherwise this is slowing down the tests with race detector
			if tt.name == "config too large in size" {
				tt.args.c = &configTest{
					Field1:  strings.Repeat("a", math.MaxUint32),
					Field2:  1,
					Version: 1,
				}
			}
			if tt.cryptoRandReader != nil {
				oldCryptoRandReader := cryptoRandReader
				cryptoRandReader = tt.cryptoRandReader
				defer func() {
					cryptoRandReader = oldCryptoRandReader
				}()
			}
			if tt.ecdsaSignASN1 != nil {
				oldEcdsaSignASN1 := ecdsaSignASN1
				ecdsaSignASN1 = tt.ecdsaSignASN1
				defer func() {
					ecdsaSignASN1 = oldEcdsaSignASN1
				}()
			}
			_, err := GenerateExecutableWithEmbeddedConfig(tt.args.exe, tt.args.c, tt.args.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateEmbeddedConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr && tt.wantErrToBe != nil {
				if !errors.Is(err, tt.wantErrToBe) {
					t.Errorf("ReadEmbeddedConfig() error = %v, wantErrToBe %v", err, tt.wantErrToBe)
				}
			}
			// TODO: this is difficult with the changing signature
			// if !reflect.DeepEqual(got, tt.want) {
			// 	t.Errorf("GenerateEmbeddedConfig() = %v, want %v", got, tt.want)
			// }
		})
	}
}

func TestReadEmbeddedConfig(t *testing.T) {
	key, cert, caPool, _, _ := generateTestKeyMaterial(elliptic.P256())
	if key == nil || cert == nil || caPool == nil {
		panic("generateTestKeyMaterial is broken")
	}
	rsaCert := generateRSAKeyAndCertAndAddToPool(caPool)

	// generate valid embedded config
	origCfg := &configTest{
		Field1:        "I'm not empty",
		Field2:        8,
		SignatureCert: cert,
		Version:       1,
	}
	exeOnly := []byte("I'm a binary")
	exe, err := GenerateExecutableWithEmbeddedConfig(exeOnly, origCfg, key)
	if err != nil {
		panic("GenerateEmbeddedConfig is broken")
	}

	// generate valid embedded config that has an unsupported config version
	// for unsupported config version test
	cfgUnsupported := &configTest{
		Field1:        "I'm not empty",
		Field2:        8,
		SignatureCert: cert,
		Version:       2,
	}
	exeUnsupportedConfig, err := GenerateExecutableWithEmbeddedConfig(exeOnly, cfgUnsupported, key)
	if err != nil {
		panic("GenerateEmbeddedConfig is broken")
	}

	// for invalid signature test
	exeWrongSignature := make([]byte, len(exe))
	copy(exeWrongSignature, exe)
	exeWrongSignature[len(exeWrongSignature)-headerMagicSize-headerVersionSize-5] = exeWrongSignature[len(exeWrongSignature)-headerMagicSize-headerVersionSize-5] + 1

	// for invalid config version test
	exeInvalidConfigVersion := make([]byte, len(exe))
	copy(exeInvalidConfigVersion, exe)
	exeInvalidConfigVersion[len(exeInvalidConfigVersion)-headerSize-2] = '0'

	type args struct {
		exe  []byte
		ca   *x509.CertPool
		opts []ReadOption
	}

	tests := []struct {
		name                string
		args                args
		wantErr             bool
		wantErrToBe         error
		certVerifyTime      func() time.Time
		certVerifyKeyUsages []x509.ExtKeyUsage
		testCfg             EmbeddedConfig
		wantCfg             *configTest
	}{
		{
			name: "success",
			args: args{
				exe: exe,
				ca:  caPool,
			},
			testCfg: &configTest{},
			wantCfg: origCfg,
		},
		{
			name: "exe too small",
			args: args{
				exe: []byte("too small"),
			},
			wantErr:     true,
			wantErrToBe: ErrExeTooSmall,
		},
		{
			name: "magic header not present",
			args: args{
				exe: bytes.Repeat([]byte{0x42}, headerSize+42),
			},
			wantErr:     true,
			wantErrToBe: ErrConfigNotPresent,
		},
		{
			name: "unsupported header version",
			args: args{
				exe: append(bytes.Repeat([]byte{0x42}, headerSize+1), []byte("hedgehog")...),
			},
			wantErr:     true,
			wantErrToBe: ErrUnsupportedHeaderVersion,
		},
		{
			name: "exe not long enough for config",
			args: args{
				exe: append(bytes.Repeat([]byte{0x42}, headerSize), append([]byte{0x1}, []byte("hedgehog")...)...),
			},
			wantErr:     true,
			wantErrToBe: ErrExeTooSmall,
		},
		{
			name: "invalid config structure",
			args: args{
				exe: exe,
			},
			testCfg: &invalidConfigTest{},
			wantErr: true,
		},
		{
			name: "invalid config version",
			args: args{
				exe: exeInvalidConfigVersion,
				ca:  caPool,
			},
			testCfg:     &configTest{},
			wantErr:     true,
			wantErrToBe: ErrInvalidConfigVersion,
		},
		{
			name: "invalid config version",
			args: args{
				exe: exeUnsupportedConfig,
				ca:  caPool,
			},
			testCfg:     &configTest{},
			wantErr:     true,
			wantErrToBe: &UnsupportedConfigVersionError{},
		},
		{
			name: "fail config validation",
			args: args{
				exe: exe,
				ca:  caPool,
			},
			testCfg:     &configTestFailValidate{},
			wantErr:     true,
			wantErrToBe: &ValidationError{},
		},
		{
			name: "fail parsing certificate",
			args: args{
				exe: exe,
			},
			testCfg: &configTestCert{},
			wantErr: true,
		},
		{
			name: "fail certificate verification",
			args: args{
				exe: exe,
			},
			testCfg: &configTest{},
			wantErr: true,
		},
		{
			name: "fail certificate verification because of expiry",
			args: args{
				exe: exe,
				ca:  caPool,
			},
			testCfg: &configTest{},
			wantErr: true,
			certVerifyTime: func() time.Time {
				return time.Now().Add(time.Hour * 24 * 3600)
			},
		},
		{
			name: "succeed certificate verification because of expiry if option is set",
			args: args{
				exe:  exe,
				ca:   caPool,
				opts: []ReadOption{ReadOptionIgnoreExpiryTime},
			},
			testCfg: &configTest{},
			wantCfg: origCfg,
			certVerifyTime: func() time.Time {
				return time.Now().Add(time.Hour * 24 * 3600)
			},
		},
		{
			name: "fail certificate verification if expiry option is set, but verification fails for other reason",
			args: args{
				exe:  exe,
				ca:   caPool,
				opts: []ReadOption{ReadOptionIgnoreExpiryTime},
			},
			testCfg: &configTest{},
			wantErr: true,
			certVerifyTime: func() time.Time {
				return time.Now().Add(time.Hour * 24 * 3600)
			},
			certVerifyKeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageEmailProtection},
		},
		{
			name: "fail if cert is RSA key based",
			args: args{
				exe: exe,
				ca:  caPool,
			},
			testCfg:     &configTestCert{OverrideCert: rsaCert},
			wantErr:     true,
			wantErrToBe: ErrUnsupportedSignatureKeyType,
		},
		{
			name: "success if signature verification is disabled",
			args: args{
				exe:  exe,
				opts: []ReadOption{ReadOptionIgnoreSignature},
			},
			testCfg: &configTest{},
			wantCfg: origCfg,
		},
		{
			name: "fail signature verification",
			args: args{
				exe: exeWrongSignature,
				ca:  caPool,
			},
			testCfg:     &configTest{},
			wantErr:     true,
			wantErrToBe: ErrSignatureVerificationFailure,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.certVerifyTime != nil {
				oldTimeNow := timeNow
				timeNow = tt.certVerifyTime
				defer func() {
					timeNow = oldTimeNow
				}()
			}
			if len(tt.certVerifyKeyUsages) > 0 {
				oldKeyUsages := keyUsages
				keyUsages = tt.certVerifyKeyUsages
				defer func() {
					keyUsages = oldKeyUsages
				}()
			}
			err := ReadEmbeddedConfig(tt.args.exe, tt.testCfg, tt.args.ca, tt.args.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadEmbeddedConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.wantErr && tt.wantErrToBe != nil {
				if !errors.Is(err, tt.wantErrToBe) {
					t.Errorf("ReadEmbeddedConfig() error = %v, wantErrToBe %v", err, tt.wantErrToBe)
				}
			}
			if err == nil {
				if !reflect.DeepEqual(tt.testCfg, tt.wantCfg) {
					t.Errorf("ReadEmbeddedConfig() cfg = %v, want %v", tt.testCfg, tt.wantCfg)
					return
				}
			}
		})
	}
}

func TestValidationError(t *testing.T) {
	// invalid config
	cfg := &configTest{
		Field1: "",
		Field2: 0,
	}
	origErr := cfg.Validate()
	if origErr == nil {
		panic("configTest is broken")
	}
	origErr = &ValidationError{Err: origErr}

	tests := []struct {
		name                string
		err                 error
		wantValidationError bool
	}{
		{
			name:                "single",
			err:                 origErr,
			wantValidationError: true,
		},
		{
			name:                "wrapped",
			err:                 fmt.Errorf("wrap the validation error: %w", origErr),
			wantValidationError: true,
		},
		{
			name:                "different",
			err:                 errors.New("different error"),
			wantValidationError: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValidationErr := errors.Is(tt.err, &ValidationError{})
			if isValidationErr != tt.wantValidationError {
				t.Errorf("%v is not a ValidationError (isValidationErr %t != tt.wantValidationError %t)", tt.err, isValidationErr, tt.wantValidationError)
			}
		})
	}

	// ensure unwrap also works as expected
	t.Run("unwrap", func(t *testing.T) {
		wrappedErr := errors.New("wrapped error")
		err := &ValidationError{Err: wrappedErr}

		if !errors.Is(err, wrappedErr) {
			t.Errorf("wrapping errors isn't working properly")
		}
	})
}

func TestUnsupportedConfigVersionError(t *testing.T) {
	origErr := &UnsupportedConfigVersionError{Version: 1}

	tests := []struct {
		name                              string
		err                               error
		wantUnsupportedConfigVersionError bool
	}{
		{
			name:                              "single",
			err:                               origErr,
			wantUnsupportedConfigVersionError: true,
		},
		{
			name:                              "wrapped",
			err:                               fmt.Errorf("wrap the validation error: %w", origErr),
			wantUnsupportedConfigVersionError: true,
		},
		{
			name:                              "different",
			err:                               errors.New("different error"),
			wantUnsupportedConfigVersionError: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isUnsupportedConfigVersionErr := errors.Is(tt.err, &UnsupportedConfigVersionError{})
			if isUnsupportedConfigVersionErr != tt.wantUnsupportedConfigVersionError {
				t.Errorf("%v is not an UnuspportedConfigVersionError (isUnsupportedConfigVersionErr %t != tt.wantUnsupportedConfigVersionError %t)", tt.err, isUnsupportedConfigVersionErr, tt.wantUnsupportedConfigVersionError)
			}
		})
	}
}
