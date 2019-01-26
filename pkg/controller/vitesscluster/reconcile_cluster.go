package vitesscluster

import (
	// "context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	vitessv1alpha2 "vitess.io/vitess-operator/pkg/apis/vitess/v1alpha2"
	lockserver_controller "vitess.io/vitess-operator/pkg/controller/vitesslockserver"
)

// ReconcileClusterResources should only be called against a fully-populated and verified VitessCluster object
func (r *ReconcileVitessCluster) ReconcileClusterResources(cluster *vitessv1alpha2.VitessCluster) (reconcile.Result, error) {
	if r, err := r.ReconcileClusterLockserver(cluster); err != nil || r.Requeue {
		return r, err
	}

	for _, cell := range cluster.GetEmbeddedCells() {
		if r, err := r.ReconcileCell(cell); err != nil || r.Requeue {
			return r, err
		}
	}

	for _, keyspace := range cluster.GetEmbeddedKeyspaces() {
		if r, err := r.ReconcileKeyspace(keyspace); err != nil || r.Requeue {
			return r, err
		}
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileVitessCluster) ReconcileClusterLockserver(cluster *vitessv1alpha2.VitessCluster) (reconcile.Result, error) {
	log.Info("Reconciling Embedded Lockserver")

	// Build a complete VitessLockserver
	lockserver := &vitessv1alpha2.VitessLockserver{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.GetName(),
			Namespace: cluster.GetNamespace(),
		},
		Spec: *cluster.Spec.Lockserver.DeepCopy(),
	}

	if cluster.Status.Lockserver != nil {
		// If status is not empty, deepcopy it into the tmp object
		cluster.Status.Lockserver.DeepCopyInto(&lockserver.Status)
	}

	// Run it through the controller's reconcile func
	recResult, recErr := lockserver_controller.ReconcileObject(lockserver, log)

	// Split and store the spec and status in the parent VitessCluster
	cluster.Spec.Lockserver = lockserver.Spec.DeepCopy()
	cluster.Status.Lockserver = lockserver.Status.DeepCopy()

	// Using the  split client here breaks the cluster normalization
	// TODO Fix and re-enable

	// if err := r.client.Status().Update(context.TODO(), cluster); err != nil {
	// 	log.Error(err, "Failed to update VitessCluster status after lockserver change.")
	// 	return reconcile.Result{}, err
	// }

	return recResult, recErr
}

func (r *ReconcileVitessCluster) ReconcileClusterConfigMap(cluster *vitessv1alpha2.VitessCluster) (reconcile.Result, error) {

	// # shared ConfigMap
	// apiVersion: v1
	// kind: ConfigMap
	// metadata:
	//   name: vitess-cm
	// data:
	//   backup.backup_storage_implementation: {{ .backup.backup_storage_implementation }}
	//   backup.gcs_backup_storage_bucket: {{ .backup.gcs_backup_storage_bucket }}
	//   backup.gcs_backup_storage_root: {{ .backup.gcs_backup_storage_root }}
	//   backup.s3_backup_aws_region: {{ .backup.s3_backup_aws_region }}
	//   backup.s3_backup_storage_bucket: {{ .backup.s3_backup_storage_bucket }}
	//   backup.s3_backup_storage_root: {{ .backup.s3_backup_storage_root }}
	//   backup.s3_backup_server_side_encryption: {{ .backup.s3_backup_server_side_encryption }}

	//   db.flavor: {{ $.Values.vttablet.flavor }}
	// {{ end }} # end with config
}

func GetClusterConfigMap(cluster *vitessv1alpha2.VitessCluster) (*corev1.ConfigMap, error) {
	name := cluster.GetScopedName("vitess", "cm")

	labels := map[string]string{
		"app":     "vitess",
		"cluster": shard.GetCluster().GetName(),
	}

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cluster.GetNamespace(),
			Labels:    labels,
		},
		Data: map[string]string{
			"backup.backup_storage_implementation":    "{{ .backup.backup_storage_implementation }}",
			"backup.gcs_backup_storage_bucket":        "{{ .backup.gcs_backup_storage_bucket }}",
			"backup.gcs_backup_storage_root":          "{{ .backup.gcs_backup_storage_root }}",
			"backup.s3_backup_aws_region":             "{{ .backup.s3_backup_aws_region }}",
			"backup.s3_backup_storage_bucket":         "{{ .backup.s3_backup_storage_bucket }}",
			"backup.s3_backup_storage_root":           "{{ .backup.s3_backup_storage_root }}",
			"backup.s3_backup_server_side_encryption": "{{ .backup.s3_backup_server_side_encryption }}",
			"": "",
		},
	}, nil
}
