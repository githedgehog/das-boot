package identity

import (
	"crypto/tls"
	"crypto/x509"
	"errors"

	"go.githedgehog.com/dasboot/pkg/partitions/location"
)

const (
	version1 int64 = 1

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

type IdentityPartition interface {
	HasClientKey() bool
	HasClientCSR() bool
	HasClientCert() bool
	GenerateClientKeyPair() error
	GenerateClientCSR() (*x509.CertificateRequest, error)
	ReadClientCSR() (*x509.CertificateRequest, error)
	StoreClientCert([]byte) error
	LoadX509KeyPair() (tls.Certificate, error)
	GetLocation() (*location.Info, error)
}

var (
	ErrWrongDevice            = errors.New("identity: not the identity partition")
	ErrNotMounted             = errors.New("identity: partition not mounted")
	ErrUnsupportedVersion     = errors.New("identity: unsupported identity partition version")
	ErrUninitializedPartition = errors.New("identity: partition uninitialized")
	ErrAlreadyInitialized     = errors.New("identity: partition already initialized")
	ErrNoPEMData              = errors.New("identity: no PEM data")
	ErrPEMEncoding            = errors.New("identity: PEM encoding error")
)
