package identity

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"

	"go.githedgehog.com/dasboot/pkg/partitions"
	"go.githedgehog.com/dasboot/pkg/partitions/location"
	"go.githedgehog.com/dasboot/pkg/tpm"

	"github.com/google/uuid"
)

type api struct {
	dev *partitions.Device
}

var _ IdentityPartition = &api{}

// Open an existing identity partition. If the partition was not previously initialized
// this function returns `ErrUninitializedPartition` in which case the caller should
// call `Init()` instead.
func Open(d *partitions.Device) (IdentityPartition, error) {
	// initial checks
	if !d.IsHedgehogIdentityPartition() {
		return nil, ErrWrongDevice
	}
	if !d.IsMounted() {
		return nil, ErrNotMounted
	}

	// read version file
	f, err := d.FS.Open(versionFilePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// this means the caller should call `Init` instead
			return nil, ErrUninitializedPartition
		}
		return nil, err
	}
	defer f.Close()
	b, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	i, err := strconv.ParseInt(string(bytes.TrimSpace(b)), 0, 0)
	if err != nil {
		return nil, err
	}

	// only version one is supported right now
	if i != version1 {
		return nil, ErrUnsupportedVersion
	}

	// all validations complete, return the API object
	return &api{
		dev: d,
	}, nil
}

