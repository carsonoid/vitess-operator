package vitesscluster

import (
	"context"
	"fmt"

	vitessv1alpha2 "github.com/vitessio/vitess-operator/pkg/apis/vitess/v1alpha2"
	"k8s.io/apimachinery/pkg/api/errors"
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
)

var log = logf.Log.WithName("controller_vitesscluster")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

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

	// Provision Lockserver when requested if it's not already created
	if vc.Spec.Lockserver != nil && vc.Spec.Lockserver.Spec.Provision {
		reqLogger.Info("Provisioning Lockserver")
	}

	// Fetch the lockserver from Ref if givien
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

	// Reconcile VitessCells

	// Reconcile embedded VitessCells
	if len(vc.Spec.Cells) != 0 {
		reqLogger.Info("Provisioning Cells")
	}

	// TODO reconcile VitessCells from selectors

	// Reconcile VitessKeyspaces

	// Reconcile embedded VitessKeyspaces
	if len(vc.Spec.Keyspaces) != 0 {
		reqLogger.Info("Provisioning Keyspaces")
	}

	// TODO reconcile VitessKeyspaces from selectors

	// pod := newPodForCR(instance)

	// // Set VitessCluster instance as the owner and controller
	// if err := controllerutil.SetControllerReference(instance, pod, r.scheme); err != nil {u
	// 	return reconcile.Result{}, err
	// }

	// // Check if this Pod already exists
	// found := &corev1.Pod{}
	// err = r.client.Get(context.TODO(), types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}, found)
	// if err != nil && errors.IsNotFound(err) {
	// 	reqLogger.Info("Creating a new Pod", "Pod.Namespace", pod.Namespace, "Pod.Name", pod.Name)
	// 	err = r.client.Create(context.TODO(), pod)
	// 	if err != nil {
	// 		return reconcile.Result{}, err
	// 	}

	// 	// Pod created successfully - don't requeue
	// 	return reconcile.Result{}, nil
	// } else if err != nil {
	// 	return reconcile.Result{}, err
	// }

	// // Pod already exists - don't requeue
	// reqLogger.Info("Skip reconcile: Pod already exists", "Pod.Namespace", found.Namespace, "Pod.Name", found.Name)
	// return reconcile.Result{}, nil

	// Nothing to do - don't reqeue
	reqLogger.Info("Skip reconcile: all managed services in sync")
	return reconcile.Result{}, nil
}

// newPodForCR returns a busybox pod with the same name/namespace as the cr
// func newVitessDeployment(cr *vitessv1alpha2.VitessCluster) *corev1.Pod {
// 	labels := map[string]string{
// 		"app": cr.Name,
// 	}
// 	return &corev1.Pod{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name:      cr.Name + "-pod",
// 			Namespace: cr.Namespace,
// 			Labels:    labels,
// 		},
// 		Spec: corev1.PodSpec{
// 			Containers: []corev1.Container{
// 				{
// 					Name:    "busybox",
// 					Image:   "busybox",
// 					Command: []string{"sleep", "3600"},
// 				},
// 			},
// 		},
// 	}
// }
