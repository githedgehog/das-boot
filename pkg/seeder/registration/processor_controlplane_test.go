package registration

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha1" //nolint: gosec
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	mathrand "math/rand"
	"reflect"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	dasbootv1alpha1 "go.githedgehog.com/dasboot/pkg/k8s/api/v1alpha1"
	"go.githedgehog.com/dasboot/pkg/seeder/controlplane"
	"go.githedgehog.com/dasboot/test/mock/seeder/mockcontrolplane"
)

func selfSignedCert() (*ecdsa.PrivateKey, *x509.Certificate) {
	// generate a key and cert
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(mathrand.Int63()), //nolint: gosec
		Subject: pkix.Name{
			Country:      []string{"US"},
			Province:     []string{"Washington"},
			Locality:     []string{"Seattle"},
			Organization: []string{"Hedgehog SONiC Foundation"},
			CommonName:   "Device Registration Controller Test CA",
		},
		NotBefore:             time.Now().Add(-15 * time.Minute),
		NotAfter:              time.Now().Add(time.Hour * 24 * 365),
		IsCA:                  true,
		BasicConstraintsValid: true,
	}
	certBytes, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		panic(err)
	}
	cert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		panic(err)
	}

	return key, cert
}

func newCSRPubKeyAndCert(id string, caKey *ecdsa.PrivateKey, caCert *x509.Certificate) ([]byte, *ecdsa.PublicKey, []byte) { //nolint: unparam
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic(err)
	}
	csr := &x509.CertificateRequest{
		PublicKey: key.PublicKey,
		Subject: pkix.Name{
			CommonName: id,
		},
	}
	csrBytes, err := x509.CreateCertificateRequest(rand.Reader, csr, key)
	if err != nil {
		panic(err)
	}
	ecdhCsrPub, err := key.PublicKey.ECDH()
	if err != nil {
		panic(err)
	}
	csrPubBytes := ecdhCsrPub.Bytes()

	subjectKeyId := sha1.Sum(csrPubBytes) //nolint: gosec
	template := &x509.Certificate{
		// we copy the subject from the CSR
		SerialNumber: big.NewInt(mathrand.Int63()), //nolint: gosec
		Subject:      csr.Subject,
		SubjectKeyId: subjectKeyId[:],
		NotBefore:    time.Now().Add(time.Minute * -5), // giving it a 5min grace period
		NotAfter:     time.Now().Add(certificateValidity),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	signedCertBytes, err := x509.CreateCertificate(rand.Reader, template, caCert, &key.PublicKey, caKey)
	if err != nil {
		panic(err)
	}

	return csrBytes, &key.PublicKey, signedCertBytes
}

func TestProcessor_getRequestWithControlPlane(t *testing.T) {
	caKey, caCert := selfSignedCert()
	csr1, _, cert1 := newCSRPubKeyAndCert("device1", caKey, caCert)
	csr2, _, _ := newCSRPubKeyAndCert("device1", caKey, caCert)
	type args struct {
		req *Request
	}
	tests := []struct {
		name  string
		args  args
		pre   func(t *testing.T, ctrl *gomock.Controller, c *mockcontrolplane.MockClient)
		want  *cert
		want1 bool
	}{
		{
			name: "device registration not found",
			args: args{
				req: &Request{
					DeviceID: "device1",
				},
			},
			pre: func(t *testing.T, ctrl *gomock.Controller, c *mockcontrolplane.MockClient) {
				c.EXPECT().GetDeviceRegistration(gomock.Any(), "device1").Times(1).Return(nil, controlplane.ErrNotFound)
			},
			want:  nil,
			want1: false,
		},
		{
			name: "other control plane errors are currently ignored",
			args: args{
				req: &Request{
					DeviceID: "device1",
				},
			},
			pre: func(t *testing.T, ctrl *gomock.Controller, c *mockcontrolplane.MockClient) {
				c.EXPECT().GetDeviceRegistration(gomock.Any(), "device1").Times(1).Return(nil, fmt.Errorf("other error"))
			},
			want:  &cert{},
			want1: true,
		},
		{
			name: "if this is a new device registration request, and there is an existing CSR, we reject it if they do not match",
			args: args{
				req: &Request{
					DeviceID: "device1",
					CSR:      []byte("csr2"),
				},
			},
			pre: func(t *testing.T, ctrl *gomock.Controller, c *mockcontrolplane.MockClient) {
				c.EXPECT().GetDeviceRegistration(gomock.Any(), "device1").Times(1).Return(&dasbootv1alpha1.DeviceRegistration{
					Spec: dasbootv1alpha1.DeviceRegistrationSpec{
						CSR: []byte("csr1"),
					},
				}, nil)
			},
			want: &cert{
				rejected: true,
				reason:   "CSR of registration request does not match CSR of existing registration request. If this is expected because for example the device was expected to generate a new identity, then you need to delete the previous device registration.",
			},
			want1: true,
		},
		{
			name: "if there is no certificate yet, this is considered still pending",
			args: args{
				req: &Request{
					DeviceID: "device1",
				},
			},
			pre: func(t *testing.T, ctrl *gomock.Controller, c *mockcontrolplane.MockClient) {
				c.EXPECT().GetDeviceRegistration(gomock.Any(), "device1").Times(1).Return(&dasbootv1alpha1.DeviceRegistration{
					Spec: dasbootv1alpha1.DeviceRegistrationSpec{
						CSR: []byte("csr1"),
					},
				}, nil)
			},
			want:  &cert{},
			want1: true,
		},
		{
			name: "if there is a certificate yet, the public keys must match, otherwise we consider this still pending",
			args: args{
				req: &Request{
					DeviceID: "device1",
				},
			},
			pre: func(t *testing.T, ctrl *gomock.Controller, c *mockcontrolplane.MockClient) {
				c.EXPECT().GetDeviceRegistration(gomock.Any(), "device1").Times(1).Return(&dasbootv1alpha1.DeviceRegistration{
					Spec: dasbootv1alpha1.DeviceRegistrationSpec{
						CSR: csr2,
					},
					Status: dasbootv1alpha1.DeviceRegistrationStatus{
						Certificate: cert1,
					},
				}, nil)
			},
			want:  &cert{},
			want1: true,
		},
		{
			name: "if there is a certificate yet, and the public keys match, we return the certificate",
			args: args{
				req: &Request{
					DeviceID: "device1",
				},
			},
			pre: func(t *testing.T, ctrl *gomock.Controller, c *mockcontrolplane.MockClient) {
				c.EXPECT().GetDeviceRegistration(gomock.Any(), "device1").Times(1).Return(&dasbootv1alpha1.DeviceRegistration{
					Spec: dasbootv1alpha1.DeviceRegistrationSpec{
						CSR: csr1,
					},
					Status: dasbootv1alpha1.DeviceRegistrationStatus{
						Certificate: cert1,
					},
				}, nil)
			},
			want: &cert{
				der:      cert1,
				reason:   "issued by registration-controller",
				rejected: false,
			},
			want1: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockclient := mockcontrolplane.NewMockClient(ctrl)
			p := &Processor{
				cpc: mockclient,
			}
			if tt.pre != nil {
				tt.pre(t, ctrl, mockclient)
			}

			got, got1 := p.getRequestWithControlPlane(ctx, tt.args.req)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Processor.getRequestWithControlPlane() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("Processor.getRequestWithControlPlane() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
