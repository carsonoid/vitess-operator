package vitesscluster

import (
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	vitessv1alpha2 "vitess.io/vitess-operator/pkg/apis/vitess/v1alpha2"
	lockserver_controller "vitess.io/vitess-operator/pkg/controller/vitesslockserver"
)

// ReconcileClusterResources should only be called against a fully-populated and verified VitessCluster object
func (r *ReconcileVitessCluster) ReconcileClusterResources(cluster *vitessv1alpha2.VitessCluster) (reconcile.Result, error) {
	if r, err := r.ReconcileClusterLockserver(cluster); err != nil || r.Requeue {
		return r, err
	}

	for _, cell := range cluster.Cells() {
		if r, err := r.ReconcileCell(cell); err != nil || r.Requeue {
			return r, err
		}
	}

	for _, keyspace := range cluster.Keyspaces() {
		if r, err := r.ReconcileKeyspace(keyspace); err != nil || r.Requeue {
			return r, err
		}
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileVitessCluster) ReconcileClusterLockserver(cluster *vitessv1alpha2.VitessCluster) (reconcile.Result, error) {
	log.Info("Reconciling Embedded Lockserver")

	// Build a complete VitessLockserver
	lockserver := cluster.Spec.Lockserver.DeepCopy()

	if cluster.Status.Lockserver != nil {
		// If status is not empty, deepcopy it into the tmp object
		cluster.Status.Lockserver.DeepCopyInto(&lockserver.Status)
	}

	// Run it through the controller's reconcile func
	recResult, recErr := lockserver_controller.ReconcileObject(lockserver, log)

	// Split and store the spec and status in the parent VitessCluster
	cluster.Spec.Lockserver = lockserver.DeepCopy()
	cluster.Status.Lockserver = lockserver.Status.DeepCopy()

	// Using the  split client here breaks the cluster normalization
	// TODO Fix and re-enable

	// if err := r.client.Status().Update(context.TODO(), cluster); err != nil {
	// 	log.Error(err, "Failed to update VitessCluster status after lockserver change.")
	// 	return reconcile.Result{}, err
	// }

	return recResult, recErr
}
