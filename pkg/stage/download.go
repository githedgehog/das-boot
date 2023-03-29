package stage

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"time"
)

func DownloadExecutable(ctx context.Context, hc *http.Client, srcURL string, destPath string, timeout time.Duration) error {
	// build the request
	subCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(subCtx, http.MethodGet, srcURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/octet-stream")
	req.Header.Add("Accept", "application/json")

	// open the destPath first
	// no need to go to the network if we cannot even write it to a file
	f, err := os.OpenFile(destPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0755)
	if err != nil {
		return fmt.Errorf("open '%s': %w", destPath, err)
	}
	defer f.Close()

	// execute the request
	httpResp, err := hc.Do(req)
	if err != nil {
		return err
	}
	defer httpResp.Body.Close()

	// if it was an error, parse the error and return as such
	contentType := httpResp.Header.Get("Content-Type")
	if httpResp.StatusCode != http.StatusOK {
		if contentType != "application/json" {
			return NewHTTPErrorf(httpResp, "failed to decode error as the content is not JSON, but '%s'", contentType)
		}
		return NewHTTPErrorFromBody(httpResp)
	}

	// check the content type
	if contentType != "application/octet-stream" {
		return NewHTTPErrorf(httpResp, "but unexpected content type: %s", contentType)
	}

	// now we can copy the body to the file
	w := bufio.NewWriter(f)
	defer w.Flush()
	if _, err := io.Copy(w, httpResp.Body); err != nil {
		return fmt.Errorf("writing HTTP response body to '%s': %w", destPath, err)
	}

	return nil
}

func BuildURL(base string, pathAddendum string) (string, error) {
	url, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("URL parsing: %w", err)
	}
	url.Path = path.Join(url.Path, pathAddendum)
	return url.String(), nil
}
