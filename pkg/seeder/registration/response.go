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

type RegistrationStatus string

const (
	RegistrationStatusUnknown  RegistrationStatus = ""
	RegistrationStatusNotFound RegistrationStatus = "NotFound"
	RegistrationStatusPending  RegistrationStatus = "Pending"
	RegistrationStatusApproved RegistrationStatus = "Approved"
	RegistrationStatusRejected RegistrationStatus = "Rejected"
	RegistrationStatusError    RegistrationStatus = "Error"
)

type Response struct {
	// Status describes the status of the registration of a device
	Status RegistrationStatus `json:"status,omitempty"`

	// StatusDescription describes the status in a human readable form
	StatusDescription string `json:"description,omitempty"`

	// ClientCertificate is the issued client certificate for the requestor
	ClientCertificate []byte `json:"client_certificate,omitempty"`
}
