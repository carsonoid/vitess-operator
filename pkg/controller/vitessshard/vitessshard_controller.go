package vitessshard

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	vitessv1alpha2 "vitess.io/vitess-operator/pkg/apis/vitess/v1alpha2"
	tablet_controller "vitess.io/vitess-operator/pkg/controller/vitesstablet"
)

var log = logf.Log.WithName("controller_vitessshard")

// Add creates a new VitessShard Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileVitessShard{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// NewReconciler returns a new reconcile.Reconciler
func NewReconciler(client client.Client, scheme *runtime.Scheme) *ReconcileVitessShard {
	return &ReconcileVitessShard{client: client, scheme: scheme}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("vitessshard-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource VitessShard
	err = c.Watch(&source.Kind{Type: &vitessv1alpha2.VitessShard{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileVitessShard{}

// ReconcileVitessShard reconciles a VitessShard object
type ReconcileVitessShard struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a VitessShard object and makes changes based on the state read
// and what is in the VitessShard.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileVitessShard) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling VitessShard")

	// // Fetch the VitessShard instance
	// instance := &vitessv1alpha2.VitessShard{}
	// err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	// if err != nil {
	// 	if errors.IsNotFound(err) {
	// 		// Request object not found, could have been deleted after reconcile request.
	// 		// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
	// 		// Return and don't requeue
	// 		return reconcile.Result{}, nil
	// 	}
	// 	// Error reading the object - requeue the request.
	// 	return reconcile.Result{}, err
	// }

	// rr, err := r.ReconcileObject(r.client, request, instance, nil, reqLogger)

	// if err := r.client.Status().Update(context.TODO(), instance); err != nil {
	// 	reqLogger.Error(err, "Failed to update VitessShard status.")
	// 	return reconcile.Result{Requeue: true}, err
	// }

	return reconcile.Result{}, nil
	// return rr, err
}

// ReconcileObject does all the actual reconcile work
func (r *ReconcileVitessShard) ReconcileObject(client client.Client, request reconcile.Request, instance *vitessv1alpha2.VitessShard, parent metav1.Object, upstreamLog logr.Logger) (reconcile.Result, error) {
	reqLogger := upstreamLog.WithValues()
	reqLogger.Info("Reconciling VitessShard")

	// if no upstream parent, look at me, i am the parent now
	updateStatus := false
	if parent == nil {
		parent = instance
		updateStatus = true
	}

	result, err := r.ReconcileClusterTablets(client, request, instance, parent, upstreamLog)
	if err != nil || result.Requeue {
		return result, err
	}

	if updateStatus {
		if err := r.client.Update(context.TODO(), instance); err != nil {
			reqLogger.Error(err, "Failed to update VitessShard")
			return reconcile.Result{}, err
		}
	}

	instance.Status.State = "Ready"
	return reconcile.Result{}, nil
}

func (r *ReconcileVitessShard) ReconcileClusterTablets(client client.Client, request reconcile.Request, vs *vitessv1alpha2.VitessShard, parent metav1.Object, upstreamLog logr.Logger) (reconcile.Result, error) {
	reqLogger := upstreamLog.WithValues()

	// Handle embedded tablets
	for tabletName, tabletSpec := range vs.Spec.Tablets {
		reqLogger.Info(fmt.Sprintf("Reconciling embedded tablet %s", tabletName))

		// make sure status map is initialized
		if vs.Status.Tablets == nil {
			vs.Status.Tablets = make(map[string]*vitessv1alpha2.VitessTabletStatus)
		}

		// Build a complete VitessTablet
		vt := &vitessv1alpha2.VitessTablet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      vs.GetName() + "-" + tabletName,
				Namespace: vs.GetNamespace(),
			},
			Spec: *tabletSpec.DeepCopy(),
		}

		if tabletStatus, ok := vs.Status.Tablets[tabletName]; ok {
			// If status is not empty, deepcopy it into the tmp object
			tabletStatus.DeepCopyInto(&vt.Status)
		}

		// Run it through the controller's reconcile func
		recResult, recErr := tablet_controller.ReconcileObject(client, request, vt, reqLogger)

		// Split and store the spec and status in the parent Shard
		vs.Spec.Tablets[tabletName] = *vt.Spec.DeepCopy()
		vs.Status.Tablets[tabletName] = vt.Status.DeepCopy()

		// Reque if needed
		if recErr != nil || recResult.Requeue {
			return recResult, recErr
		}
	}

	// for _, keyspaceSelector := range vc.Spec.KeyspaceSelector {
	// 	// TODO Fetch the Shards from the selector
	// }

	return reconcile.Result{}, nil
}
