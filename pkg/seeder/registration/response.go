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
