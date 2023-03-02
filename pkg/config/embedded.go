// The embedded config package is implementing an embedded config format so that the installer configuration can be "embedded" (or essentially appended)
// to the binary which is going to be executed. This allows the control plane (seeder) to serve specific configuration for a device.
//
// The layout / byte format can be best described in the following diagram:
//
// +------------------------------+------------------------------------------------------------------------------------------------------------------------------------------+
// |          ELF binary          |                                                        Embedded Config                                                                   |
// +------------------------------+-------------------+----------------------------------------------------------------------------------------------------------------------+
// |                              |      Content      |                                               Header                                                                 |
// +------------------------------+-------------------+-----------------------------+---------------------------------------------------+----------------+-------------------+
// | arm, arm64 or x86_64 version |    Config JSON    |        Content size         |     Signature (binary + Content + Size)           |     Version    | Config Magic Word |
// |  Original binary size bytes  | Config size bytes | uint32 Big Endian (4 bytes) | EC DSA (P-256) from SHA-256 hash (up to 73 bytes) | uint8 (1 byte) |      8 bytes      |
// +------------------------------+-------------------+-----------------------------+---------------------------------------------------+----------------+-------------------+
//
// The package implements both generating an embedded configuration for a binary, as well as reading and validating an embedded configuration.
package config

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"
)

// HeaderVersion represents a version of the embedded config format
type HeaderVersion uint8

// HeaderVersion1 represents version 1 of the embedded config format
const HeaderVersion1 HeaderVersion = 1

const (
	headerMagic         = "hedgehog"
	headerMagicSize     = len(headerMagic)
	headerVersionSize   = 1
	headerSignatureSize = 73 // An ECDSA signature on the P256 curve can be up to 73 bytes when DER encoded
	headerContentSize   = 4
	headerSize          = headerMagicSize + headerVersionSize + headerSignatureSize + headerContentSize
)

var (
	ErrExeTooSmall                  = errors.New("embedded config: executable not large enough to contain embedded config")
	ErrConfigTooLarge               = errors.New("embedded config: config JSON is too large")
	ErrInvalidKey                   = errors.New("embedded config: key is not an ECDSA P-256 key")
	ErrSignatureSize                = fmt.Errorf("embedded config: signature is exceeding %d bytes", headerSignatureSize)
	ErrConfigNotPresent             = errors.New("embedded config: config not present: magic marker missing")
	ErrInvalidConfigVersion         = errors.New("embedded config: invalid config version")
	ErrUnsupportedHeaderVersion     = errors.New("embedded config: unsupported header version")
	ErrUnsupportedSignatureKeyType  = errors.New("embedded config: unsupported signature key type")
	ErrSignatureVerificationFailure = errors.New("embedded config: signature verification failed")
)

var _ error = &ValidationError{}

// ValidationError will be returned by `GenerateExecutableWithEmbeddedConfig`
// or `ReadEmbeddedConfig` if the config fails to pass its `Validate()` function.
type ValidationError struct {
	// Err is the error as wrapped by
	Err error
}

// Error implements error
func (e *ValidationError) Error() string {
	return fmt.Sprintf("embedded config: validation: %s", e.Err)
}

// Unwrap implements the go 1.13 unwrap interface
func (e *ValidationError) Unwrap() error {
	return e.Err
}

func (e *ValidationError) Is(target error) bool {
	_, ok := target.(*ValidationError) //nolint: errorlint
	return ok
}

// UnsupportedConfigVersionError will be returned by `ReadEmbeddedConfig`
// if the config version of the read structure is not supported by the
// code which is trying to read the structure.
type UnsupportedConfigVersionError struct {
	Version ConfigVersion
}

func (e *UnsupportedConfigVersionError) Error() string {
	return fmt.Sprintf("embedded config: unsupported config version %d", e.Version)
}

func (e *UnsupportedConfigVersionError) Is(target error) bool {
	_, ok := target.(*UnsupportedConfigVersionError) //nolint: errorlint
	return ok
}

