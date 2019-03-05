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
	lockserver_controller "vitess.io/vitess-operator/pkg/controller/vitesslockserver"
)

// ReconcileClusterResources should only be called against a fully-populated and verified VitessCluster object
func (r *ReconcileVitessCluster) ReconcileClusterResources(cluster *vitessv1alpha2.VitessCluster) (reconcile.Result, error) {
	if r, err := r.ReconcileClusterLockserver(cluster); err != nil || r.Requeue {
		return r, err
	}

	if r, err := r.ReconcileClusterTabletService(cluster); err != nil || r.Requeue {
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

	if r, err := r.ReconcileClusterOrchestratorResources(cluster); err != nil || r.Requeue {
		return r, err
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

func (r *ReconcileVitessCluster) ReconcileClusterTabletService(cluster *vitessv1alpha2.VitessCluster) (reconcile.Result, error) {
	service, serviceErr := getServiceForClusterTablets(cluster)
	if serviceErr != nil {
		log.Error(serviceErr, "failed to generate service for VitessCluster tablets", "VitessCluster.Namespace", cluster.GetNamespace(), "VitessCluster.Name", cluster.GetNamespace())
		return reconcile.Result{}, serviceErr
	}
	foundService := &corev1.Service{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: service.GetName(), Namespace: service.GetNamespace()}, foundService)
	if err != nil && errors.IsNotFound(err) {
		controllerutil.SetControllerReference(cluster, service, r.scheme)
		err = r.client.Create(context.TODO(), service)
		if err != nil {
			return reconcile.Result{}, err
		}
	} else if err != nil {
		log.Error(err, "failed to get Service")
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

// getServiceForClusterTablets takes a vitess cluster and returns a headless service that will point to all of the cluster's tablets
func getServiceForClusterTablets(cluster *vitessv1alpha2.VitessCluster) (*corev1.Service, error) {
	labels := map[string]string{
		"app":       "vitess",
		"cluster":   cluster.GetName(),
		"component": "vttablet",
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.GetTabletServiceName(),
			Namespace: cluster.GetNamespace(),
			Labels:    labels,
			Annotations: map[string]string{
				"service.alpha.kubernetes.io/tolerate-unready-endpoints": "true",
			},
		},
		Spec: corev1.ServiceSpec{
			ClusterIP:                corev1.ClusterIPNone,
			Selector:                 labels,
			Type:                     corev1.ServiceTypeClusterIP,
			PublishNotReadyAddresses: true,
			Ports: []corev1.ServicePort{
				{
					Name: "web",
					Port: 15002,
				},
				{
					Name: "grpc",
					Port: 16002,
				},
				// TODO: Configure ports below only if if ppm is enabled
				{
					Name: "query-data",
					Port: 42001,
				},
				{
					Name: "mysql-metrics",
					Port: 42002,
				},
			},
		},
	}

	// The error return is always nil right now, but it still returns one just
	// in case there are error states in the future
	return service, nil
}

func (r *ReconcileVitessCluster) ReconcileClusterOrchestratorResources(cluster *vitessv1alpha2.VitessCluster) (reconcile.Result, error) {
	return reconcile.Result{}, nil
}

func GetClusterOrchestratorStatefulSetAndServices(cluster *vitessv1alpha2.VitessCluster) (ss *appsv1.StatefulSet, directSvc *corev1.Service, headlessSvc *corev1.Service, err error) {
	labels := map[string]string{
		"app":       "vitess",
		"cluster":   cluster.GetName(),
		"component": "orchestrator",
	}

	// Build affinity
	affinity := &corev1.Affinity{
		PodAntiAffinity: &corev1.PodAntiAffinity{
			PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
				// Hard preference to avoid running on the same host as another orchestrator
				{
					Weight: 100,
					PodAffinityTerm: corev1.PodAffinityTerm{
						LabelSelector: &metav1.LabelSelector{
							MatchLabels: labels,
						},
						TopologyKey: "kubernetes.io/hostname",
					},
				},
			},
		},
	}

	containers, initContainers, containerGetErr := GetOrchestratorContainers(cluster)
	if err != nil {
		err = containerGetErr
		return
	}

	replicas := int32(3) // TODO get this from the CR
	ss = &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.GetScopedName("orchestrator"),
			Namespace: cluster.GetNamespace(),
			Labels:    labels,
		},
		Spec: appsv1.StatefulSetSpec{
			PodManagementPolicy: appsv1.ParallelPodManagement,
			Replicas:            &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
				Type: appsv1.RollingUpdateStatefulSetStrategyType,
			},
			ServiceName: cluster.GetScopedName("orchestrator-headless"),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Affinity:       affinity,
					Containers:     containers,
					InitContainers: initContainers,
					Volumes: []corev1.Volume{
						{
							Name: "vt",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
					SecurityContext: &corev1.PodSecurityContext{
						FSGroup:   getInt64Ptr(2000),
						RunAsUser: getInt64Ptr(1000),
					},
					TerminationGracePeriodSeconds: getInt64Ptr(60000000),
				},
			},
		},
	}

	directSvc = &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.GetScopedName("orchestrator"),
			Namespace: cluster.GetNamespace(),
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector:                 labels,
			Type:                     corev1.ServiceTypeClusterIP,
			PublishNotReadyAddresses: true,
			Ports: []corev1.ServicePort{
				{
					Name:       "web",
					Port:       80,
					TargetPort: 3000,
				},
			},
		},
	}

	headlessSvc = &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.GetScopedName("orchestrator-headless"),
			Namespace: cluster.GetNamespace(),
			Labels:    labels,
			Annotations: map[string]string{
				"service.alpha.kubernetes.io/tolerate-unready-endpoints": "true",
			},
		},
		Spec: corev1.ServiceSpec{
			ClusterIP:                corev1.ClusterIPNone,
			Selector:                 labels,
			Type:                     corev1.ServiceTypeClusterIP,
			PublishNotReadyAddresses: true,
			Ports: []corev1.ServicePort{
				{
					Name:       "web",
					Port:       80,
					TargetPort: 3000,
				},
			},
		},
	}

	return
}

