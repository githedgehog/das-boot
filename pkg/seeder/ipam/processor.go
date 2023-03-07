package ipam

import (
	"context"

	"go.githedgehog.com/dasboot/pkg/seeder/controlplane"
)

// ProcessRequest processes an IPAM request and delivers back a response object.
func ProcessRequest(ctx context.Context, cpc controlplane.Client, req *Request) (*Response, error) {
	return &Response{}, nil
}
