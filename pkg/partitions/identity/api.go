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

package identity

import (
	"crypto/tls"
	"crypto/x509"
	"errors"

	"go.githedgehog.com/dasboot/pkg/partitions/location"
)

const (
	version1 int = 1

	versionFilePath         = "/version"
	identityDirPath         = "/identity"
	locationDirPath         = "/location"
	clientKeyPath           = identityDirPath + "/client.key"
	clientCSRPath           = identityDirPath + "/client.csr"
	clientCertPath          = identityDirPath + "/client.crt"
	tpmPrimaryCtxPath       = identityDirPath + "/primary.tpm.ctx"
	tpmClientPubPath        = identityDirPath + "/client.tpm.pub"
	tpmClientPrivPath       = identityDirPath + "/client.tpm.priv"
	locationUUIDPath        = locationDirPath + "/uuid"
	locationUUIDSigPath     = locationDirPath + "/uuid.sig"
	locationMetadataPath    = locationDirPath + "/metadata"
	locationMetadataSigPath = locationDirPath + "/metadata.sig"
)

// Version is the contents of the version file.
type Version struct {
	// Version is the version number of the partition format. This field
	// must always be present.
	Version int `json:"version"`
}

type IdentityPartition interface {
	// HasClientKey tests if the partition already holds a valid client key. The implementation needs to validate
	// that the key on disk (or TPM) is in fact valid.
	HasClientKey() bool

	// HasClientCSR tests if the partition already holds a valid client certificate request. The implementation needs
	// to validate that the CSR on disk is in fact valid, and optionally should check if the embedded public key belongs
	// to the key on disk (or TPM).
	HasClientCSR() bool

	// HasClientCert tests if the partition already holds a certificate. The implementation MUST NOT validate that
	// the certificate on disk is in fact valid. Use `HasValidClientCert()` for that. It simply needs to check if there
	// is a parseable certificate on disk.
	HasClientCert() bool

	// HasValidClientCert tests if the partition already holds a valid certificate. The implementation needs to validate that
	// the certificate on disk is in fact valid, and optionally should check if the embedded public key belongs to the
	// key on disk (or TPM).
	HasValidClientCert() bool

	// MatchesClientCertificate tests if the provided certificate matches the client certificate on disk.
	// If there is no client certificate on disk, it must return false. Note that this call does not verify the validity
	// of the certificate on disk, it will
	MatchesClientCertificate(cert *x509.Certificate) bool

	// GenerateClientKeyPair generates a new key client key pair. It must overwrite any existing keys on disk (or TPM).
	// Therefore a call to `HasClientKey` is recommended if overwriting would not be the intention. Subsequently, it must
	// delete any already existing CSR and certificate on disk if they exist. If there is an error deleting already
	// existing CSR or certificate, it must return an error.
	GenerateClientKeyPair() error

	// GenerateClientCSR generates a new CSR using the key pair on disk (or TPM). It overwrites any existing CSR on disk.
	// Therefore a call to `HasClientCSR` is recommended if overwriting would not be the intention. The returned CSR is
	// in DER encoded format. Subsequently, it must delete any already existing certificate on disk if it exists. If there
	// is an error deleting the already existing certificate, it must return an error.
	GenerateClientCSR() ([]byte, error)

	// ReadClientCSR reads the client CSR from the partition. It fails if it does not exist yet, in which case the caller
	// should call `GenerateClientCSR` first. The returned CSR is in DER encoded format.
	ReadClientCSR() ([]byte, error)

	// StoreClientCert stores a certificate to disk which is passed in the argument in DER encoding.
	StoreClientCert([]byte) error

	// LoadX509KeyPair loads the key from the partition (or TPM) and the certificate from the partition and returns a
	// TLS certificate which is ready to be used in a TLS config as a client certificate.
	LoadX509KeyPair() (tls.Certificate, error)

	// GetLocation reads the location information from the partition if it was previously stored, and returns an error
	// otherwise.
	GetLocation() (*location.Info, error)

	// StoreLocation stores the location information to disk on the identity partition. It is going to overwrite any
	// existing location information on disk if it already exists.
	StoreLocation(*location.Info) error

	// CopyLocation copies the location information from a location partition and stores it in the identity partition.
	// It is going to overwrite existing location information on disk if it already exists. The implementation may call
	// internally `StoreLocation` to persist the information onto the disk.
	CopyLocation(location.LocationPartition) error
}

var (
	ErrWrongDevice            = errors.New("identity: not the identity partition")
	ErrNotMounted             = errors.New("identity: partition not mounted")
	ErrUnsupportedVersion     = errors.New("identity: unsupported identity partition version")
	ErrUninitializedPartition = errors.New("identity: partition uninitialized")
	ErrAlreadyInitialized     = errors.New("identity: partition already initialized")
	ErrNoPEMData              = errors.New("identity: no PEM data")
	ErrNoDevID                = errors.New("identity: no device ID")
)
