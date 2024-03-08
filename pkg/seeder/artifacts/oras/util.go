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

package oras

import (
	"crypto/tls"
	"crypto/x509"
	"io"
	"os"
)

func caPool(path string) *x509.CertPool {
	if path != "" {
		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()
		b, err := io.ReadAll(f)
		if err != nil {
			return nil
		}
		ret := x509.NewCertPool()
		if !ret.AppendCertsFromPEM(b) {
			return nil
		}
		return ret
	}
	return nil
}

func clientCertificates(certPath, keyPath string) []tls.Certificate {
	tlsCert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil
	}
	return []tls.Certificate{tlsCert}
}
