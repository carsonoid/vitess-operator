package vitesscluster

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	vitessv1alpha2 "vitess.io/vitess-operator/pkg/apis/vitess/v1alpha2"
	"vitess.io/vitess-operator/pkg/util/scripts"
)

func (r *ReconcileVitessCluster) ReconcileClusterTablet(request reconcile.Request, vc *vitessv1alpha2.VitessCluster, vt *vitessv1alpha2.VitessTablet) (reconcile.Result, error) {
	reqLogger := log.WithValues()

	// Each embedded tablet should result in a StatefulSet
	found := &appsv1.StatefulSet{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: vt.GetFullName(), Namespace: vt.GetNamespace()}, found)
	if err != nil && errors.IsNotFound(err) {
		ss, ssErr := getStatefulSetForTablet(vt)
		if ssErr != nil {
			reqLogger.Error(ssErr, "failed to generate StatefulSet for VitessTablet", "VitessTablet.Namespace", vt.GetNamespace(), "VitessTablet.Name", vt.GetNamespace())
			return reconcile.Result{}, ssErr
		}
		controllerutil.SetControllerReference(vc, ss, r.scheme)
		// reqLogger.Info(fmt.Sprintf("%#v", ss.ObjectMeta))
		err = r.client.Create(context.TODO(), ss)
		if err != nil {
			return reconcile.Result{}, err
		}
		// Deployment created successfully - return and requeue
		return reconcile.Result{Requeue: true}, nil
	} else if err != nil {
		reqLogger.Error(err, "failed to get StatefulSet")
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func getStatefulSetForTablet(vt *vitessv1alpha2.VitessTablet) (*appsv1.StatefulSet, error) {
	selfLabels := map[string]string{
		"tabletname": vt.GetName(),
		"app":        "vitess",
		"cluster":    vt.GetCluster().GetName(),
		"cell":       vt.GetCell().GetName(),
		"keyspace":   vt.GetKeyspace().GetName(),
		"shard":      vt.GetShard().GetName(),
		"component":  "vttablet",
		"type":       string(vt.Spec.Type),
	}

	vtgateLabels := map[string]string{
		"app":       "vitess",
		"cluster":   vt.GetCluster().GetName(),
		"cell":      vt.GetCell().GetName(),
		"component": "vtgate",
	}

	sameClusterTabletLabels := map[string]string{
		"app":       "vitess",
		"cluster":   vt.GetCluster().GetName(),
		"component": "vttablet",
	}

	sameShardTabletLabels := map[string]string{
		"app":       "vitess",
		"cluster":   vt.GetCluster().GetName(),
		"cell":      vt.GetCell().GetName(),
		"keyspace":  vt.GetKeyspace().GetName(),
		"shard":     vt.GetShard().GetName(),
		"component": "vttablet",
	}

	// Build affinity
	affinity := &corev1.Affinity{
		PodAffinity: &corev1.PodAffinity{
			// Prefer to run on the same host as a vtgate pod
			PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
				{
					Weight: 10,
					PodAffinityTerm: corev1.PodAffinityTerm{
						LabelSelector: &metav1.LabelSelector{
							MatchLabels: vtgateLabels,
						},
						TopologyKey: "kubernetes.io/hostname",
					},
				},
			},
		},
		PodAntiAffinity: &corev1.PodAntiAffinity{
			PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
				// Hard preference to avoid running on the same host as another tablet in the same shard/keyspace
				{
					Weight: 100,
					PodAffinityTerm: corev1.PodAffinityTerm{
						LabelSelector: &metav1.LabelSelector{
							MatchLabels: sameShardTabletLabels,
						},
						TopologyKey: "kubernetes.io/hostname",
					},
				},
				// Soft preference to avoid running on the same host as another tablet in the same cluster
				{
					Weight: 10,
					PodAffinityTerm: corev1.PodAffinityTerm{
						LabelSelector: &metav1.LabelSelector{
							MatchLabels: sameClusterTabletLabels,
						},
						TopologyKey: "kubernetes.io/hostname",
					},
				},
			},
		},
	}

	dbContainers, dbInitContainers, err := GetTabletMysqlContainers(vt)
	if err != nil {
		return nil, err
	}

	vttabletContainers, vttabletInitContainers, err := GetTabletVTTabletContainers(vt)
	if err != nil {
		return nil, err
	}

	// build containers
	containers := []corev1.Container{}
	containers = append(containers, dbContainers...)
	containers = append(containers, vttabletContainers...)

	// build initcontainers
	initContainers := []corev1.Container{}
	initContainers = append(initContainers, dbInitContainers...)
	initContainers = append(initContainers, vttabletInitContainers...)

	// setup volume requests
	volumeRequests := make(corev1.ResourceList)
	volumeRequests[corev1.ResourceStorage] = resource.MustParse("10Gi")

	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vt.GetFullName(),
			Namespace: vt.GetNamespace(),
			Labels:    selfLabels,
		},
		Spec: appsv1.StatefulSetSpec{
			//PodManagementPolicy: appsv1.PodManagementPolicyParallel{},
			PodManagementPolicy: appsv1.ParallelPodManagement,
			Replicas:            vt.GetReplicas(),
			Selector: &metav1.LabelSelector{
				MatchLabels: selfLabels,
			},
			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
				Type: appsv1.RollingUpdateStatefulSetStrategyType,
			},
			ServiceName: "vttablet",
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: selfLabels,
				},
				Spec: corev1.PodSpec{
					Affinity:       affinity,
					Containers:     containers,
					InitContainers: initContainers,
					//   - emptyDir: {}
					// name: vt
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
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "vtdataroot",
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						Resources: corev1.ResourceRequirements{
							Requests: volumeRequests,
						},
					},
				},
			},
		},
	}, nil
}

