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

import (
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"fmt"
	"sync"
	"time"

	"go.githedgehog.com/dasboot/pkg/seeder/controlplane"
)

const (
	defaultCertsCacheRefresh = time.Minute
)

type cert struct {
	der      []byte
	rejected bool
	reason   string
	err      error
}

type Processor struct {
	key                *ecdsa.PrivateKey
	cert               *x509.Certificate
	cpc                controlplane.Client
	certsCacheRefresh  time.Duration
	certsCache         map[string]*cert
	certsCacheLock     sync.RWMutex
	stopFunc           context.CancelFunc
	processRequestFunc func(*Request)
	addRequestFunc     func(context.Context, *Request)
	getRequestFunc     func(context.Context, *Request) (*cert, bool)
	deleteRequestFunc  func(context.Context, *Request)
}

func NewProcessor(ctx context.Context, cpc controlplane.Client, key *ecdsa.PrivateKey, crt *x509.Certificate) *Processor {
	subctx, cancel := context.WithCancel(ctx)
	ret := &Processor{
		key:               key,
		cert:              crt,
		cpc:               cpc,
		certsCache:        make(map[string]*cert),
		certsCacheRefresh: defaultCertsCacheRefresh,
		stopFunc:          cancel,
	}
	if key != nil && crt != nil {
		ret.processRequestFunc = ret.processRequestLocally
		ret.addRequestFunc = ret.addRequestLocally
		ret.getRequestFunc = ret.getRequestLocally
		ret.deleteRequestFunc = ret.deleteRequestLocally
	} else {
		ret.processRequestFunc = ret.processRequestWithControlPlane
		ret.addRequestFunc = ret.addRequestWithControlPlane
		ret.getRequestFunc = ret.getRequestWithControlPlane
		ret.deleteRequestFunc = ret.deleteRequestWithControlPlane
	}
	go ret.loop(subctx)
	return ret
}

func (p *Processor) loop(ctx context.Context) {
	t := time.NewTicker(p.certsCacheRefresh)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			// talk to the control plane and refresh cache
		}
	}
}

// Stop stops the processor
func (p *Processor) Stop() {
	p.stopFunc()
}

func (p *Processor) ProcessRequest(ctx context.Context, req *Request) *Response {
	// get the cache entry
	cert, ok := p.getRequestFunc(ctx, req)

	// not found
	if !ok {
		// not found, but there is a CSR in this request, we treat this as a new request, and will act accordingly
		if len(req.DeviceID) > 0 && req.CSR != nil {
			// add request before we submit it for processing
			p.addRequestFunc(ctx, req)

			// submit the request for processing
			go p.processRequestFunc(req)

			// return the request with status pending for now
			return &Response{
				Status:            RegistrationStatusPending,
				StatusDescription: fmt.Sprintf("registration request for '%s' submitted, pending approval", req.DeviceID),
			}
		}

		// not found, but no new submission
		return &Response{
			Status:            RegistrationStatusNotFound,
			StatusDescription: fmt.Sprintf("device '%s' did not perform a registration attempt before", req.DeviceID),
		}
	}

	// processing error
	if cert.err != nil {
		p.deleteRequestFunc(ctx, req)
		return &Response{
			Status:            RegistrationStatusError,
			StatusDescription: fmt.Sprintf("processing registration request for device '%s' failed: %s", req.DeviceID, cert.err.Error()),
		}
	}

	// cert was rejected
	if cert.rejected {
		p.deleteRequestFunc(ctx, req)
		return &Response{
			Status:            RegistrationStatusRejected,
			StatusDescription: fmt.Sprintf("registration request for device '%s' was rejected: %s", req.DeviceID, cert.reason),
		}
	}

	// device approved and cert signed
	if len(cert.der) > 0 {
		p.deleteRequestFunc(ctx, req)
		return &Response{
			Status:            RegistrationStatusApproved,
			StatusDescription: fmt.Sprintf("device '%s' approved", req.DeviceID),
			ClientCertificate: cert.der,
		}
	}

	// if we are here, that means that registration is still pending
	return &Response{
		Status:            RegistrationStatusPending,
		StatusDescription: fmt.Sprintf("registration request for '%s' is still pending approval", req.DeviceID),
	}
}
