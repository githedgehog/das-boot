package registration

import (
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
		l.Error("Retrieving DeviceRegistration failed", zap.String("deviceID", req.DeviceID), zap.Error(err))
		return &cert{}, true
	}

	// TODO: evaluate status of object properly
	l.Info("DeviceRegistration retrieved")
	reason := "issued by registration-controller"
	rejected := false
	return &cert{
		der:      reg.Status.Certificate,
		reason:   reason,
		rejected: rejected,
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
