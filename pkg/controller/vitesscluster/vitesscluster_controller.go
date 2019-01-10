package vitesscluster

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	// "k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
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
	vc := &vitessv1alpha2.VitessCluster{}
	err := r.client.Get(context.TODO(), request.NamespacedName, vc)
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

	// Normalize handles all nested resources and selectors
	if err := r.NormalizeVitessCluster(vc); err != nil {
		return reconcile.Result{}, err
	}

	// Reconciliations

	// Reconcile Lockserver
	if result, err := r.ReconcileClusterLockserver(request, vc); err != nil {
		reqLogger.Info("Error Reconciling Lockserver")
		return result, err
	} else if result.Requeue {
		reqLogger.Info("Requeue after reconciling Lockserver")
		return result, nil
	}

	// Reconcile Tablets (StatefulSets)
	if result, err := r.ReconcileClusterTablets(r.client, request, vc); err != nil {
		reqLogger.Info("Error Reconciling Keyspaces")
		return result, err
	} else if result.Requeue {
		reqLogger.Info("Requeue after reconciling Keyspaces")
		return result, nil
	}

	// Nothing to do - don't reqeue
	reqLogger.Info("Skip reconcile: all managed services in sync")
	return reconcile.Result{}, nil
}

func (r *ReconcileVitessCluster) NormalizeVitessCluster(vc *vitessv1alpha2.VitessCluster) error {
	reqLogger := log.WithValues()

	// Handle Lockserver

	// Sanity check: If both Lockserver and LockserverRef are defined, error without requeue
	if vc.Spec.Lockserver != nil && vc.Spec.LockserverRef != nil {
		return fmt.Errorf("Cannot specify both a lockserver and lockserverRef")
	}

	// Populate the embedded lockserver spec from Ref if given
	if vc.Spec.LockserverRef != nil {
		ls := &vitessv1alpha2.VitessLockserver{}
		err := r.client.Get(context.TODO(), types.NamespacedName{Name: vc.Spec.LockserverRef.Name, Namespace: vc.GetNamespace()}, ls)
		if err != nil {
			if errors.IsNotFound(err) {
				// If the referenced Lockserver is not found, error out witout requeue, requeue will happen the next time a lockserver is added/updated
				return fmt.Errorf("Lockserver referenced by lockserverRef %s not found", vc.Spec.LockserverRef.Name)
			}
			return err
		}

		// Since Lockserver and Lockserver Ref are mutually-exclusive, it should be safe
		// to simply populate the Lockserver struct member with a pointer to the fetched lockserver
		vc.Spec.Lockserver = &ls.Spec
	}

	// Handle VitessCells

	// Get VitessCells from selector
	if len(vc.Spec.CellSelector) != 0 {
		cellList := &vitessv1alpha2.VitessCellList{}
		if err := r.ListFromSelectors(context.TODO(), vc.Spec.KeyspaceSelector, cellList); err != nil {
			return fmt.Errorf("Error getting cells for cluster %s", err)
		}

		reqLogger.Info(fmt.Sprintf("VitessCluster's cellSelector matched %d cells", len(cellList.Items)))
		for _, cell := range cellList.Items {
			if err := vc.AddCell(&cell); err != nil {
				return fmt.Errorf("Error adding matched cell to cluster %s", err)
			}
		}
	}

	// At least one cell must be defined / selected
	if len(vc.Spec.Cells) == 0 {
		return fmt.Errorf("No cells defined or selected")
	}

	// Handle Keyspaces

	// Get VitessKeyspaces from selector
	if len(vc.Spec.KeyspaceSelector) != 0 {
		keyspaceList := &vitessv1alpha2.VitessKeyspaceList{}
		if err := r.ListFromSelectors(context.TODO(), vc.Spec.KeyspaceSelector, keyspaceList); err != nil {
			return fmt.Errorf("Error getting keyspaces for cluster %s", err)
		}

		reqLogger.Info(fmt.Sprintf("VitessCluster's keyspaceSelector matched %d keyspaces", len(keyspaceList.Items)))
		for _, keyspace := range keyspaceList.Items {
			if err := vc.AddKeyspace(&keyspace); err != nil {
				return fmt.Errorf("Error adding matched keyspace to cluster %s", err)
			}
		}
	}

	// At least one keyspace must be defined / selected
	if len(vc.Spec.Keyspaces) == 0 {
		return fmt.Errorf("No keyspaces defined or selected")
	}

	// Handle VitessShards from ShardSelector for each keyspace
	totalShards := 0
	for keyspaceName, keyspaceSpec := range vc.Spec.Keyspaces {
		kss := vc.Spec.Keyspaces[keyspaceName]
		shardList := &vitessv1alpha2.VitessShardList{}
		err := r.ListFromSelectors(context.TODO(), keyspaceSpec.ShardSelector, shardList)
		if err != nil {
			return fmt.Errorf("Error getting shards for keyspace %s", err)
		}

		reqLogger.Info(fmt.Sprintf("VitessKeyspace's shardSelector matched %d shards", len(shardList.Items)))
		for _, shard := range shardList.Items {
			if err := kss.AddShard(&shard); err != nil {
				return fmt.Errorf("Error adding matched shard to keyspace %s", err)
			}
		}

		totalShards = totalShards + len(kss.Shards)
	}

	// At least one keyspace must be defined / selected
	if totalShards == 0 {
		return fmt.Errorf("No shards defined or selected")
	}

	// Handle VitessTablets from TabletSelector for each shard
	totalTablets := 0
	for _, keyspaceSpec := range vc.Spec.Keyspaces {
		for _, shardSpec := range keyspaceSpec.Shards {
			tabletList := &vitessv1alpha2.VitessTabletList{}
			err := r.ListFromSelectors(context.TODO(), shardSpec.TabletSelector, tabletList)
			if err != nil {
				return fmt.Errorf("Error getting tablets for shard %s", err)
			}

			reqLogger.Info(fmt.Sprintf("VitessShard's tabletSelector matched %d tablets", len(tabletList.Items)))
			for _, tablet := range tabletList.Items {
				if err := shardSpec.AddTablet(&tablet); err != nil {
					return fmt.Errorf("Error adding matched tablets to shard %s", err)
				}
			}

			for tabletName, tabletSpec := range shardSpec.Tablets {
				_, cellFound := vc.Spec.Cells[tabletSpec.Cell]
				if !cellFound {
					return fmt.Errorf("Tablet %s assigned to cell %s that does not exist", tabletName, tabletSpec.Cell)
				}

			}

			totalTablets = totalTablets + len(shardSpec.Tablets)
		}
	}

	// At least one tablet must be defined / selected
	if totalTablets == 0 {
		return fmt.Errorf("No tablets defined or selected")
	}

	return nil
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

// GetClusterTablets takes a VitessCluster and returns a list of fully prepared VitessTablets
func (r *ReconcileVitessCluster) GetClusterTablets(vc *vitessv1alpha2.VitessCluster) (tablets []*vitessv1alpha2.VitessTablet) {
	for keyspaceName, keyspaceSpec := range vc.Spec.Keyspaces {
		for shardName, shardSpec := range keyspaceSpec.Shards {
			for tabletName, tabletSpec := range shardSpec.Tablets {
				cellSpec, cellFound := vc.Spec.Cells[tabletSpec.Cell]
				if !cellFound {
					// This shouldn't be possible if the cluster has already been normalized.
					panic(fmt.Sprintf("Tablet %s assigned to cell %s that does not exist.", tabletName, tabletSpec.Cell))
				}

				// Properly setup/validate the tablet
				tabletSpec.SetParentSet(vitessv1alpha2.VitessTabletParentSet{
					Cluster:  vc,
					Cell:     &vitessv1alpha2.VitessCell{ObjectMeta: metav1.ObjectMeta{Name: tabletSpec.Cell}, Spec: *cellSpec},
					Keyspace: &vitessv1alpha2.VitessKeyspace{ObjectMeta: metav1.ObjectMeta{Name: keyspaceName}, Spec: *keyspaceSpec},
					Shard:    &vitessv1alpha2.VitessShard{ObjectMeta: metav1.ObjectMeta{Name: shardName}, Spec: *shardSpec},
				})

				// add the the return slice
				tablets = append(tablets, &vitessv1alpha2.VitessTablet{
					ObjectMeta: metav1.ObjectMeta{
						Name:      tabletName,
						Namespace: vc.GetNamespace(),
					},
					Spec: *tabletSpec,
				})
			}
		}
	}
	return
}

// ReconcileClusterTablets should only be called against a fully-populated and verified VitessCluster object
func (r *ReconcileVitessCluster) ReconcileClusterTablets(client client.Client, request reconcile.Request, vc *vitessv1alpha2.VitessCluster) (reconcile.Result, error) {
	for _, tablet := range r.GetClusterTablets(vc) {
		if result, err := r.ReconcileClusterTablet(client, request, vc, tablet); err != nil {
			return result, err
		}
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
		"app":        vt.GetCluster().GetName(),
		"cell":       vt.GetCell().GetName(),
		"keyspace":   vt.GetKeyspace().GetName(),
		"shard":      vt.GetShard().GetName(),
		"component":  "vttablet",
		"type":       string(vt.Spec.Type),
	}

	vtgateLabels := map[string]string{
		"app":       vt.GetCluster().GetName(),
		"cell":      vt.GetCell().GetName(),
		"component": "vtgate",
	}

	sameClusterTabletLabels := map[string]string{
		"app":       vt.GetCluster().GetName(),
		"component": "vtgate",
	}

	sameShardTabletLabels := map[string]string{
		"app":       vt.GetCluster().GetName(),
		"cell":      vt.GetCell().GetName(),
		"keyspace":  vt.GetKeyspace().GetName(),
		"shard":     vt.GetShard().GetName(),
		"component": "vtgate",
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
					Name:      vt.GetName(),
					Namespace: vt.GetNamespace(),
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

func getInt64Ptr(id int64) *int64 {
	return &id
}

func (r *ReconcileVitessCluster) ListFromSelectors(ctx context.Context, rSels []vitessv1alpha2.ResourceSelector, retList runtime.Object) error {
	labelSelector, err := ResourceSelectorsAsLabelSelector(rSels)
	if err == nil {
		err := r.client.List(ctx, &client.ListOptions{LabelSelector: labelSelector}, retList)
		if err != nil {
			return err
		}
		return nil
	}
	return err
}

// ResourceSelectorsAsLabelSelector converts the []ResourceSelector api type into a struct that implements
// labels.Selector.
func ResourceSelectorsAsLabelSelector(rSels []vitessv1alpha2.ResourceSelector) (labels.Selector, error) {
	if len(rSels) == 0 {
		return labels.Nothing(), nil
	}

	selector := labels.NewSelector()
	for _, expr := range rSels {
		var op selection.Operator
		switch expr.Operator {
		case vitessv1alpha2.ResourceSelectorOpIn:
			op = selection.In
		case vitessv1alpha2.ResourceSelectorOpNotIn:
			op = selection.NotIn
		case vitessv1alpha2.ResourceSelectorOpExists:
			op = selection.Exists
		case vitessv1alpha2.ResourceSelectorOpDoesNotExist:
			op = selection.DoesNotExist
		default:
			return nil, fmt.Errorf("%q is not a valid resource selector operator", expr.Operator)
		}
		r, err := labels.NewRequirement(expr.Key, op, expr.Values)
		if err != nil {
			return nil, err
		}
		selector = selector.Add(*r)
	}
	return selector, nil
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
			Image:           "vitess/mysqlctld:latest",
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
			Image:           tabletConf.Image,
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

	// add log containers
	for _, logtype := range []string{"general", "error", "slow-query"} {
		containers = append(containers, corev1.Container{
			Name:            logtype,
			Image:           "vitess/logtail:helm-1.0.4",
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

	containers = append(containers,
		corev1.Container{
			Name:            "logrotate",
			Image:           tabletConf.Image,
			ImagePullPolicy: corev1.PullIfNotPresent,
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "vtdataroot",
					MountPath: "/vtdataroot",
				},
			},
		},
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
		})
	return
}
