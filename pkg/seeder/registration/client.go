package registration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"time"

	"go.githedgehog.com/dasboot/pkg/stage"
)

var (
	// HTTPProcessError is returned when there was an internal processing issue during either the device approval or the certificate issuing process
	HTTPProcessError = 566

	// HTTPRegistrationRequestNotFound is returned when the registration request could not be located by the internal processor
	HTTPRegistrationRequestNotFound = 464
)

var (
	// ErrRegistrationRequestNotFound is returned when the registration request could not be located by the internal processor
	ErrRegistrationRequestNotFound = errors.New("registration request not found")
)

func DoPollRequest(ctx context.Context, hc *http.Client, deviceID string, registrationURL string) (*Response, error) {
	// this is just an internal check to ensure that we have a good device ID
	registrationReq := &Request{DeviceID: deviceID}
	// validate request first
	if err := registrationReq.Validate(); err != nil {
		return nil, err
	}
	url, err := url.Parse(registrationURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse registration URL: %w", err)
	}
	url.Path = path.Join(url.Path, deviceID)

	// build the request
	subCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(subCtx, http.MethodGet, url.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	// execute the request
	httpResp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	// if this was a good response, we parse it as such
	// the following status codes indicate that
	// - 200
	// - 202
	// - 464
	// - 566
	// NOTE: 464 and 566 are errors but will be represented with the same data structure
	if httpResp.StatusCode == http.StatusOK || httpResp.StatusCode == http.StatusAccepted || httpResp.StatusCode == HTTPRegistrationRequestNotFound || httpResp.StatusCode == HTTPProcessError {
		var resp Response
		if err := json.NewDecoder(httpResp.Body).Decode(&resp); err != nil {
			return nil, err
		}

		// in this case somebody must have cleaned out the registration request
		// we cannot recover from this, and need to start over
		if httpResp.StatusCode == HTTPRegistrationRequestNotFound {
			return nil, fmt.Errorf("%w: registration request not found by the processor: %s: %s", ErrRegistrationRequestNotFound, resp.Status, resp.StatusDescription)
		}

		// we cannot recover from internal processing errors, and need to retry
		if httpResp.StatusCode == HTTPProcessError {
			return nil, fmt.Errorf("device approval or certificate issuing processing error: %s: %s", resp.Status, resp.StatusDescription)
		}

		return &resp, nil
	}

	// if it was an error, return as such
	// the following response codes indicate that
	// - 400
	// - 500
	// - 501
	// NOTE: all others indicate an unknwon behaviour, but NewHTTPErrorFromBody accounts for that anyways
	// we cannot recover from any of these errors either
	return nil, stage.NewHTTPErrorFromBody(httpResp)
}

// DoRequest will submit the initial device registration request as passed in `registrationReq`, and it will then poll
// potentially *forever* until it receives a response which has an approved registration request and contains a DER encoded
// client certificate
func DoRequest(ctx context.Context, hc *http.Client, registrationReq *Request, registrationURL string) (*Response, error) {
	// validate request first
	if err := registrationReq.Validate(); err != nil {
		return nil, err
	}

	// create the post body for it
	// NOTE: json encoder has a problem which is why json.Marshal is better for creating post bodies
	postBody, err := json.Marshal(registrationReq)
	if err != nil {
		return nil, err
	}

	// build the request
	subCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(subCtx, http.MethodPost, registrationURL, bytes.NewBuffer(postBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	// execute the request
	httpResp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	// if this was a good response, we parse it as such
	// the following status codes indicate that
	// - 200
	// - 202
	// - 464
	// - 566
	// NOTE: 464 and 566 are errors but will be represented with the same data structure
	if httpResp.StatusCode == http.StatusOK || httpResp.StatusCode == http.StatusAccepted || httpResp.StatusCode == HTTPRegistrationRequestNotFound || httpResp.StatusCode == HTTPProcessError {
		var resp Response
		if err := json.NewDecoder(httpResp.Body).Decode(&resp); err != nil {
			return nil, err
		}

		// in this case somebody must have cleaned out the registration request
		// we cannot recover from this, and need to start over
		if httpResp.StatusCode == HTTPRegistrationRequestNotFound {
			return nil, fmt.Errorf("registration request not found by the processor: %s: %s", resp.Status, resp.StatusDescription)
		}

		// we cannot recover from internal processing errors, and need to retry
		if httpResp.StatusCode == HTTPProcessError {
			return nil, fmt.Errorf("device approval or certificate issuing processing error: %s: %s", resp.Status, resp.StatusDescription)
		}

		return &resp, nil
	}

	// if it was an error, return as such
	// the following response codes indicate that
	// - 400
	// - 500
	// - 501
	// NOTE: all others indicate an unknwon behaviour, but NewHTTPErrorFromBody accounts for that anyways
	// we cannot recover from any of these errors either
	return nil, stage.NewHTTPErrorFromBody(httpResp)
}
