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

package seeder

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5/middleware"
	"go.githedgehog.com/dasboot/pkg/log"
)

var l = log.L()

func readKeyFromPath(path string) (*ecdsa.PrivateKey, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open '%s': %w", path, err)
	}
	b, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("reading '%s': %w", path, err)
	}
	p, _ := pem.Decode(b)
	key, err := x509.ParseECPrivateKey(p.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parsing key '%s': %w", path, err)
	}
	return key, nil
}

func readCertFromPath(path string) (*x509.Certificate, []byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("open '%s': %w", path, err)
	}
	b, err := io.ReadAll(f)
	if err != nil {
		return nil, nil, fmt.Errorf("reading '%s': %w", path, err)
	}
	p, _ := pem.Decode(b)
	cert, err := x509.ParseCertificate(p.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing certficate '%s': %w", path, err)
	}
	return cert, p.Bytes, nil
}

func errorWithJSON(w http.ResponseWriter, r *http.Request, statusCode int, format string, a ...any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	v := struct {
		ReqID string `json:"request_id,omitempty"`
		Err   string `json:"error"`
	}{
		ReqID: middleware.GetReqID(r.Context()),
		Err:   fmt.Sprintf(format, a...),
	}
	b, err := json.Marshal(&v)
	if err == nil {
		w.Write(b) //nolint: errcheck
	}
}
