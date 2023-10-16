package identity

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

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
	var version Version
	if err := json.NewDecoder(f).Decode(&version); err != nil {
		return nil, err
	}

	// only version one is supported right now
	if version.Version != version1 {
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
	version := Version{
		Version: version1,
	}
	// cannot fail, we can be certain
	b, _ := json.Marshal(version) //nolint: errchkjson
	b = append(b, byte('\n'))
	if _, err := f.Write(b); err != nil {
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
func (a *api) GenerateClientCSR() ([]byte, error) {
	var b []byte
	var err error
	if tpm.HasTPM() {
		b, err = a.generateClientCSRWithTPM()
	} else {
		b, err = a.generateClientCSRWithoutTPM()
	}
	if err != nil {
		return nil, err
	}

	// and delete an existing certificate if it is there
	err = a.dev.FS.Remove(clientCertPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("deleting already existing certificate: %w", err)
	}

	return b, nil
}

func (a *api) generateClientCSRWithTPM() ([]byte, error) {
	// TODO: implement
	return nil, nil
}

func (a *api) generateClientCSRWithoutTPM() ([]byte, error) {
	// read client key from disk
	f, err := a.dev.FS.Open(clientKeyPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	p, _ := pem.Decode(b)
	if p == nil {
		return nil, ErrNoPEMData
	}
	key, err := x509.ParseECPrivateKey(p.Bytes)
	if err != nil {
		return nil, err
	}

	// now generate CSR
	id := devidID()
	if id == "" {
		return nil, ErrNoDevID
	}
	// TODO: the Subject needs review
	csr := &x509.CertificateRequest{
		PublicKey: key.PublicKey,
		Subject: pkix.Name{
			CommonName: id,
		},
	}
	csrBytes, err := x509CreateCertificateRequest(rand.Reader, csr, key)
	if err != nil {
		return nil, fmt.Errorf("failed to create CSR: %w", err)
	}

	// save it to disk
	f2, err := a.dev.FS.OpenFile(clientCSRPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	defer f2.Close()

	p2 := pem.Block{
		Type:  "CERTIFICATE REQUEST",
		Bytes: csrBytes,
	}
	if _, err := f2.Write(pem.EncodeToMemory(&p2)); err != nil {
		return nil, err
	}

	// return with the generated CSR
	return csrBytes, nil
}

// GenerateClientKeyPair implements IdentityPartition
func (a *api) GenerateClientKeyPair() error {
	var err error
	if tpm.HasTPM() {
		err = a.generateClientKeyPairWithTPM()
	} else {
		err = a.generateClientKeyPairWithoutTPM()
	}
	if err != nil {
		return err
	}

	// now ensure to delete an existing CSR if it is there
	err = a.dev.FS.Remove(clientCSRPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("deleting already existing CSR: %w", err)
	}

	// and delete an existing certificate if it is there
	err = a.dev.FS.Remove(clientCertPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("deleting already existing certificate: %w", err)
	}

	return nil
}

func (a *api) generateClientKeyPairWithTPM() error {
	// TODO: implement
	return nil
}

func (a *api) generateClientKeyPairWithoutTPM() error {
	key, err := ecdsaGenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}
	keyBytes, err := x509MarshalECPrivateKey(key)
	if err != nil {
		return err
	}
	p := &pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: keyBytes,
	}
	keyPEMBytes := pem.EncodeToMemory(p)
	f, err := a.dev.FS.OpenFile(clientKeyPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(keyPEMBytes); err != nil {
		return err
	}
	return nil
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

// StoreLocation implements IdentityPartition
func (a *api) StoreLocation(info *location.Info) error {
	// uuid
	f1, err := a.dev.FS.OpenFile(locationUUIDPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f1.Close()
	if _, err := f1.Write([]byte(info.UUID)); err != nil {
		return err
	}

	// uuid.sig
	f2, err := a.dev.FS.OpenFile(locationUUIDSigPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f2.Close()
	if _, err := f2.Write(info.UUIDSig); err != nil {
		return err
	}

	// metadata
	f3, err := a.dev.FS.OpenFile(locationMetadataPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f3.Close()
	if _, err := f3.Write([]byte(info.Metadata)); err != nil {
		return err
	}

	// metadata.sig
	f4, err := a.dev.FS.OpenFile(locationMetadataSigPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f4.Close()
	if _, err := f4.Write(info.MetadataSig); err != nil {
		return err
	}

	return nil
}

// CopyLocation implements IdentityPartition
func (a *api) CopyLocation(lp location.LocationPartition) error {
	info, err := lp.GetLocation()
	if err != nil {
		return err
	}
	return a.StoreLocation(info)
}

// HasClientCSR im plements IdentityPartition
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

// HasValidClientCert implements IdentityPartition
func (a *api) HasValidClientCert() bool {
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
	cert, err := x509.ParseCertificate(p.Bytes)
	if err != nil {
		return false
	}

	// Up until here this just confirms that we have a certificate
	// Now perform the checks to see if we have a valid certificate
	// check that the cert is not expired
	now := time.Now()
	if !(cert.NotBefore.Before(now) && cert.NotAfter.After(now)) {
		return false
	}

	// check that the public key of the cert matches the client key
	// the best way to test this is simply to see if we can load the golang TLS pair
	_, err = a.LoadX509KeyPair()
	return err == nil
}

// MatchesClientCertificate implements IdentityPartition.
func (a *api) MatchesClientCertificate(cert *x509.Certificate) bool {
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
	certOnDisk, err := x509.ParseCertificate(p.Bytes)
	if err != nil {
		return false
	}
	return certOnDisk.Equal(cert)
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
	if p.Type != "EC PRIVATE KEY" {
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
func (a *api) ReadClientCSR() ([]byte, error) {
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
	if _, err := x509.ParseCertificateRequest(p.Bytes); err != nil {
		return nil, err
	}
	return p.Bytes, nil
}

// StoreClientCert implements IdentityPartition
func (a *api) StoreClientCert(certBytes []byte) error {
	// validate input first
	cert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return fmt.Errorf("identity: not a valid certificate: %w", err)
	}
	certPub, ok := cert.PublicKey.(*ecdsa.PublicKey)
	if !ok {
		return fmt.Errorf("identity: not an ECDSA certificate")
	}

	// we need to check the certificate against the CSR first
	// before we allow it to be stored
	csrBytes, err := a.ReadClientCSR()
	if err != nil {
		return fmt.Errorf("identity: failed to read CSR while trying to store cert: %w", err)
	}
	csr, err := x509.ParseCertificateRequest(csrBytes)
	if err != nil {
		return fmt.Errorf("identity: failed to parse CSR while trying to store cert: %w", err)
	}
	csrPub, ok := csr.PublicKey.(*ecdsa.PublicKey)
	if !ok {
		return fmt.Errorf("identity: CSR is not an ECDSA CSR")
	}

	// the public keys need to match
	if !csrPub.Equal(certPub) {
		return fmt.Errorf("identity: CSR and certificate public keys do not match")
	}

	p := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	}
	// This can only fail if writing to a memory buffer fails
	// in which case this would return nil.
	// All other cases are impossible as this is a static known-to-work
	// struct definition.
	// We will accept the risk of this being nil. Nothing will work
	// anymore anyways if Go runs out of memory here.
	certPEMBytes := pem.EncodeToMemory(p)

	f, err := a.dev.FS.OpenFile(clientCertPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(certPEMBytes)
	return err
}
