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

package ipam

import "github.com/google/uuid"

// Request represents an IPAM request as being performed by the Stage 0 installer
type Request struct {
	Arch                  string   `json:"arch"`
	DevID                 string   `json:"devid"`
	LocationUUID          string   `json:"location_uuid"`
	LocationUUIDSignature []byte   `json:"location_uuid_signature"`
	Interfaces            []string `json:"interfaces,omitempty"`
}

func (r *Request) Validate() error {
	// arch
	switch r.Arch {
	case "x86_64":
		fallthrough
	case "arm64":
		fallthrough
	case "arm":
		// no error
	default:
		return unsupportedArchError(r.Arch)
	}

	// devid
	if _, err := uuid.Parse(r.DevID); err != nil {
		return invalidUUIDError("devid", err)
	}

	// location uuid
	if r.LocationUUID != "" {
		if _, err := uuid.Parse(r.LocationUUID); err != nil {
			return invalidUUIDError("location_uuid", err)
		}

		// location uuid signature
		if len(r.LocationUUIDSignature) == 0 {
			return emptyValueError("location_uuid_signature")
		}
	}

	// interfaces
	if len(r.Interfaces) == 0 {
		return emptyValueError("interfaces")
	}

	return nil
}
