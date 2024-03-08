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
	"bytes"
	"context"
	"errors"

	dasbootv1alpha1 "go.githedgehog.com/dasboot/pkg/k8s/api/v1alpha1"
	"go.githedgehog.com/dasboot/pkg/log"
	"go.githedgehog.com/dasboot/pkg/seeder/controlplane"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (p *Processor) getRequestWithControlPlane(ctx context.Context, req *Request) (*cert, bool) {
	l := log.L()
	reg, err := p.cpc.GetDeviceRegistration(ctx, req.DeviceID)
	if err != nil {
		// in case of not found error, we return as such which will trigger a call to addRequestWithControlPlane
		if errors.Is(err, controlplane.ErrNotFound) {
			return nil, false
		}

		// TODO: not entirely sure what is best here
		// turning this into an error is probably wrong as the client aborts completely
		// so we'll essentially return with what is considered the "pending" state
		l.Error("registration processor: retrieving DeviceRegistration failed", zap.String("deviceID", req.DeviceID), zap.Error(err))
		return &cert{}, true
	}

	// if there is a CSR in the request, we check if it matches the CSR in the spec
	// we consider this rejected if they do not match, as this requires manual intervention
	if len(req.CSR) > 0 {
		if !bytes.Equal(req.CSR, reg.Spec.CSR) {
			// TODO: there are some cases where we might be okay to let this slide:
			// - when the certificate expired and we are expecting a new CSR for the same device (must be sanctioned by the controller though)
			// - when the device changes location, and this is an expected change of location change (must be sanctioned by the controller as well)
			l.Error("registration processor: DeviceRegistration retrieved but CSR does not match", zap.String("deviceID", req.DeviceID))
			return &cert{
				rejected: true,
				reason:   "CSR of registration request does not match CSR of existing registration request. If this is expected because for example the device was expected to generate a new identity, then you need to delete the previous device registration.",
			}, true
		}
	}

	// if there is no certificate yet, we simply return
	if len(reg.Status.Certificate) == 0 {
		l.Info("registration processor: DeviceRegistration retrieved but no certificate has been issued yet", zap.String("deviceID", req.DeviceID))
		return &cert{}, true
	}

	// if there is a certificate, we check its public key against the CSR of the spec
	// only if they match do we know that a new certificate was issued
	// otherwise we assume that the certificate is still being issued
	// and we return with the "pending" state
	if !matchesPublicKeys(reg.Spec.CSR, reg.Status.Certificate) {
		l.Warn("registration processor: DeviceRegistration retrieved but the public key of the certificate does not match the public key of the CSR. This can happen when a new certificate is still being issued for a new CSR.", zap.String("deviceID", req.DeviceID))
		return &cert{}, true
	}

	// TODO: check more things here, like:
	// - certificate is not expired

	l.Info("registration processor: DeviceRegistration and issued certificate retrieved", zap.String("deviceID", req.DeviceID))
	return &cert{
		der:      reg.Status.Certificate,
		reason:   "issued by registration-controller",
		rejected: false,
	}, true
}

func (p *Processor) addRequestWithControlPlane(ctx context.Context, req *Request) {
	l := log.L()
	regReq := &dasbootv1alpha1.DeviceRegistration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.DeviceID,
			Namespace: p.cpc.DeviceNamespace(),
		},
		Spec: dasbootv1alpha1.DeviceRegistrationSpec{
			LocationUUID: req.LocationInfo.UUID,
			CSR:          req.CSR,
		},
	}
	ret, err := p.cpc.CreateDeviceRegistration(ctx, regReq)
	if err != nil {
		l.Error("Creating device registration object failed", zap.Error(err))
		return
	}
	l.Info("Device registration object created", zap.Reflect("deviceregistration", ret))
}

func (p *Processor) processRequestWithControlPlane(req *Request) {
	// nothing to do here when we are using the control plane
	// this is done by the registration controller
}

func (p *Processor) deleteRequestWithControlPlane(ctx context.Context, req *Request) {
	// nothing to do here when we are using the control plane
	// this is done by the registration controller
}
