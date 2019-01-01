package vitesstablet

import (
	"context"

	"github.com/go-logr/logr"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	// "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
	vitessv1alpha2 "vitess.io/vitess-operator/pkg/apis/vitess/v1alpha2"
)

var log = logf.Log.WithName("controller_vitesstablet")

// Add creates a new VitessTablet Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileVitessTablet{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("vitesstablet-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource VitessTablet
	err = c.Watch(&source.Kind{Type: &vitessv1alpha2.VitessTablet{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileVitessTablet{}

// ReconcileVitessTablet reconciles a VitessTablet object
type ReconcileVitessTablet struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a VitessTablet object and makes changes based on the state read
// and what is in the VitessTablet.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileVitessTablet) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling VitessTablet")

	// Fetch the VitessTablet instance
	instance := &vitessv1alpha2.VitessTablet{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	rr, err := ReconcileObject(r.client, request, instance, reqLogger)

	if err := r.client.Status().Update(context.TODO(), instance); err != nil {
		reqLogger.Error(err, "Failed to update VitessTablet status.")
		return reconcile.Result{Requeue: true}, err
	}

	return rr, err
}

func ReconcileObject(client client.Client, request reconcile.Request, instance *vitessv1alpha2.VitessTablet, upstreamLog logr.Logger) (reconcile.Result, error) {
	reqLogger := upstreamLog.WithValues()
	reqLogger.Info("Reconciling VitessTablet")

	instance.Status.State = "Ready"
	return reconcile.Result{}, nil
}
