package ipam

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"go.githedgehog.com/dasboot/pkg/stage"
)

func DoRequest(ctx context.Context, hc *http.Client, ipamReq *Request, ipamURL string) (*Response, error) {
	// validate request first
	if err := ipamReq.Validate(); err != nil {
		return nil, err
	}

	// create the post body for it
	// NOTE: json encoder has a problem which is why json.Marshal is better for creating post bodies
	postBody, err := json.Marshal(ipamReq)
	if err != nil {
		return nil, err
	}

	// build the request
	subCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(subCtx, http.MethodPost, ipamURL, bytes.NewBuffer(postBody))
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

	// parse response
	// if it was an error, return as such
	if httpResp.StatusCode != http.StatusOK {
		return nil, stage.NewHTTPError(httpResp.StatusCode, httpResp.Body)
	}

	// otherwise we parse it as an IPAM response
	var resp Response
	if err := json.NewDecoder(httpResp.Body).Decode(&resp); err != nil {
		return nil, err
	}

	// return with response
	return &resp, nil
}
