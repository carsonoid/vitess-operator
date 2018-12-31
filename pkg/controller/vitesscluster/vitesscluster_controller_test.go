package vitesscluster

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"

	vitessv1alpha2 "vitess.io/vitess-operator/pkg/apis/vitess/v1alpha2"
)

// TestLockserverLockserverRefMutuallyExclusive makes sure that lockserver and lockserverRef are mutually exclusive
func TestLockserverLockserverRefMutuallyExclusive(t *testing.T) {
	// Set the logger to development mode for verbose logs.
	logf.SetLogger(logf.ZapLogger(true))

	var (
		namespace = "vitess"
		vcName    = "vitess-operator"
	)

	// Define a minimal cluster with both a lockserver and lockserverRef given
	vc := &vitessv1alpha2.VitessCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vcName,
			Namespace: namespace,
		},
		Spec: vitessv1alpha2.VitessClusterSpec{
			Lockserver: &vitessv1alpha2.VitessLockserver{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "new",
					Namespace: namespace,
				},
			},
			LockserverRef: &corev1.LocalObjectReference{
				Name: "exists",
			},
		},
	}

	// Objects to track in the fake client.
	objs := []runtime.Object{
		vc,
	}

	// Register operator types with the runtime scheme.
	s := scheme.Scheme
	s.AddKnownTypes(vitessv1alpha2.SchemeGroupVersion, vc)
	// Create a fake client to mock API calls.
	cl := fake.NewFakeClient(objs...)
	// Create a ReconcileVitessCluster object with the scheme and fake client.
	r := &ReconcileVitessCluster{client: cl, scheme: s}

	// Mock request to simulate Reconcile() being called on an event for a
	// watched resource .
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      vcName,
			Namespace: namespace,
		},
	}
	res, err := r.Reconcile(req)
	if err == nil {
		t.Error("Sanity check failure not caught")
	}

	// Check the result of reconciliation to make sure it has the desired state.
	if res.Requeue {
		t.Error("reconcile requeued request and should not have")
	}
}
