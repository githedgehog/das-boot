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

package registration

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"

	"go.githedgehog.com/dasboot/pkg/log"
	"go.uber.org/zap"
)

func matchesPublicKeys(csrDERBytes []byte, certDERBytes []byte) bool {
	l := log.L()
	csr, err := x509.ParseCertificateRequest(csrDERBytes)
	if err != nil {
		l.Error("registration processor: matchesPublicKeys: failed to parse CSR", zap.Error(err))
		return false
	}
	cert, err := x509.ParseCertificate(certDERBytes)
	if err != nil {
		l.Error("registration processor: matchesPublicKeys: failed to parse certificate", zap.Error(err))
		return false
	}
	switch csrPub := csr.PublicKey.(type) {
	case *ecdsa.PublicKey:
		certPub, ok := cert.PublicKey.(*ecdsa.PublicKey)
		if !ok {
			l.Error("registration processor: matchesPublicKeys: certificate public key is not an ECDSA public key, but CSR is ECDSA")
			return false
		}
		return csrPub.Equal(certPub)
	case *rsa.PublicKey:
		certPub, ok := cert.PublicKey.(*rsa.PublicKey)
		if !ok {
			l.Error("registration processor: matchesPublicKeys: certificate public key is not an RSA public key, but CSR is RSA")
			return false
		}
		return csrPub.Equal(certPub)
	default:
		l.Error("registration processor: matchesPublicKeys: CSR public key is neither ECDSA nor RSA")
		return false
	}
}
