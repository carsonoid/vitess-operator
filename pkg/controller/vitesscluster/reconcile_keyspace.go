package vitesscluster

import (
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	vitessv1alpha2 "vitess.io/vitess-operator/pkg/apis/vitess/v1alpha2"
)

func (r *ReconcileVitessCluster) ReconcileKeyspace(keyspace *vitessv1alpha2.VitessKeyspace) (reconcile.Result, error) {
	log.Info("Reconciling Keyspace", "Namespace", keyspace.GetNamespace(), "VitessCluster.Name", keyspace.GetCluster().GetName(), "Keyspace.Name", keyspace.GetName())

	// Reconcile all shards
	for _, shard := range keyspace.GetEmbeddedShards() {
		if result, err := r.ReconcileShard(shard); err != nil {
			return result, err
		}
	}

	return reconcile.Result{}, nil
}
