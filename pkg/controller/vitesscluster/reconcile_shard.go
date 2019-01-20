package vitesscluster

import (
	"context"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	vitessv1alpha2 "vitess.io/vitess-operator/pkg/apis/vitess/v1alpha2"
	"vitess.io/vitess-operator/pkg/util/scripts"
)

func (r *ReconcileVitessCluster) ReconcileShard(shard *vitessv1alpha2.VitessShard) (reconcile.Result, error) {
	if r, err := r.ReconcileShardResources(shard); err != nil {
		return r, err
	}

	// Reconcile all shards
	for _, tablet := range shard.GetEmbeddedTablets() {
		if result, err := r.ReconcileTablet(tablet); err != nil {
			return result, err
		}
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileVitessCluster) ReconcileShardResources(shard *vitessv1alpha2.VitessShard) (reconcile.Result, error) {
	// Each shard needs a master election job
	job, jobErr := GetShardInitMasterJob(shard)
	if jobErr != nil {
		log.Error(jobErr, "failed to generate MasterElect Job for VitessShard", "VitessShard.Namespace", shard.GetNamespace(), "VitessShard.Name", shard.GetNamespace())
		return reconcile.Result{}, jobErr
	}

	found := &batchv1.Job{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: job.GetName(), Namespace: job.GetNamespace()}, found)
	if err != nil && errors.IsNotFound(err) {
		controllerutil.SetControllerReference(shard.GetCluster(), job, r.scheme)
		err = r.client.Create(context.TODO(), job)
		if err != nil {
			return reconcile.Result{}, err
		}
		// Job created successfully - return and requeue
		return reconcile.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "failed to get Job")
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func GetShardInitMasterJob(shard *vitessv1alpha2.VitessShard) (*batchv1.Job, error) {
	jobName := shard.GetScopedName("init-shard-master")

	scripts := scripts.NewContainerScriptGenerator("init_shard_master", shard)
	if err := scripts.Generate(); err != nil {
		return nil, err
	}

	jobLabels := map[string]string{
		"app":                "vitess",
		"cluster":            shard.GetCluster().GetName(),
		"keyspace":           shard.GetKeyspace().GetName(),
		"shard":              shard.GetName(),
		"component":          "vttablet",
		"initShardMasterJob": "true",
		"job-name":           jobName,
	}

	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: shard.GetCluster().GetNamespace(),
			Labels:    jobLabels,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: getInt32Ptr(1),
			Completions:  getInt32Ptr(1),
			Parallelism:  getInt32Ptr(1),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: jobLabels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "init-shard-master",
							Image: "vitess/vtctlclient:helm-1.0.3", // TODO use CRD w/default
							Command: []string{
								"bash",
							},
							Args: []string{
								"-c",
								scripts.Start,
							},
						},
					},
					RestartPolicy: corev1.RestartPolicyOnFailure,
				},
			},
		},
	}, nil
}
