package registration

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha1" //nolint: gosec
	"crypto/x509"
	"math/big"
	mathrand "math/rand"
	"time"

	"go.githedgehog.com/dasboot/pkg/log"
	"go.uber.org/zap"
)

var certificateValidity = time.Hour * 24 * 360

func (p *Processor) getRequestLocally(_ context.Context, req *Request) (*cert, bool) {
	p.certsCacheLock.RLock()
	defer p.certsCacheLock.RUnlock()

	certTmp, ok := p.certsCache[req.DeviceID]
	if !ok {
		return nil, false
	}

	cert := *certTmp
	cert.der = make([]byte, len(certTmp.der))
	copy(cert.der, certTmp.der)
	return &cert, true
}

func (p *Processor) addRequestLocally(_ context.Context, req *Request) {
	// make an entry in the cache immediately
	p.certsCacheLock.Lock()
	p.certsCache[req.DeviceID] = &cert{}
	p.certsCacheLock.Unlock()
}

func (p *Processor) deleteRequestLocally(_ context.Context, req *Request) {
	p.certsCacheLock.Lock()
	delete(p.certsCache, req.DeviceID)
	p.certsCacheLock.Unlock()
}

func (p *Processor) processRequestLocally(req *Request) {
	l := log.L()
	csr, err := x509.ParseCertificateRequest(req.CSR)
	if err != nil {
		l.Error("registration: parsing CSR failed", zap.String("devID", req.DeviceID), zap.Error(err))
		return
	}
	if csr.Subject.CommonName == "" {
		l.Error("registration: CN in CSR empty", zap.String("devID", req.DeviceID))
		return
	}
	if csr.Subject.CommonName != req.DeviceID {
		l.Error("registration: device ID mismatch, not issuing certificate", zap.String("devID", req.DeviceID), zap.String("csrDevID", csr.Subject.CommonName))
		return
	}
	csrPub, ok := csr.PublicKey.(*ecdsa.PublicKey)
	if !ok {
		l.Error("registration: CSR must contain ECDSA key", zap.String("devID", req.DeviceID))
		return
	}
	ecdhCsrPub, err := csrPub.ECDH()
	if err != nil {
		l.Error("registration: cannot convert ECDSA public key to ECDH public key", zap.String("devID", req.DeviceID), zap.Error(err))
		return
	}
	csrPubBytes := ecdhCsrPub.Bytes()
	subjectKeyId := sha1.Sum(csrPubBytes) //nolint: gosec
	template := &x509.Certificate{
		// we copy the subject from the CSR
		SerialNumber: big.NewInt(mathrand.Int63()), //nolint: gosec
		Subject:      csr.Subject,
		SubjectKeyId: subjectKeyId[:],
		NotBefore:    time.Now().Add(time.Minute * -5), // giving it a 5min grace period
		NotAfter:     time.Now().Add(certificateValidity),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	signedCert, err := x509.CreateCertificate(rand.Reader, template, p.cert, csr.PublicKey, p.key)
	if err != nil {
		l.Error("registration: certificate signing failed", zap.String("devID", req.DeviceID), zap.Error(err))
		return
	}

	p.certsCacheLock.Lock()
	p.certsCache[req.DeviceID] = &cert{
		der:    signedCert,
		reason: "device approved and is allowed onto the network",
	}
	p.certsCacheLock.Unlock()
	l.Info("registration: successfully issued device certificate", zap.String("devID", req.DeviceID))
}
