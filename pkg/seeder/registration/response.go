package registration

type Response struct {
	// ClientCertificate is the issued client certificate for the requestor
	ClientCertificate []byte `json:"client_certificate,omitempty"`
}
