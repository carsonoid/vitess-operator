package vitesskeyspace

import (
	"fmt"

	"github.com/go-logr/logr"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	// "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	// "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	vitessv1alpha2 "vitess.io/vitess-operator/pkg/apis/vitess/v1alpha2"
	shard_controller "vitess.io/vitess-operator/pkg/controller/vitessshard"
)

var log = logf.Log.WithName("controller_vitesskeyspace")

// Add creates a new VitessKeyspace Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileVitessKeyspace{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// NewReconciler returns a new reconcile.Reconciler
func NewReconciler(client client.Client, scheme *runtime.Scheme) *ReconcileVitessKeyspace {
	return &ReconcileVitessKeyspace{client: client, scheme: scheme}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("vitesskeyspace-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource VitessKeyspace
	err = c.Watch(&source.Kind{Type: &vitessv1alpha2.VitessKeyspace{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileVitessKeyspace{}

// ReconcileVitessKeyspace reconciles a VitessKeyspace object
type ReconcileVitessKeyspace struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a VitessKeyspace object and makes changes based on the state read
// and what is in the VitessKeyspace.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileVitessKeyspace) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling VitessKeyspace")

	// // Fetch the VitessKeyspace instance
	// instance := &vitessv1alpha2.VitessKeyspace{}
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

	// rr, err := r.ReconcileObject(r.client, request, instance, instance, reqLogger)

	// if err := r.client.Status().Update(context.TODO(), instance); err != nil {
	// 	reqLogger.Error(err, "Failed to update VitessKeyspace status.")
	// 	return reconcile.Result{Requeue: true}, err
	// }

	return reconcile.Result{}, nil
	// return rr, err
}

// ReconcileObject does all the actual reconcile work
func (r *ReconcileVitessKeyspace) ReconcileObject(client client.Client, request reconcile.Request, instance *vitessv1alpha2.VitessKeyspace, parent metav1.Object, upstreamLog logr.Logger) (reconcile.Result, error) {
	reqLogger := upstreamLog.WithValues()
	reqLogger.Info("Reconciling VitessKeyspace")

	// if no upstream parent, look at me, i am the parent now
	if parent == nil {
		parent = instance
	}

	result, err := r.ReconcileClusterShards(client, request, instance, parent, upstreamLog)
	if err != nil || result.Requeue {
		return result, err
	}

	// TODO actual reconcile
	instance.Status.State = "Ready"
	return reconcile.Result{}, nil
}

func (r *ReconcileVitessKeyspace) ReconcileClusterShards(client client.Client, request reconcile.Request, vk *vitessv1alpha2.VitessKeyspace, parent metav1.Object, upstreamLog logr.Logger) (reconcile.Result, error) {
	reqLogger := upstreamLog.WithValues()

	// Handle embedded keyspaces
	for shardName, shardSpec := range vk.Spec.Shards {
		reqLogger.Info(fmt.Sprintf("Reconciling embedded shard %s", shardName))

		// make sure status map is initialized
		if vk.Status.Shards == nil {
			vk.Status.Shards = make(map[string]*vitessv1alpha2.VitessShardStatus)
		}

		// Build a complete VitessShard
		vs := &vitessv1alpha2.VitessShard{
			ObjectMeta: metav1.ObjectMeta{
				Name:      vk.GetName() + "-" + shardName,
				Namespace: vk.GetNamespace(),
			},
			Spec: *shardSpec.DeepCopy(),
		}

		if keyspaceStatus, ok := vk.Status.Shards[shardName]; ok {
			// If status is not empty, deepcopy it into the tmp object
			keyspaceStatus.DeepCopyInto(&vs.Status)
		}

		// Run it through the controller's reconcile func
		shardReconciler := shard_controller.NewReconciler(r.client, r.scheme)
		recResult, recErr := shardReconciler.ReconcileObject(client, request, vs, parent, reqLogger)

		// Split and store the spec and status in the parent VitessCluster
		vk.Spec.Shards[shardName] = vs.Spec.DeepCopy()
		vk.Status.Shards[shardName] = vs.Status.DeepCopy()

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
