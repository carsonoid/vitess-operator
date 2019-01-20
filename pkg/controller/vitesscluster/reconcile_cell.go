package vitesscluster

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	vitessv1alpha2 "vitess.io/vitess-operator/pkg/apis/vitess/v1alpha2"
	"vitess.io/vitess-operator/pkg/util/scripts"
)

func (r *ReconcileVitessCluster) ReconcileCell(cell *vitessv1alpha2.VitessCell) (reconcile.Result, error) {
	// Each shard needs a master election job
	deploy, service, deployErr := GetCellVtctldResources(cell)
	if deployErr != nil {
		log.Error(deployErr, "failed to generate Vtctld Deployment for VitessCell", "VitessCell.Namespace", cell.GetNamespace(), "VitessCell.Name", cell.GetNamespace())
		return reconcile.Result{}, deployErr
	}

	found := &appsv1.Deployment{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: deploy.GetName(), Namespace: deploy.GetNamespace()}, found)
	if err != nil && errors.IsNotFound(err) {
		controllerutil.SetControllerReference(cell.GetCluster(), deploy, r.scheme)
		err = r.client.Create(context.TODO(), deploy)
		if err != nil {
			return reconcile.Result{}, err
		}
		// Job created successfully - return and requeue
		return reconcile.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "failed to get Deployment")
		return reconcile.Result{}, err
	}

	foundCluster := &corev1.Service{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: service.GetName(), Namespace: service.GetNamespace()}, foundCluster)
	if err != nil && errors.IsNotFound(err) {
		controllerutil.SetControllerReference(cell.GetCluster(), service, r.scheme)
		err = r.client.Create(context.TODO(), service)
		if err != nil {
			return reconcile.Result{}, err
		}
		// Job created successfully - return and requeue
		return reconcile.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "failed to get Service")
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func GetCellVtctldResources(cell *vitessv1alpha2.VitessCell) (*appsv1.Deployment, *corev1.Service, error) {
	name := cell.GetScopedName("-vtctld")

	scripts := scripts.NewContainerScriptGenerator("vtctld", cell)
	if err := scripts.Generate(); err != nil {
		return nil, nil, err
	}

	labels := map[string]string{
		"app":       "vitess",
		"cluster":   cell.GetCluster().GetName(),
		"component": "vtctld",
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cell.GetCluster().GetNamespace(),
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			ProgressDeadlineSeconds: getInt32Ptr(1),
			Replicas:                getInt32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "vtctld",
							Image: "vitess/vtctld:helm-1.0.3", // TODO use CRD w/default
							Command: []string{
								"bash",
							},
							Args: []string{
								"-c",
								scripts.Start,
							},
							LivenessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   "/debug/status",
										Port:   intstr.FromInt(15000),
										Scheme: corev1.URISchemeHTTP,
									},
								},
								InitialDelaySeconds: 30,
								TimeoutSeconds:      5,
								PeriodSeconds:       10,
								SuccessThreshold:    1,
								FailureThreshold:    3,
							},
							ReadinessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   "/debug/health",
										Port:   intstr.FromInt(15000),
										Scheme: corev1.URISchemeHTTP,
									},
								},
								InitialDelaySeconds: 30,
								TimeoutSeconds:      5,
								PeriodSeconds:       10,
								SuccessThreshold:    1,
								FailureThreshold:    3,
							},
						},
					},
					SecurityContext: &corev1.PodSecurityContext{
						FSGroup:   getInt64Ptr(2000),
						RunAsUser: getInt64Ptr(1000),
					},
				},
			},
		},
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cell.GetCluster().GetNamespace(),
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Type:     corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{
					Name: "web",
					Port: 15000,
				},
				{
					Name: "grpc",
					Port: 15999,
				},
			},
		},
	}

	return deployment, service, nil
}
