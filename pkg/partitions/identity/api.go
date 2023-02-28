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
	// HasClientKey tests if the partition already holds a valid client key. The implementation needs to validate
	// that the key on disk (or TPM) is in fact valid.
	HasClientKey() bool

	// HasClientCSR tests if the partition already holds a valid client certificate request. The implementation needs
	// to validate that the CSR on disk is in fact valid, and optionally should check if the embedded public key belongs
	// to the key on disk (or TPM).
	HasClientCSR() bool

	// HasClientCert tests if the partition already holds a valid certificate. The implementation needs to validate that
	// the certificate on disk is in fact valid, and optionally should check if the embedded public key belongs to the
	// key on disk (or TPM).
	HasClientCert() bool

	// GenerateClientKeyPair generates a new key client key pair. It must overwrite any existing keys on disk (or TPM).
	// Therefore a call to `HasClientKey` is recommended if overwriting would not be the intention.
	GenerateClientKeyPair() error

	// GenerateClientCSR generates a new CSR using the key pair on disk (or TPM). It overwrites any existing CSR on disk.
	// Therefore a call to `HasClientCSR` is recommended if overwriting would not be the intention.
	GenerateClientCSR() (*x509.CertificateRequest, error)

	// ReadClientCSR reads the client CSR from the partition. It fails if it does not exist yet, in which case the caller
	// should call `GenerateClientCSR` first.
	ReadClientCSR() (*x509.CertificateRequest, error)

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
