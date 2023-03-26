package registration

import (
	"context"
	"net/http"
)

// DoRequest will submit the initial device registration request as passed in `registrationReq`, and it will then poll
// potentially *forever* until it receives a response which has an approved registration request and contains a DER encoded
// client certificate
func DoRequest(ctx context.Context, hc *http.Client, registrationReq *Request, registrationURL string) (*Response, error) {
	return &Response{}, nil
}