func GetClusterOrchestratorContainers(cluster *vitessv1alpha2.VitessCluster) (containers []corev1.Container, initContainers []corev1.Container, err error) {
	initContainers = append(initContainers,
		corev1.Container{
			Name:            "init-orchestrator",
			Image:           "vitess/orchestrator:3.0.14", // TODO get this from a CR w/default
			ImagePullPolicy: corev1.PullIfNotPresent,
			Env: []corev1.EnvVar{
				{
					Name: "MY_POD_NAME",
					ValueFrom: &corev1.EnvVarSource{
						FieldRef: &corev1.ObjectFieldSelector{
							FieldPath: "metadata.name",
						},
					},
				},
			},
			Command: []string{"bash"},
			Args: []string{
				"-c",
				`set -ex

# make a copy of the config map file before editing it locally
cp /conftmp/orchestrator.conf.json /conf/orchestrator.conf.json

# set the local config to advertise/bind its own service IP
sed -i -e "s/POD_NAME/$MY_POD_NAME/g" /conf/orchestrator.conf.json
				`,
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "config-map",
					MountPath: "/conftmp",
				},
				{
					Name:      "config-shared",
					MountPath: "/conf",
				},
			},
		})

	containers = append(containers, corev1.Container{
		Name:            "orchestrator",
		Image:           "vitess/orchestrator:3.0.14",
		ImagePullPolicy: corev1.PullIfNotPresent,
		ReadinessProbe: &corev1.Probe{
			Handler: corev1.Handler{
				HTTPGet: &corev1.HTTPGetAction{
					Path:   "/api/raft-health",
					Port:   intstr.FromInt(3000),
					Scheme: corev1.URISchemeHTTP,
				},
			},
			InitialDelaySeconds: 300,
			TimeoutSeconds:      10,
			PeriodSeconds:       10,
			SuccessThreshold:    1,
			FailureThreshold:    3,
		},
		LivenessProbe: &corev1.Probe{
			Handler: corev1.Handler{
				HTTPGet: &corev1.HTTPGetAction{
					Path:   "/api/raft-health",
					Port:   intstr.FromInt(3000),
					Scheme: corev1.URISchemeHTTP,
				},
			},
			InitialDelaySeconds: 300,
			TimeoutSeconds:      10,
			PeriodSeconds:       10,
			SuccessThreshold:    1,
			FailureThreshold:    3,
		},
		Ports: []corev1.ContainerPort{
			{
				ContainerPort: 3000,
				Name:          "web",
				Protocol:      corev1.ProtocolTCP,
			},
			{
				ContainerPort: 10008,
				Name:          "raft",
				Protocol:      corev1.ProtocolTCP,
			},
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "config-map",
				MountPath: "/conftmp",
			},
			{
				Name:      "config-shared",
				MountPath: "/conf",
			},
		},
		Env: []corev1.EnvVar{
			{
				Name:  "VTCTLD_SERVER_PORT",
				Value: "15999",
			},
		},
	})

	// add log containers with a slice of filename + containername slices
	for _, logtype := range [][]string{
		{"recovery", "recovery"},
		{"orchestrator-audit", "orchestrator"},
	} {
		containers = append(containers, corev1.Container{
			Name:            logtype[1] + "-log",
			Image:           "vitess/logtail:helm-1.0.4", // TODO get this from a CR w/default
			ImagePullPolicy: corev1.PullIfNotPresent,
			Env: []corev1.EnvVar{
				{
					Name:  "TAIL_FILEPATH",
					Value: fmt.Sprintf("/tmp/%s.log", logtype[0]),
				},
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "tmplogs",
					MountPath: "/tmp",
				},
			},
		})
	}

	return
}