// EmbeddedConfig is the interface which all structs, which want to become embedded
// configuration structs, must implement.
// Essentially they must provide a validation function and a function which
// returns
type EmbeddedConfig interface {
	EmbeddedConfigVersion

	// Validate must ensure to validate the config settings for valid settings
	Validate() error

	// Cert must return the certificate with its public key which can be used
	// to verify the embedded signature of the config.
	// The certificate must be a DER encoded X509 certificate based on an
	// EC DSA key pair.
	Cert() []byte
}

// unit test overrides
var (
	timeNow          = time.Now
	cryptoRandReader = rand.Reader
	ecdsaSignASN1    = ecdsa.SignASN1
	keyUsages        = []x509.ExtKeyUsage{x509.ExtKeyUsageAny}
)

// GenerateExecutableWithEmbeddedConfig takes the bytes of an executable, a config structure and the
// private key for signing the exe+config and generates an executable with the config and signature embedded.
// The format of the executable is described in the package documentation.
func GenerateExecutableWithEmbeddedConfig(exe []byte, c EmbeddedConfig, key *ecdsa.PrivateKey) ([]byte, error) {
	// ensure the key is an ECDSA P-256 key
	if key.Curve != elliptic.P256() {
		return nil, ErrInvalidKey
	}

	// validate configuration version isn't invalid
	if cfgVer := c.ConfigVersion(); cfgVer <= 0 {
		return nil, ErrInvalidConfigVersion
	}

	// validate configuration
	if err := c.Validate(); err != nil {
		return nil, &ValidationError{Err: err}
	}

	// marshal it to JSON
	contentBytes, err := json.Marshal(c)
	if err != nil {
		return nil, fmt.Errorf("embedded config: JSON encoding: %w", err)
	}

	// ensure the configuration is not too big
	contentBytesSize := len(contentBytes)
	if contentBytesSize > math.MaxUint32 {
		return nil, ErrConfigTooLarge
	}

	// build the blob for signing
	// Note: ensure not to mess with the "exe" variable, hence the copy
	// the blob must contain:
	// - executable
	// - config JSON
	// - size of config JSON as uint32 (Big Endian)
	exeSize := len(exe)
	blob := make([]byte, exeSize, exeSize+headerSize)
	copy(blob, exe)
	blob = append(blob, contentBytes...)
	blob = binary.BigEndian.AppendUint32(blob, uint32(contentBytesSize))

	// create SHA-256 checksum from it
	cks := sha256.Sum256(blob)

	// create Signature
	signature, err := ecdsaSignASN1(cryptoRandReader, key, cks[:])
	if err != nil {
		return nil, fmt.Errorf("embedded config: ECDSA signature: %w", err)
	}

	sigLen := len(signature)
	if sigLen > headerSignatureSize {
		// this can never fail
		// for as long as the key size / curve does not change
		return nil, ErrSignatureSize
	}

	// pad with zeroes as necessary
	if sigLen < headerSignatureSize {
		for i := 0; i < headerSignatureSize-sigLen; i++ {
			signature = append(signature, 0x0)
		}
	}

	// now finish building the blob by adding:
	// - signature
	// - version
	// - magic word
	blob = append(blob, signature...)
	blob = append(blob, byte(HeaderVersion1))
	blob = append(blob, []byte(headerMagic)...)

	return blob, nil
}

// ReadOption represents read options that can be passed to `ReadEmbeddedConfig()`
type ReadOption uint

const (
	// ReadOptionUndefined should not be used
	ReadOptionUndefined ReadOption = iota

	// ReadOptionIgnoreExpiryTime instructs `ReadEmbeddedConfig()` to ignore the
	// potential failure that the certificate which was used for signing is expired.
	// This is particularly useful in cases when we read the embedded configuration,
	// but we cannot rely on the system time yet (aka stage0).
	ReadOptionIgnoreExpiryTime

	// ReadOptionIgnoreSignature instructs `ReadEmbeddedConfig()` to completely
	// ignore the signature verification. This will also skip certificate
	// verification.
	//
	// This is particularly useful in cases when an installer needs to be executed
	// manually outside of the automated execution context, and for whatever
	// reason signature verification would stand in the way.
	//
	// NOTE: As signature verification is not really a security mechanism here,
	// but rather a way to prevent the execution of an installer with a generated
	// configuration from an unexpected seeder, this is a legitimate use-case.
	ReadOptionIgnoreSignature
)

