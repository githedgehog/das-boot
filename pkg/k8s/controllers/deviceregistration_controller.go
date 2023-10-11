package controllers

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha1" //nolint: gosec
	"crypto/x509"
	"fmt"
	"math/big"
	mathrand "math/rand"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	dasbootv1alpha1 "go.githedgehog.com/dasboot/pkg/k8s/api/v1alpha1"
)

var certificateValidity = time.Hour * 24 * 360

//go:generate mockgen -destination ../../../test/mock/controller-runtime/mockclient/client.go -package mockclient sigs.k8s.io/controller-runtime/pkg/client Client
//go:generate mockgen -destination ../../../test/mock/controller-runtime/mockclient/subresource_writer.go -package mockclient sigs.k8s.io/controller-runtime/pkg/client SubResourceWriter

// DeviceRegistrationReconciler reconciles a DeviceRegistration object
type DeviceRegistrationReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	// Key (CA) used to sign requests with
	Key *ecdsa.PrivateKey

	// Public Cert (CA used to sign requests with
	Cert *x509.Certificate
}

//+kubebuilder:rbac:groups=dasboot.githedgehog.com,resources=deviceregistrations,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=dasboot.githedgehog.com,resources=deviceregistrations/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=dasboot.githedgehog.com,resources=deviceregistrations/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the DeviceRegistration object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *DeviceRegistrationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)

	// get the reconciling object in question
	var dr dasbootv1alpha1.DeviceRegistration
	if err := r.Get(ctx, req.NamespacedName, &dr); err != nil {
		l.Error(err, "unable to fetch DeviceRegistration")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	l.Info("Got DeviceRegistration", "req", req.NamespacedName, "dr", dr)

	// parse the CSR
	csr, err := x509.ParseCertificateRequest(dr.Spec.CSR)
	if err != nil {
		l.Error(err, "Parsing CSR failed", "req", req.NamespacedName)
		return ctrl.Result{}, err
	}
	if csr.Subject.CommonName == "" {
		err = fmt.Errorf("CN in CSR empty")
		l.Error(err, "Processing CSR", "req", req.NamespacedName)
		return ctrl.Result{}, err
	}
	if csr.Subject.CommonName != req.Name {
		err = fmt.Errorf("device ID mismatch: CN != Device ID (%s != %s)", csr.Subject.CommonName, req.Name)
		l.Error(err, "Processing CSR", "req", req.NamespacedName)
		return ctrl.Result{}, err
	}
	csrPub, ok := csr.PublicKey.(*ecdsa.PublicKey)
	if !ok {
		err = fmt.Errorf("CSR must contain ECDSA key")
		l.Error(err, "Processing CSR", "req", req.NamespacedName)
		return ctrl.Result{}, err
	}
	ecdhCsrPub, err := csrPub.ECDH()
	if err != nil {
		err = fmt.Errorf("cannot convert ECDSA public key to ECDH public key: %w", err)
		l.Error(err, "Processing CSR", "req", req.NamespacedName)
		return ctrl.Result{}, err
	}
	csrPubBytes := ecdhCsrPub.Bytes()

	// check if we need to create a certificate
	if !needToGenerateCertificate(l, &dr, csrPub) {
		l.Info("No need to generate a new certificate", "req", req.NamespacedName)
		return ctrl.Result{}, nil
	}

	l.Info("Generating a new certificate", "req", req.NamespacedName)
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
	signedCert, err := x509.CreateCertificate(rand.Reader, template, r.Cert, csr.PublicKey, r.Key)
	if err != nil {
		l.Error(err, "Signing CSR", "req", req.NamespacedName)
		return ctrl.Result{}, err
	}

	dr.Status.Certificate = signedCert
	if err := r.Status().Update(ctx, &dr); err != nil {
		l.Error(err, "Updating Status failed", "req", req.NamespacedName)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func needToGenerateCertificate(l logr.Logger, dr *dasbootv1alpha1.DeviceRegistration, csrPub *ecdsa.PublicKey) bool {
	// if we have no certificate, we need to generate one
	if len(dr.Status.Certificate) == 0 {
		return true
	}

	// if we have a certificate, parse it first
	cert, err := x509.ParseCertificate(dr.Status.Certificate)
	if err != nil {
		// if we cannot parse the certificate, then we need to regenerate it
		l.Error(err, "needToGenerateCertificate: parsing exisiting certificate failed, generating a new one...")
		return true
	}
	certPub, ok := cert.PublicKey.(*ecdsa.PublicKey)
	if !ok {
		l.Error(err, "needToGenerateCertificate: existing certificate does not contain an ECDSA public key, generating a new one...")
		return true
	}

	// if the public keys match, then we do NOT have to generate a new certificate
	// otherwise it is a new CSR, so we need to generate a new certificate
	return !csrPub.Equal(certPub)
}

// SetupWithManager sets up the controller with the Manager.
func (r *DeviceRegistrationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dasbootv1alpha1.DeviceRegistration{}).
		Complete(r)
}