func GetTabletMysqlContainers(vt *vitessv1alpha2.VitessTablet) (containers []corev1.Container, initContainers []corev1.Container, err error) {
	dbName, dbConf := vt.GetDBNameAndConfig()
	if dbConf == nil {
		return containers, initContainers, fmt.Errorf("No database container configuration found")
	}

	dbScripts := scripts.NewContainerScriptGenerator("mysql", vt)
	if err := dbScripts.Generate(); err != nil {
		return containers, initContainers, fmt.Errorf("Error generating DB container scripts: %s", err)
	}

	initContainers = append(initContainers,
		corev1.Container{
			Name:            "init-mysql",
			Image:           "vitess/mysqlctld:helm-1.0.3", // TODO get this from a crd w/default
			ImagePullPolicy: corev1.PullIfNotPresent,
			Command:         []string{"bash"},
			Args: []string{
				"-c",
				dbScripts.Init,
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "vtdataroot",
					MountPath: "/vtdataroot",
				},
				{
					Name:      "vt",
					MountPath: "/vttmp",
				},
			},
		})

	containers = append(containers, corev1.Container{
		Name:            dbName,
		Image:           dbConf.Image,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command:         []string{"bash"},
		Args: []string{
			"-c",
			dbScripts.Start,
		},
		Lifecycle: &corev1.Lifecycle{
			PreStop: &corev1.Handler{
				Exec: &corev1.ExecAction{
					Command: []string{
						"bash",
						"-c",
						dbScripts.PreStop,
					},
				},
			},
		},
		ReadinessProbe: &corev1.Probe{
			Handler: corev1.Handler{
				Exec: &corev1.ExecAction{
					Command: []string{
						"mysqladmin",
						"ping",
						"-uroot",
						"--socket=/vtdataroot/tabletdata/mysql.sock",
					},
				},
			},
			InitialDelaySeconds: 60,
			TimeoutSeconds:      10,
			PeriodSeconds:       10,
			SuccessThreshold:    1,
			FailureThreshold:    3,
		},
		Resources: dbConf.Resources,
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "vtdataroot",
				MountPath: "/vtdataroot",
			},
			{
				Name:      "vt",
				MountPath: "/vt",
			},
		},
		Env: []corev1.EnvVar{
			{
				Name:  "VTROOT",
				Value: "/vt",
			},
			{
				Name:  "VTDATAROOT",
				Value: "/vtdataroot",
			},
			{
				Name:  "GOBIN",
				Value: "/vt/bin",
			},
			{
				Name:  "VT_MYSQL_ROOT",
				Value: "/usr",
			},
			{
				Name:  "PKG_CONFIG_PATH",
				Value: "/vt/lib",
			},
			{
				Name: "VT_DB_FLAVOR",
				ValueFrom: &corev1.EnvVarSource{
					ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
						Key: "db.flavor",
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "vitess-cm",
						},
					},
				},
			},
		},
	})

	return
}

