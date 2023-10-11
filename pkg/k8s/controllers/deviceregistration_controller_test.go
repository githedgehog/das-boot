package controllers

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha1"
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
	"go.githedgehog.com/dasboot/test/mock/controller-runtime/mockclient"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func newDeviceRegistrationReconciler(client client.Client) *DeviceRegistrationReconciler {
	// create scheme
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(dasbootv1alpha1.AddToScheme(scheme))

	// generate a key and cert
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(mathrand.Int63()),
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

	return &DeviceRegistrationReconciler{
		Client: client,
		Scheme: scheme,
		Key:    key,
		Cert:   cert,
	}
}

func newCSRPubKeyAndCert(id string, caKey *ecdsa.PrivateKey, caCert *x509.Certificate) ([]byte, *ecdsa.PublicKey, []byte) {
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

func TestDeviceRegistrationReconciler_Reconcile(t *testing.T) {
	type args struct {
		req ctrl.Request
	}

	tests := []struct {
		name    string
		args    args
		pre     func(t *testing.T, ctrl *gomock.Controller, r *DeviceRegistrationReconciler, c *mockclient.MockClient)
		want    ctrl.Result
		wantErr bool
	}{
		{
			name: "new device registration object created",
			args: args{
				req: ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      "test-device",
						Namespace: "default",
					},
				},
			},
			pre: func(t *testing.T, ctrl *gomock.Controller, r *DeviceRegistrationReconciler, c *mockclient.MockClient) {
				c.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).DoAndReturn(func(ctx context.Context, key types.NamespacedName, obj client.Object, opts ...client.GetOption) error {
					o := obj.(*dasbootv1alpha1.DeviceRegistration)
					csr, _, _ := newCSRPubKeyAndCert("test-device", r.Key, r.Cert)
					*o = dasbootv1alpha1.DeviceRegistration{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-device",
							Namespace: "default",
						},
						Spec: dasbootv1alpha1.DeviceRegistrationSpec{
							CSR: csr,
						},
					}
					return nil
				})
				w := mockclient.NewMockSubResourceWriter(ctrl)
				w.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).DoAndReturn(func(ctx context.Context, obj client.Object, opts ...client.SubResourceUpdateOption) error {
					o := obj.(*dasbootv1alpha1.DeviceRegistration)
					if o.Status.Certificate == nil {
						return fmt.Errorf("expected certificate to be set")
					}
					return nil
				})
				c.EXPECT().Status().DoAndReturn(func() client.SubResourceWriter {
					return w
				})
			},
			want:    ctrl.Result{},
			wantErr: false,
		},
		{
			name: "new CSR and updating certificate",
			args: args{
				req: ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      "test-device",
						Namespace: "default",
					},
				},
			},
			pre: func(t *testing.T, ctrl *gomock.Controller, r *DeviceRegistrationReconciler, c *mockclient.MockClient) {
				_, _, certPrev := newCSRPubKeyAndCert("test-device", r.Key, r.Cert)
				csr, newPub, _ := newCSRPubKeyAndCert("test-device", r.Key, r.Cert)
				c.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).DoAndReturn(func(ctx context.Context, key types.NamespacedName, obj client.Object, opts ...client.GetOption) error {
					o := obj.(*dasbootv1alpha1.DeviceRegistration)
					*o = dasbootv1alpha1.DeviceRegistration{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-device",
							Namespace: "default",
						},
						Spec: dasbootv1alpha1.DeviceRegistrationSpec{
							CSR: csr,
						},
						Status: dasbootv1alpha1.DeviceRegistrationStatus{
							Certificate: certPrev,
						},
					}
					return nil
				})
				w := mockclient.NewMockSubResourceWriter(ctrl)
				w.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).DoAndReturn(func(ctx context.Context, obj client.Object, opts ...client.SubResourceUpdateOption) error {
					o := obj.(*dasbootv1alpha1.DeviceRegistration)
					cert, err := x509.ParseCertificate(o.Status.Certificate)
					if err != nil {
						return err
					}
					if !cert.PublicKey.(*ecdsa.PublicKey).Equal(newPub) {
						return fmt.Errorf("expected certificate public key to be newPub")
					}
					return nil
				})
				c.EXPECT().Status().DoAndReturn(func() client.SubResourceWriter {
					return w
				})
			},
			want:    ctrl.Result{},
			wantErr: false,
		},
		{
			name: "CSR pub key and certificate pub key match, no need to regenerate",
			args: args{
				req: ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      "test-device",
						Namespace: "default",
					},
				},
			},
			pre: func(t *testing.T, ctrl *gomock.Controller, r *DeviceRegistrationReconciler, c *mockclient.MockClient) {
				csr, _, cert := newCSRPubKeyAndCert("test-device", r.Key, r.Cert)
				c.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).DoAndReturn(func(ctx context.Context, key types.NamespacedName, obj client.Object, opts ...client.GetOption) error {
					o := obj.(*dasbootv1alpha1.DeviceRegistration)
					*o = dasbootv1alpha1.DeviceRegistration{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-device",
							Namespace: "default",
						},
						Spec: dasbootv1alpha1.DeviceRegistrationSpec{
							CSR: csr,
						},
						Status: dasbootv1alpha1.DeviceRegistrationStatus{
							Certificate: cert,
						},
					}
					return nil
				})
			},
			want:    ctrl.Result{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockclient := mockclient.NewMockClient(ctrl)
			r := newDeviceRegistrationReconciler(mockclient)
			if tt.pre != nil {
				tt.pre(t, ctrl, r, mockclient)
			}
			got, err := r.Reconcile(ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeviceRegistrationReconciler.Reconcile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DeviceRegistrationReconciler.Reconcile() = %v, want %v", got, tt.want)
			}
		})
	}
}
