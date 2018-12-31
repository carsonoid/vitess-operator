package vitesstablet

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
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	// "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
	vitessv1alpha2 "vitess.io/vitess-operator/pkg/apis/vitess/v1alpha2"
)

var log = logf.Log.WithName("controller_vitesstablet")

// Add creates a new VitessTablet Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileVitessTablet{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("vitesstablet-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource VitessTablet
	err = c.Watch(&source.Kind{Type: &vitessv1alpha2.VitessTablet{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileVitessTablet{}

// ReconcileVitessTablet reconciles a VitessTablet object
type ReconcileVitessTablet struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a VitessTablet object and makes changes based on the state read
// and what is in the VitessTablet.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileVitessTablet) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling VitessTablet")

	// Fetch the VitessTablet instance
	instance := &vitessv1alpha2.VitessTablet{}
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

	rr, err := ReconcileObject(r.client, request, instance, reqLogger)

	if err := r.client.Status().Update(context.TODO(), instance); err != nil {
		reqLogger.Error(err, "Failed to update VitessTablet status.")
		return reconcile.Result{Requeue: true}, err
	}

	return rr, err
}

func ReconcileObject(client client.Client, request reconcile.Request, instance *vitessv1alpha2.VitessTablet, upstreamLog logr.Logger) (reconcile.Result, error) {
	reqLogger := upstreamLog.WithValues()
	reqLogger.Info("Reconciling VitessTablet")

	found := &appsv1.StatefulSet{}
	err := client.Get(context.TODO(), types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		ss := getStatefulSetForTablet(request, instance, reqLogger)
		reqLogger.Info(fmt.Sprintf("StatefulSet: %#v", ss))
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

	instance.Status.State = "Ready"
	return reconcile.Result{}, nil
}

func getStatefulSetForTablet(request reconcile.Request, instance *vitessv1alpha2.VitessTablet, upstreamLog logr.Logger) *appsv1.StatefulSet {
	reqLogger := upstreamLog.WithValues()
	reqLogger.Info("Reconciling VitessTablet")

	dbContainer := corev1.Container{
		Name:  "mysql",
		Image: "image",
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
			// TODO lifecycle hooks
			// TODO args
		},
	}

	// Each VitessTablet should result in a StatefulSet that is maintained in the cluster. Create/Reconcile them here

	labels := map[string]string{
		"tabletname": instance.GetName(),
	}

	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.GetName(), // TODO gen name from: cellName + keyspaceName + keyrangeString + type
			Namespace: instance.GetNamespace(),
			Labels:    labels,
		},
		Spec: appsv1.StatefulSetSpec{
			//PodManagementPolicy: appsv1.PodManagementPolicyParallel{},
			PodManagementPolicy: appsv1.ParallelPodManagement,
			Replicas:            &instance.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			ServiceName: "servicename",
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      instance.GetName(),
					Namespace: instance.GetNamespace(),
					Labels:    labels,
				},
				Spec: corev1.PodSpec{
					// TODO affinity
					Containers: []corev1.Container{
						dbContainer,
						// TODO containers
						// TODO probes
						// TODO ports
						// TODO volumes
					},
					// TODO initContainers
				},
			},
		},
	}
}