func GetTabletVTTabletContainers(vt *vitessv1alpha2.VitessTablet) (containers []corev1.Container, initContainers []corev1.Container, err error) {
	tabletConf := vt.GetTabletConfig()
	if tabletConf == nil {
		err = fmt.Errorf("No database container configuration found")
		return
	}

	vtScripts := scripts.NewContainerScriptGenerator("vttablet", vt)
	if err = vtScripts.Generate(); err != nil {
		err = fmt.Errorf("Error generating DB container scripts: %s", err)
		return
	}

	initContainers = append(initContainers,
		corev1.Container{
			Name:            "init-vttablet",
			Image:           "vitess/vtctl:helm-1.0.3", // TODO get this from a crd w/default
			ImagePullPolicy: corev1.PullIfNotPresent,
			Command:         []string{"bash"},
			Args: []string{
				"-c",
				vtScripts.Init,
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "vtdataroot",
					MountPath: "/vtdataroot",
				},
			},
		})

	containers = append(containers,
		corev1.Container{
			Name:            "vttablet",
			Image:           tabletConf.Image,
			ImagePullPolicy: corev1.PullIfNotPresent,
			Command:         []string{"bash"},
			Args: []string{
				"-c",
				vtScripts.Start,
			},
			Lifecycle: &corev1.Lifecycle{
				PreStop: &corev1.Handler{
					Exec: &corev1.ExecAction{
						Command: []string{
							"bash",
							"-c",
							vtScripts.PreStop,
						},
					},
				},
			},
			ReadinessProbe: &corev1.Probe{
				Handler: corev1.Handler{
					HTTPGet: &corev1.HTTPGetAction{
						Path:   "/debug/health",
						Port:   intstr.FromInt(15002),
						Scheme: corev1.URISchemeHTTP,
					},
				},
				InitialDelaySeconds: 60,
				TimeoutSeconds:      10,
				PeriodSeconds:       10,
				SuccessThreshold:    1,
				FailureThreshold:    3,
			},
			LivenessProbe: &corev1.Probe{
				Handler: corev1.Handler{
					HTTPGet: &corev1.HTTPGetAction{
						Path:   "/debug/status",
						Port:   intstr.FromInt(15002),
						Scheme: corev1.URISchemeHTTP,
					},
				},
				InitialDelaySeconds: 60,
				TimeoutSeconds:      10,
				PeriodSeconds:       10,
				SuccessThreshold:    1,
				FailureThreshold:    3,
			},
			Ports: []corev1.ContainerPort{
				{
					ContainerPort: 15002,
					Name:          "web",
					Protocol:      corev1.ProtocolTCP,
				},
				{
					ContainerPort: 16002,
					Name:          "grpc",
					Protocol:      corev1.ProtocolTCP,
				},
			},
			Resources: corev1.ResourceRequirements{
				// Limits:   corev1.ResourceList{},
				// Requests: corev1.ResourceList{},
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "vtdataroot",
					MountPath: "/vtdataroot",
				},
			},
			Env: []corev1.EnvVar{
				{
					Name:  "VTROOT",
					Value: "/vt",
				},
				{
					Name:  "VTDATAROOT",
					Value: "/vtdataroot",
				},
				{
					Name:  "GOBIN",
					Value: "/vt/bin",
				},
				{
					Name:  "VT_MYSQL_ROOT",
					Value: "/usr",
				},
				{
					Name:  "PKG_CONFIG_PATH",
					Value: "/vt/lib",
				},
				{
					Name: "VT_DB_FLAVOR",
					ValueFrom: &corev1.EnvVarSource{
						ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
							Key: "db.flavor",
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "vitess-cm",
							},
						},
					},
				},
			},
		},
		corev1.Container{
			Name:            "logrotate",
			Image:           "vitess/logrotate:helm-1.0.4", // TODO get this from a crd w/default
			ImagePullPolicy: corev1.PullIfNotPresent,
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "vtdataroot",
					MountPath: "/vtdataroot",
				},
			},
		})

	// add log containers with a slice of filename + containername slices
	for _, logtype := range [][]string{
		{"general", "general"},
		{"error", "error"},
		{"slow-query", "slow"},
	} {
		containers = append(containers, corev1.Container{
			Name:            logtype[1] + "-log",
			Image:           "vitess/logtail:helm-1.0.4", // TODO get this from a crd w/default
			ImagePullPolicy: corev1.PullIfNotPresent,
			Env: []corev1.EnvVar{
				{
					Name:  "TAIL_FILEPATH",
					Value: fmt.Sprintf("/vtdataroot/tabletdata/%s.log", logtype[0]),
				},
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "vtdataroot",
					MountPath: "/vtdataroot",
				},
			},
		})
	}

	return
}
