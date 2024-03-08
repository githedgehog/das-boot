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
	return Download(ctx, hc, srcURL, destPath, 0755, timeout)
}

func Download(ctx context.Context, hc *http.Client, srcURL string, destPath string, destPerm os.FileMode, timeout time.Duration) error {
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
	f, err := os.OpenFile(destPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, destPerm)
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
	if contentType != "application/octet-stream" && contentType != "application/yaml" {
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
