package vitesscluster

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	// "k8s.io/apimachinery/pkg/fields"
	// "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	// "k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	// "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	vitessv1alpha2 "vitess.io/vitess-operator/pkg/apis/vitess/v1alpha2"
	lockserver_controller "vitess.io/vitess-operator/pkg/controller/vitesslockserver"
	"vitess.io/vitess-operator/pkg/normalizer"
	"vitess.io/vitess-operator/pkg/util/scripts"
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

	// Watch for changes to VitessShards and requeue the owner VitessCluster
	err = c.Watch(&source.Kind{Type: &vitessv1alpha2.VitessShard{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &vitessv1alpha2.VitessCluster{},
	})
	if err != nil {
		return err
	}

	// Watch for changes to VitessTablets and requeue the owner VitessCluster
	err = c.Watch(&source.Kind{Type: &vitessv1alpha2.VitessTablet{}}, &handler.EnqueueRequestForOwner{
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
	cluster := &vitessv1alpha2.VitessCluster{}
	err := r.client.Get(context.TODO(), request.NamespacedName, cluster)
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

	// Check

	// Normalize
	n := normalizer.New(r.client)
	if err := n.NormalizeCluster(cluster); err != nil {
		return reconcile.Result{}, err
	}

	// Valdate

	// Reconcile

	// Reconcile Lockserver
	if result, err := r.ReconcileClusterLockserver(request, cluster); err != nil {
		reqLogger.Info("Error Reconciling Lockserver")
		return result, err
	} else if result.Requeue {
		reqLogger.Info("Requeue after reconciling Lockserver")
		return result, nil
	}

	// Reconcile Tablets (StatefulSets)
	if result, err := r.ReconcileClusterResources(r.client, request, cluster); err != nil {
		reqLogger.Info("Error reconciling cluster member resources")
		return result, err
	} else if result.Requeue {
		reqLogger.Info("Requeue after reconciling cluster member resources")
		return result, nil
	}

	// Nothing to do - don't reqeue
	reqLogger.Info("Skip reconcile: all managed services in sync")
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

// ReconcileClusterResources should only be called against a fully-populated and verified VitessCluster object
func (r *ReconcileVitessCluster) ReconcileClusterResources(client client.Client, request reconcile.Request, vc *vitessv1alpha2.VitessCluster) (reconcile.Result, error) {
	reconciledShards := make(map[string]struct{})
	reconciledCells := make(map[string]struct{})
	for _, tablet := range vc.GetEmbeddedTablets() {
		// Create the resources for each tablet
		if result, err := r.ReconcileClusterTablet(client, request, vc, tablet); err != nil {
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

func (r *ReconcileVitessCluster) ReconcileTabletShard(client client.Client, request reconcile.Request, vt *vitessv1alpha2.VitessTablet) (reconcile.Result, error) {
	reqLogger := log.WithValues()

	vc := vt.GetCluster()
	vs := vt.GetShard()

	// Each shard needs a master election job
	job, jobErr := GetInitShardMasterJob(vt, vt.GetShard(), vt.GetCluster())
	if jobErr != nil {
		reqLogger.Error(jobErr, "failed to generate MasterElect Job for VitessShard", "VitessShard.Namespace", vs.GetNamespace(), "VitessShard.Name", vs.GetNamespace())
		return reconcile.Result{}, jobErr
	}

	found := &batchv1.Job{}
	err := client.Get(context.TODO(), types.NamespacedName{Name: job.GetName(), Namespace: job.GetNamespace()}, found)
	if err != nil && errors.IsNotFound(err) {
		controllerutil.SetControllerReference(vc, job, r.scheme)
		err = client.Create(context.TODO(), job)
		if err != nil {
			return reconcile.Result{}, err
		}
		// Job created successfully - return and requeue
		return reconcile.Result{Requeue: true}, nil
	} else if err != nil {
		reqLogger.Error(err, "failed to get Job")
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileVitessCluster) ReconcileClusterTablet(client client.Client, request reconcile.Request, vc *vitessv1alpha2.VitessCluster, vt *vitessv1alpha2.VitessTablet) (reconcile.Result, error) {
	reqLogger := log.WithValues()

	// Each embedded tablet should result in a StatefulSet
	found := &appsv1.StatefulSet{}
	err := client.Get(context.TODO(), types.NamespacedName{Name: vt.GetFullName(), Namespace: vt.GetNamespace()}, found)
	if err != nil && errors.IsNotFound(err) {
		ss, ssErr := getStatefulSetForTablet(vt, reqLogger)
		if ssErr != nil {
			reqLogger.Error(ssErr, "failed to generate StatefulSet for VitessTablet", "VitessTablet.Namespace", vt.GetNamespace(), "VitessTablet.Name", vt.GetNamespace())
			return reconcile.Result{}, ssErr
		}
		controllerutil.SetControllerReference(vc, ss, r.scheme)
		// reqLogger.Info(fmt.Sprintf("%#v", ss.ObjectMeta))
		err = client.Create(context.TODO(), ss)
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

func getStatefulSetForTablet(vt *vitessv1alpha2.VitessTablet, upstreamLog logr.Logger) (*appsv1.StatefulSet, error) {
	reqLogger := upstreamLog.WithValues()
	reqLogger.Info("Reconciling VitessTablet")

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

func getInt32Ptr(id int32) *int32 {
	return &id
}

func getInt64Ptr(id int64) *int64 {
	return &id
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

func GetInitShardMasterJob(vt *vitessv1alpha2.VitessTablet, vs *vitessv1alpha2.VitessShard, vc *vitessv1alpha2.VitessCluster) (*batchv1.Job, error) {
	jobName := vt.GetScopedName() + "-init-shard-master"

	scripts := scripts.NewContainerScriptGenerator("init_shard_master", vt)
	if err := scripts.Generate(); err != nil {
		return nil, err
	}

	jobLabels := map[string]string{
		"app":                "vitess",
		"cluster":            vt.GetCluster().GetName(),
		"cell":               vt.GetCell().GetName(),
		"keyspace":           vt.GetKeyspace().GetName(),
		"shard":              vt.GetShard().GetName(),
		"component":          "vttablet",
		"initShardMasterJob": "true",
		"job-name":           jobName,
	}

	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: vt.GetNamespace(),
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

func (r *ReconcileVitessCluster) ReconcileClusterCellVtctld(request reconcile.Request, vt *vitessv1alpha2.VitessTablet) (reconcile.Result, error) {
	reqLogger := log.WithValues()

	// Each shard needs a master election job
	deploy, service, deployErr := GetClusterCellVtctld(vt.GetCluster(), vt.GetCell(), vt)
	if deployErr != nil {
		reqLogger.Error(deployErr, "failed to generate Vtctld Deployment for VitessCell", "VitessCell.Namespace", vt.GetCell().GetNamespace(), "VitessCell.Name", vt.GetCell().GetNamespace())
		return reconcile.Result{}, deployErr
	}

	found := &appsv1.Deployment{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: deploy.GetName(), Namespace: deploy.GetNamespace()}, found)
	if err != nil && errors.IsNotFound(err) {
		controllerutil.SetControllerReference(vt.GetCluster(), deploy, r.scheme)
		err = r.client.Create(context.TODO(), deploy)
		if err != nil {
			return reconcile.Result{}, err
		}
		// Job created successfully - return and requeue
		return reconcile.Result{Requeue: true}, nil
	} else if err != nil {
		reqLogger.Error(err, "failed to get Deployment")
		return reconcile.Result{}, err
	}

	foundSvc := &corev1.Service{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: service.GetName(), Namespace: service.GetNamespace()}, foundSvc)
	if err != nil && errors.IsNotFound(err) {
		controllerutil.SetControllerReference(vt.GetCluster(), service, r.scheme)
		err = r.client.Create(context.TODO(), service)
		if err != nil {
			return reconcile.Result{}, err
		}
		// Job created successfully - return and requeue
		return reconcile.Result{Requeue: true}, nil
	} else if err != nil {
		reqLogger.Error(err, "failed to get Service")
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func GetClusterCellVtctld(vc *vitessv1alpha2.VitessCluster, vcell *vitessv1alpha2.VitessCell, vt *vitessv1alpha2.VitessTablet) (*appsv1.Deployment, *corev1.Service, error) {
	name := vc.GetName() + "-" + vcell.GetName() + "-vtctld"

	scripts := scripts.NewContainerScriptGenerator("vtctld", vt)
	if err := scripts.Generate(); err != nil {
		return nil, nil, err
	}

	labels := map[string]string{
		"app":       "vitess",
		"cluster":   vt.GetCluster().GetName(),
		"component": "vtctld",
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: vc.GetNamespace(),
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
			Namespace: vc.GetNamespace(),
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
