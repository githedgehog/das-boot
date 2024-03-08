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
	"crypto/x509"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.githedgehog.com/dasboot/pkg/partitions/location"
)

var (
	ErrInvalidUUID = errors.New("registration: invalid uuid")
	ErrInvalidCSR  = errors.New("registration: invalid CSR")
)

func invalidUUIDError(str string, err error) error {
	return fmt.Errorf("%w: %s: %w", ErrInvalidUUID, str, err)
}

func invalidCSRError(err error) error {
	return fmt.Errorf("%w: %w", ErrInvalidCSR, err)
}

// Request represents a registration request as performed by the stage 1 installer
type Request struct {
	DeviceID     string         `json:"devid,omitempty"`
	CSR          []byte         `json:"csr,omitempty"`
	LocationInfo *location.Info `json:"location_info,omitempty"`
}

func (r *Request) Validate() error {
	// devid
	if _, err := uuid.Parse(r.DeviceID); err != nil {
		return invalidUUIDError("devid", err)
	}

	if len(r.CSR) > 0 {
		if _, err := x509.ParseCertificateRequest(r.CSR); err != nil {
			return invalidCSRError(err)
		}
	}

	return nil
}
