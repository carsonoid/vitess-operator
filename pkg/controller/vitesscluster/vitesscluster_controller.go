package vitesscluster

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	// "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	vitessv1alpha2 "vitess.io/vitess-operator/pkg/apis/vitess/v1alpha2"
	keyspace_controller "vitess.io/vitess-operator/pkg/controller/vitesskeyspace"
	lockserver_controller "vitess.io/vitess-operator/pkg/controller/vitesslockserver"
)

var log = logf.Log.WithName("controller_vitesscluster")

// Add creates a new VitessCluster Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileVitessCluster{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("vitesscluster-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource VitessCluster
	err = c.Watch(&source.Kind{Type: &vitessv1alpha2.VitessCluster{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to VitessCells and requeue the owner VitessCluster
	err = c.Watch(&source.Kind{Type: &vitessv1alpha2.VitessCell{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &vitessv1alpha2.VitessCluster{},
	})
	if err != nil {
		return err
	}

	// Watch for changes to VitessKeyspaces and requeue the owner VitessCluster
	err = c.Watch(&source.Kind{Type: &vitessv1alpha2.VitessKeyspace{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &vitessv1alpha2.VitessCluster{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileVitessCluster{}

// ReconcileVitessCluster reconciles a VitessCluster object
type ReconcileVitessCluster struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a VitessCluster object and makes changes based on the state read
// and what is in the VitessCluster.Spec
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileVitessCluster) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling VitessCluster")

	// Fetch the VitessCluster instance
	vc := &vitessv1alpha2.VitessCluster{}
	err := r.client.Get(context.TODO(), request.NamespacedName, vc)
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

	// Sanity Checks
	// If both Lockserver and LockserverRef are given, error without requeue
	if vc.Spec.Lockserver != nil && vc.Spec.LockserverRef != nil {
		return reconcile.Result{}, fmt.Errorf("Cannot specify both a lockserver and lockserverRef")
	}

	// Reconcile Lockserver
	if result, err := r.ReconcileClusterLockserver(request, vc, reqLogger); err != nil {
		reqLogger.Info("Error Reconciling Lockserver")
		return result, err
	} else if result.Requeue {
		reqLogger.Info("Requeue after reconciling Lockserver")
		return result, nil
	}

	// Reconcile VitessCells

	// Reconcile embedded VitessCells
	if len(vc.Spec.Cells) != 0 {
		reqLogger.Info("Provisioning Cells")
	}

	// TODO reconcile VitessCells from selectors

	// Reconcile VitessKeyspaces

	// Reconcile embedded VitessKeyspaces
	if result, err := r.ReconcileClusterKeyspaces(r.client, request, vc, reqLogger); err != nil {
		reqLogger.Info("Error Reconciling Keyspaces")
		return result, err
	} else if result.Requeue {
		reqLogger.Info("Requeue after reconciling Keyspaces")
		return result, nil
	}

	// TODO reconcile VitessKeyspaces from selectors

	// Nothing to do - don't reqeue
	reqLogger.Info("Skip reconcile: all managed services in sync")
	return reconcile.Result{}, nil
}

func (r *ReconcileVitessCluster) ReconcileClusterLockserver(request reconcile.Request, vc *vitessv1alpha2.VitessCluster, upstreamLog logr.Logger) (reconcile.Result, error) {
	reqLogger := upstreamLog.WithValues()
	reqLogger.Info("Reconciling Embedded Lockserver")

	if vc.Spec.Lockserver != nil {
		// Build a complete VitessLockserver
		vl := &vitessv1alpha2.VitessLockserver{
			ObjectMeta: metav1.ObjectMeta{
				Name:      vc.GetName(),
				Namespace: vc.GetNamespace(),
			},
			Spec: *vc.Spec.Lockserver.DeepCopy(),
		}

		if vc.Status.Lockserver != nil {
			// If status is not empty, deepcopy it into the tmp object
			vc.Status.Lockserver.DeepCopyInto(&vl.Status)
		}

		// Run it through the controller's reconcile func
		recResult, recErr := lockserver_controller.ReconcileObject(vl, reqLogger)

		// Split and store the spec and status in the parent VitessCluster
		vc.Spec.Lockserver = vl.Spec.DeepCopy()
		vc.Status.Lockserver = vl.Status.DeepCopy()

		if err := r.client.Status().Update(context.TODO(), vc); err != nil {
			reqLogger.Error(err, "Failed to update VitessCluster status after lockserver change.")
			return reconcile.Result{}, err
		}

		// Reque if needed
		if recErr != nil || recResult.Requeue {
			return recResult, recErr
		}
	}

	// Fetch the lockserver from Ref if given
	if vc.Spec.LockserverRef != nil {
		// Since Lockserver and Lockserver Ref are mutually-exclusive, it should be safe
		// to simply populate the Lockserver struct member with a pointer to the fetched lockserver
		// get this pointer manully for now
		// TODO use an object cache to make the ref
		ls := &vitessv1alpha2.VitessLockserver{}
		err := r.client.Get(context.TODO(), types.NamespacedName{Name: vc.Spec.LockserverRef.Name, Namespace: request.NamespacedName.Namespace}, ls)
		if err != nil {
			if errors.IsNotFound(err) {
				// If the referenced Lockserver is not found, error out and requeue
				return reconcile.Result{Requeue: true}, fmt.Errorf("Lockserver referenced by lockserverRef %s not found", vc.Spec.LockserverRef.Name)
			}
			// Error reading the object - requeue the request.
			return reconcile.Result{Requeue: true}, err
		}
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileVitessCluster) ReconcileClusterKeyspaces(client client.Client, request reconcile.Request, vc *vitessv1alpha2.VitessCluster, upstreamLog logr.Logger) (reconcile.Result, error) {
	reqLogger := upstreamLog.WithValues()

	// Handle embedded keyspaces
	for keyspaceName, keyspaceSpec := range vc.Spec.Keyspaces {
		reqLogger.Info(fmt.Sprintf("Reconciling Embedded Keyspace %s", keyspaceName))

		// make sure status map is initialized
		if vc.Status.Keyspaces == nil {
			vc.Status.Keyspaces = make(map[string]*vitessv1alpha2.VitessKeyspaceStatus)
		}

		// Build a complete VitessKeyspace
		vk := &vitessv1alpha2.VitessKeyspace{
			ObjectMeta: metav1.ObjectMeta{
				Name:      vc.GetName() + "-" + keyspaceName,
				Namespace: vc.GetNamespace(),
			},
			Spec: *keyspaceSpec.DeepCopy(),
		}

		if keyspaceStatus, ok := vc.Status.Keyspaces[keyspaceName]; ok {
			// If status is not empty, deepcopy it into the tmp object
			keyspaceStatus.DeepCopyInto(&vk.Status)
		}

		// Run it through the controller's reconcile func
		// Run it through the controller's reconcile func
		keyspaceReconciler := keyspace_controller.NewReconciler(r.client, r.scheme)
		recResult, recErr := keyspaceReconciler.ReconcileObject(client, request, vk, reqLogger)

		// Split and store the spec and status in the parent VitessCluster
		vc.Spec.Keyspaces[keyspaceName] = *vk.Spec.DeepCopy()
		vc.Status.Keyspaces[keyspaceName] = vk.Status.DeepCopy()

		if err := r.client.Status().Update(context.TODO(), vc); err != nil {
			reqLogger.Error(err, "Failed to update VitessCluster status after Keyspace change.")
			return reconcile.Result{}, err
		}

		// Reque if needed
		if recErr != nil || recResult.Requeue {
			return recResult, recErr
		}
	}

	// for _, keyspaceSelector := range vc.Spec.KeyspaceSelector {
	// 	// TODO Fetch the Keyspaces from the selector
	// }

	return reconcile.Result{}, nil
}
