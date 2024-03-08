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
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/x509"
	"io"

	"go.githedgehog.com/dasboot/pkg/devid"
)

var (
	ecdsaGenerateKey             func(c elliptic.Curve, rand io.Reader) (*ecdsa.PrivateKey, error)                         = ecdsa.GenerateKey
	x509MarshalECPrivateKey      func(key *ecdsa.PrivateKey) ([]byte, error)                                               = x509.MarshalECPrivateKey
	x509CreateCertificateRequest func(rand io.Reader, template *x509.CertificateRequest, priv any) (csr []byte, err error) = x509.CreateCertificateRequest
	devidID                      func() string                                                                             = devid.ID
)
