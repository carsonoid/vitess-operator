package vitessshard

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	vitessv1alpha2 "vitess.io/vitess-operator/pkg/apis/vitess/v1alpha2"
	tablet_controller "vitess.io/vitess-operator/pkg/controller/vitesstablet"
)

var log = logf.Log.WithName("controller_vitessshard")

// Add creates a new VitessShard Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileVitessShard{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// NewReconciler returns a new reconcile.Reconciler
func NewReconciler(client client.Client, scheme *runtime.Scheme) *ReconcileVitessShard {
	return &ReconcileVitessShard{client: client, scheme: scheme}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("vitessshard-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource VitessShard
	err = c.Watch(&source.Kind{Type: &vitessv1alpha2.VitessShard{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileVitessShard{}

// ReconcileVitessShard reconciles a VitessShard object
type ReconcileVitessShard struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a VitessShard object and makes changes based on the state read
// and what is in the VitessShard.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileVitessShard) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling VitessShard")

	// Fetch the VitessShard instance
	instance := &vitessv1alpha2.VitessShard{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
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

	rr, err := r.ReconcileObject(r.client, request, instance, reqLogger)

	if err := r.client.Status().Update(context.TODO(), instance); err != nil {
		reqLogger.Error(err, "Failed to update VitessShard status.")
		return reconcile.Result{Requeue: true}, err
	}

	return rr, err
}

// ReconcileObject does all the actual reconcile work
func (r *ReconcileVitessShard) ReconcileObject(client client.Client, request reconcile.Request, instance *vitessv1alpha2.VitessShard, upstreamLog logr.Logger) (reconcile.Result, error) {
	reqLogger := upstreamLog.WithValues()
	reqLogger.Info("Reconciling VitessShard")

	result, err := r.ReconcileClusterTablets(client, request, instance, upstreamLog)
	if err != nil || result.Requeue {
		return result, err
	}

	if err := r.client.Status().Update(context.TODO(), instance); err != nil {
		reqLogger.Error(err, "Failed to update VitessShard")
		return reconcile.Result{}, err
	}

	instance.Status.State = "Ready"
	return reconcile.Result{}, nil
}

func (r *ReconcileVitessShard) ReconcileClusterTablets(client client.Client, request reconcile.Request, vs *vitessv1alpha2.VitessShard, upstreamLog logr.Logger) (reconcile.Result, error) {
	reqLogger := upstreamLog.WithValues()

	// Handle embedded tablets
	for tabletName, tabletSpec := range vs.Spec.Tablets {
		reqLogger.Info(fmt.Sprintf("Reconciling embedded tablet %s", tabletName))

		// make sure status map is initialized
		if vs.Status.Tablets == nil {
			vs.Status.Tablets = make(map[string]*vitessv1alpha2.VitessTabletStatus)
		}

		// Build a complete VitessTablet
		vt := &vitessv1alpha2.VitessTablet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      vs.GetName() + "-" + tabletName,
				Namespace: vs.GetNamespace(),
			},
			Spec: *tabletSpec.DeepCopy(),
		}

		if tabletStatus, ok := vs.Status.Tablets[tabletName]; ok {
			// If status is not empty, deepcopy it into the tmp object
			tabletStatus.DeepCopyInto(&vt.Status)
		}

		// Run it through the controller's reconcile func
		recResult, recErr := tablet_controller.ReconcileObject(client, request, vt, reqLogger)

		// Each embedded tablet should result in a StatefulSet
		found := &appsv1.StatefulSet{}
		err := client.Get(context.TODO(), types.NamespacedName{Name: vs.Name, Namespace: vs.Namespace}, found)
		if err != nil && errors.IsNotFound(err) {
			ss := getStatefulSetForTablet(vs, vt, reqLogger)
			controllerutil.SetControllerReference(vs, ss, r.scheme)
			// reqLogger.Info(fmt.Sprintf("StatefulSet: %#v", ss))
			err = client.Create(context.TODO(), ss)
			if err != nil {
				reqLogger.Error(err, "failed to create new StatefulSet", "StatefulSet.Namespace", ss.Namespace, "StatefulSet.Name", ss.Name)
				return reconcile.Result{}, err
			}
			// Deployment created successfully - return and requeue
			return reconcile.Result{Requeue: true}, nil
		} else if err != nil {
			reqLogger.Error(err, "failed to get StatefulSet")
			return reconcile.Result{}, err
		}

		// Split and store the spec and status in the parent VitessCluster
		vs.Spec.Tablets[tabletName] = *vt.Spec.DeepCopy()
		vs.Status.Tablets[tabletName] = vt.Status.DeepCopy()

		// Reque if needed
		if recErr != nil || recResult.Requeue {
			return recResult, recErr
		}
	}

	// for _, keyspaceSelector := range vc.Spec.KeyspaceSelector {
	// 	// TODO Fetch the Shards from the selector
	// }

	return reconcile.Result{}, nil
}

func getStatefulSetForTablet(vs *vitessv1alpha2.VitessShard, vt *vitessv1alpha2.VitessTablet, upstreamLog logr.Logger) *appsv1.StatefulSet {
	reqLogger := upstreamLog.WithValues()
	reqLogger.Info("Reconciling VitessTablet")

	selfLabels := map[string]string{
		"tabletname": vs.GetName(),
		"app":        "vitess",
		"cell":       "zone1",
		"component":  "vttablet",
		"keyspace":   "sharded-db",
		"shard":      "80-x",
		"type":       "rdonly",
	}

	vtgateLabels := map[string]string{
		"app":       "vitess",
		"cell":      "zone1",
		"component": "vtgate",
	}

	sameClusterTabletLabels := map[string]string{
		"app":       "vitess",
		"component": "vtgate",
	}

	sameShardTabletLabels := map[string]string{
		"app":       "vitess",
		"cell":      "zone1",
		"component": "vtgate",
		"keyspace":  "sharded",
		"shard":     "80-x",
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

	// build containers
	containers := []corev1.Container{
		{
			Name:            "mysql",
			Image:           "percona:5.7.23",
			ImagePullPolicy: corev1.PullIfNotPresent,
			Command:         []string{"bash"},
			Args: []string{
				"-c",
				dbContainerStart,
			},
			Lifecycle: &corev1.Lifecycle{
				PreStop: &corev1.Handler{
					Exec: &corev1.ExecAction{
						Command: []string{
							"bash",
							"-c",
							dbContainerPrestop,
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
			Resources: corev1.ResourceRequirements{
				// Limits:   corev1.ResourceList{},
				// Requests: corev1.ResourceList{},
			},
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
		},
		{
			Name:            "vttablet",
			Image:           "vitess/vttablet:helm-1.0.3",
			ImagePullPolicy: corev1.PullIfNotPresent,
			Command:         []string{"bash"},
			Args: []string{
				"-c",
				tabletContainerStart,
			},
			Lifecycle: &corev1.Lifecycle{
				PreStop: &corev1.Handler{
					Exec: &corev1.ExecAction{
						Command: []string{
							"bash",
							"-c",
							tabletContainerPrestop,
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
		{
			Name:            "logrotate",
			Image:           "vitess/vttablet:helm-1.0.3",
			ImagePullPolicy: corev1.PullIfNotPresent,
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "vtdataroot",
					MountPath: "/vtdataroot",
				},
			},
		},
	}

	// add log containers
	for _, logtype := range []string{"general", "error", "slow-query"} {
		containers = append(containers, corev1.Container{
			Name:            logtype,
			Image:           "vitess/vttablet:helm-1.0.3",
			ImagePullPolicy: corev1.PullIfNotPresent,
			Env: []corev1.EnvVar{
				{
					Name:  "TAIL_FILEPATH",
					Value: fmt.Sprintf("/vtdataroot/tabletdata/%s.log", logtype),
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

	// build initcontainers
	initContainers := []corev1.Container{
		{
			Name:            "init-mysql",
			Image:           "vitess/vttablet:helm-1.0.3",
			ImagePullPolicy: corev1.PullIfNotPresent,
			Command:         []string{"bash"},
			Args: []string{
				"-c",
				initMySQL,
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "vtdataroot",
					MountPath: "/vtdataroot",
				},
			},
		},
		{
			Name:            "init-vttablet",
			Image:           "vitess/vttablet:helm-1.0.3",
			ImagePullPolicy: corev1.PullIfNotPresent,
			Command:         []string{"bash"},
			Args: []string{
				"-c",
				initVTTablet,
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "vtdataroot",
					MountPath: "/vtdataroot",
				},
			},
		},
	}

	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vs.GetName(), // TODO gen name from: cellName + keyspaceName + keyrangeString + type
			Namespace: vs.GetNamespace(),
			Labels:    selfLabels,
		},
		Spec: appsv1.StatefulSetSpec{
			//PodManagementPolicy: appsv1.PodManagementPolicyParallel{},
			PodManagementPolicy: appsv1.ParallelPodManagement,
			Replicas:            vs.GetTabletReplicas(vt, 0),
			Selector: &metav1.LabelSelector{
				MatchLabels: selfLabels,
			},
			ServiceName: "vttablet",
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      vs.GetName(),
					Namespace: vs.GetNamespace(),
					Labels:    selfLabels,
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
		},
	}
}

func getInt64Ptr(id int64) *int64 {
	return &id
}
