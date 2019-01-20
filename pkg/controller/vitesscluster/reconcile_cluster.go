package vitesscluster

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	vitessv1alpha2 "vitess.io/vitess-operator/pkg/apis/vitess/v1alpha2"
	lockserver_controller "vitess.io/vitess-operator/pkg/controller/vitesslockserver"
)

// ReconcileClusterResources should only be called against a fully-populated and verified VitessCluster object
func (r *ReconcileVitessCluster) ReconcileClusterResources(client client.Client, request reconcile.Request, vc *vitessv1alpha2.VitessCluster) (reconcile.Result, error) {
	reconciledShards := make(map[string]struct{})
	reconciledCells := make(map[string]struct{})
	for _, tablet := range vc.GetEmbeddedTablets() {
		// Create the resources for each tablet
		if result, err := r.ReconcileClusterTablet(request, vc, tablet); err != nil {
			return result, err
		}

		// Reconcile each shard once
		// Right now this is done during tablet reconciliation as a quick-and-dirty
		// way to have all parent information already set up and available.
		// TODO refactor to something cleaner
		if _, ok := reconciledShards[tablet.GetShard().GetName()]; !ok {
			if result, err := r.ReconcileTabletShard(client, request, tablet); err != nil {
				return result, err
			}
			// populate the map so we don't reconcile again
			reconciledShards[tablet.GetShard().GetName()] = struct{}{}
		}

		// Reconcile each cell once
		// Right now this is done during tablet reconciliation as a quick-and-dirty
		// way to have all parent information already set up and available.
		// TODO refactor to something cleaner
		if _, ok := reconciledCells[tablet.GetCell().GetName()]; !ok {
			if result, err := r.ReconcileClusterCellVtctld(request, tablet); err != nil {
				return result, err
			}
			// populate the map so we don't reconcile again
			reconciledCells[tablet.GetCell().GetName()] = struct{}{}
		}
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileVitessCluster) ReconcileClusterLockserver(request reconcile.Request, vc *vitessv1alpha2.VitessCluster) (reconcile.Result, error) {
	reqLogger := log.WithValues()
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

	return reconcile.Result{}, nil
}
