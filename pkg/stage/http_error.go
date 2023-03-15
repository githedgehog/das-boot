package stage

import (
	"encoding/json"
	"fmt"
	"io"
)

// HTTPError is the error structure as it is always being returned for any unsuccessful HTTP requests
// by the seeder. Let's define it once here, and reuse it where we need it.
type HTTPError struct {
	StatusCode int    `json:"-"`
	ReqID      string `json:"request_id,omitempty"`
	Err        string `json:"error"`
}

// Error implements error
func (e *HTTPError) Error() string {
	reqID := ""
	if e.ReqID != "" {
		reqID = fmt.Sprintf(" (ReqID: %s)", e.ReqID)
	}
	return fmt.Sprintf("HTTP %d%s: %s", e.StatusCode, reqID, e.Err)
}

func (e *HTTPError) Is(target error) bool {
	_, ok := target.(*HTTPError) //nolint: errorlint
	return ok
}

func NewHTTPError(statusCode int, body io.Reader) error {
	var v HTTPError
	if err := json.NewDecoder(body).Decode(&v); err != nil {
		return &HTTPError{
			StatusCode: statusCode,
			ReqID:      "<unknown>",
			Err:        fmt.Sprintf("failed to parse HTTP error from body: %s", err),
		}
	}
	v.StatusCode = statusCode
	return &v
}