// ReadEmbeddedConfig will read the embedded configuration from `exe` and will store the configuration
// in the value pointed to by `config` and validate it. It is going to perform certificate verification of
// the certificate which is embedded in the `config`, and it is going to verify the embedded signature,
// unless this is disabled with the `ReadOptionIgnoreSignature` option.
func ReadEmbeddedConfig(exe []byte, config EmbeddedConfig, ca *x509.CertPool, opts ...ReadOption) error {
	// parse options
	var ignoreExpiryTime, ignoreSignature bool

	for _, opt := range opts {
		switch opt {
		case ReadOptionIgnoreExpiryTime:
			ignoreExpiryTime = true
		case ReadOptionIgnoreSignature:
			ignoreSignature = true
		}
	}

	// just a sanity check that the code below will not panic on logic
	exeSize := len(exe)
	if exeSize < headerSize {
		return ErrExeTooSmall
	}

	// check if the header magic is present where we expect it
	if string(exe[exeSize-headerMagicSize:]) != headerMagic {
		return ErrConfigNotPresent
	}

	// we only support version 1 right now, so abort in all other cases
	if HeaderVersion(exe[exeSize-headerMagicSize-headerVersionSize]) != HeaderVersion1 {
		return ErrUnsupportedHeaderVersion
	}

	// calculate the config content size
	headerStart := exeSize - headerSize
	contentBytesSize := binary.BigEndian.Uint32(exe[headerStart : headerStart+headerContentSize])

	contentStart := headerStart - int(contentBytesSize)
	if contentStart <= 0 {
		return ErrExeTooSmall
	}

	// get the config
	if err := json.Unmarshal(exe[contentStart:contentStart+int(contentBytesSize)], config); err != nil {
		return fmt.Errorf("embedded config: JSON decoding: %w", err)
	}

	// ensure config version is not an invalid value (which cannot be this way if it was generated with this package)
	cfgVer := config.ConfigVersion()
	if cfgVer <= 0 {
		return ErrInvalidConfigVersion
	}

	// validate config certificate against CA pool
	if !ignoreSignature {
		cert, err := x509.ParseCertificate(config.Cert())
		if err != nil {
			return fmt.Errorf("embedded config: parsing X509 signature certificate: %w", err)
		}
		if _, err := cert.Verify(x509.VerifyOptions{
			Intermediates: ca,
			Roots:         ca,
			KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
			CurrentTime:   timeNow(), // for unit testing
		}); err != nil {
			var certErr x509.CertificateInvalidError
			if errors.As(err, &certErr) && ignoreExpiryTime && certErr.Reason == x509.Expired {
				if _, err := cert.Verify(x509.VerifyOptions{
					Intermediates: ca,
					Roots:         ca,
					KeyUsages:     keyUsages, // for unit testing
					CurrentTime:   cert.NotBefore.Add(time.Second),
				}); err != nil {
					return fmt.Errorf("embedded config: signature certificate verification: %w", err)
				}
			} else {
				return fmt.Errorf("embedded config: signature certificate verification: %w", err)
			}
		}
		// TODO: also should check CRLs, and OCSP if given in cert

		// get the public key
		pubKey, ok := cert.PublicKey.(*ecdsa.PublicKey)
		if !ok {
			return ErrUnsupportedSignatureKeyType
		}

		// calculate SHA-256 checksum
		cks := sha256.Sum256(exe[:headerStart+headerContentSize])

		// verify signature
		sig := make([]byte, headerSignatureSize)
		copy(sig, exe[headerStart+headerContentSize:headerStart+headerContentSize+headerSignatureSize])
		sig = bytes.TrimRight(sig, "\x00") // remove padding
		if !ecdsa.VerifyASN1(pubKey, cks[:], sig) {
			return ErrSignatureVerificationFailure
		}
	}

	// validate configuration version
	if !config.IsSupportedConfigVersion(cfgVer) {
		return &UnsupportedConfigVersionError{Version: cfgVer}
	}

	// validate configuration
	if err := config.Validate(); err != nil {
		return &ValidationError{Err: err}
	}

	// success
	return nil
}