// Initializes the identity partition. If the partition has been
// previously initialized already, this function will fail with
// `ErrAlreadyInitialized`.
func Init(d *partitions.Device) (IdentityPartition, error) {
	// initial checks
	if !d.IsHedgehogIdentityPartition() {
		return nil, ErrWrongDevice
	}
	if !d.IsMounted() {
		return nil, ErrNotMounted
	}

	// check it's not initialized already
	_, err := d.FS.Stat(versionFilePath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if err == nil {
		return nil, ErrAlreadyInitialized
	}

	// clean the partition
	entries, err := d.FS.ReadDir("")
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if entry.Name() == "lost+found" {
			continue
		}
		if err := d.FS.RemoveAll(entry.Name()); err != nil {
			return nil, fmt.Errorf("identity: cleaning partition failed at '%s': %w", entry.Name(), err)
		}
	}

	// write the version file, and create identity and location directories
	// which is the minimum to initialize it
	f, err := d.FS.OpenFile(versionFilePath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if _, err := f.Write([]byte(strconv.Itoa(int(version1)))); err != nil {
		return nil, err
	}
	if err := d.FS.Mkdir(identityDirPath, 0755); err != nil {
		return nil, err
	}
	if err := d.FS.Mkdir(locationDirPath, 0755); err != nil {
		return nil, err
	}

	// initialized, return the API object
	return &api{
		dev: d,
	}, nil
}

// GenerateClientCSR implements IdentityPartition
func (*api) GenerateClientCSR() (*x509.CertificateRequest, error) {
	panic("unimplemented")
}

// GenerateClientKeyPair implements IdentityPartition
func (*api) GenerateClientKeyPair() error {
	panic("unimplemented")
}

// GetLocation implements IdentityPartition
func (a *api) GetLocation() (*location.Info, error) {
	// uuid
	f1, err := a.dev.FS.Open(locationUUIDPath)
	if err != nil {
		return nil, err
	}
	defer f1.Close()
	uuidBytes, err := io.ReadAll(f1)
	if err != nil {
		return nil, err
	}
	luuid, err := uuid.ParseBytes(uuidBytes)
	if err != nil {
		return nil, err
	}

	// uuid.sig
	f2, err := a.dev.FS.Open(locationUUIDSigPath)
	if err != nil {
		return nil, err
	}
	defer f2.Close()
	uuidSigBytes, err := io.ReadAll(f2)
	if err != nil {
		return nil, err
	}

	// metadata
	f3, err := a.dev.FS.Open(locationMetadataPath)
	if err != nil {
		return nil, err
	}
	defer f3.Close()
	metadataBytes, err := io.ReadAll(f3)
	if err != nil {
		return nil, err
	}
	var md location.Metadata
	if err := json.Unmarshal(metadataBytes, &md); err != nil {
		return nil, err
	}

	// metadata.sig
	f4, err := a.dev.FS.Open(locationMetadataSigPath)
	if err != nil {
		return nil, err
	}
	defer f4.Close()
	metadataSigBytes, err := io.ReadAll(f4)
	if err != nil {
		return nil, err
	}

	// now return it
	// we validated as good as we can at this point that this is good data
	return &location.Info{
		UUID:        luuid.String(),
		UUIDSig:     uuidSigBytes,
		Metadata:    string(metadataBytes),
		MetadataSig: metadataSigBytes,
	}, nil
}

// HasClientCSR implements IdentityPartition
func (a *api) HasClientCSR() bool {
	f, err := a.dev.FS.Open(clientCSRPath)
	if err != nil {
		return false
	}
	defer f.Close()
	csrPEMBytes, err := io.ReadAll(f)
	if err != nil {
		return false
	}
	p, _ := pem.Decode(csrPEMBytes)
	if p == nil {
		return false
	}
	if p.Type != "CERTIFICATE REQUEST" {
		return false
	}
	_, err = x509.ParseCertificateRequest(p.Bytes)
	return err == nil
}

// HasClientCert implements IdentityPartition
func (a *api) HasClientCert() bool {
	f, err := a.dev.FS.Open(clientCertPath)
	if err != nil {
		return false
	}
	defer f.Close()
	certPEMBytes, err := io.ReadAll(f)
	if err != nil {
		return false
	}
	p, _ := pem.Decode(certPEMBytes)
	if p == nil {
		return false
	}
	if p.Type != "CERTIFICATE" {
		return false
	}
	_, err = x509.ParseCertificate(p.Bytes)
	return err == nil
}

// HasClientKey implements IdentityPartition
func (a *api) HasClientKey() bool {
	if tpm.HasTPM() {
		return a.hasClientKeyFromTPM()
	}
	return a.hasClientKeyFromFiles()
}

func (a *api) hasClientKeyFromTPM() bool {
	// TODO: implement
	return false
}

func (a *api) hasClientKeyFromFiles() bool {
	f, err := a.dev.FS.Open(clientKeyPath)
	if err != nil {
		return false
	}
	defer f.Close()
	keyPEMBytes, err := io.ReadAll(f)
	if err != nil {
		return false
	}
	p, _ := pem.Decode(keyPEMBytes)
	if p == nil {
		return false
	}
	_, err = x509.ParseECPrivateKey(p.Bytes)
	return err == nil
}

// LoadX509KeyPair implements IdentityPartition
func (a *api) LoadX509KeyPair() (tls.Certificate, error) {
	if tpm.HasTPM() {
		return a.loadX509KeyPairFromTPM()
	}
	return a.loadX509KeyPairFromFiles()
}

func (a *api) loadX509KeyPairFromFiles() (tls.Certificate, error) {
	return tls.LoadX509KeyPair(a.dev.FS.Path(clientCertPath), a.dev.FS.Path(clientKeyPath))
}

func (a *api) loadX509KeyPairFromTPM() (tls.Certificate, error) {
	// TODO: implement
	return tls.Certificate{}, nil
}

// ReadClientCSR implements IdentityPartition
func (a *api) ReadClientCSR() (*x509.CertificateRequest, error) {
	f, err := a.dev.FS.Open(clientCSRPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	csrPEMBytes, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	p, _ := pem.Decode(csrPEMBytes)
	if p == nil {
		return nil, ErrNoPEMData
	}
	return x509.ParseCertificateRequest(p.Bytes)
}

// StoreClientCert implements IdentityPartition
func (a *api) StoreClientCert(certBytes []byte) error {
	if _, err := x509.ParseCertificate(certBytes); err != nil {
		return fmt.Errorf("identity: not a valid certificate: %w", err)
	}
	p := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	}
	certPEMBytes := pem.EncodeToMemory(p)
	if certPEMBytes == nil {
		return ErrPEMEncoding
	}
	f, err := a.dev.FS.OpenFile(clientCertPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(certPEMBytes)
	return err
}
