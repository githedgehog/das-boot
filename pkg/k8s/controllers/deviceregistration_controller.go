package controllers

import (
	"context"
	"crypto/ecdsa"
	"crypto/x509"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	dasbootv1alpha1 "go.githedgehog.com/dasboot/pkg/k8s/api/v1alpha1"
)

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
	_ = log.FromContext(ctx)

	// TODO(user): your logic here

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DeviceRegistrationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dasbootv1alpha1.DeviceRegistration{}).
		Complete(r)
}
