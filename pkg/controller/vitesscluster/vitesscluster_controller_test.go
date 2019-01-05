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
	// logf.SetLogger(logf.ZapLogger(true))

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
			Lockserver: &vitessv1alpha2.VitessLockserverSpec{},
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

// TestVitessClusterCellSelector makes sure that the cellSelelector field works during VitessCluster reconcile
func TestVitessClusterCellSelector(t *testing.T) {
	// Set the logger to development mode for verbose logs.
	logf.SetLogger(logf.ZapLogger(true))

	var (
		namespace = "vitess"
		vcName    = "vitess-operator"
	)
	// Define a minimal cluster which matches one of the cells above
	vc := &vitessv1alpha2.VitessCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vcName,
			Namespace: namespace,
		},
		Spec: vitessv1alpha2.VitessClusterSpec{
			CellSelector: []vitessv1alpha2.ResourceSelector{
				{
					Key:      "app",
					Operator: vitessv1alpha2.ResourceSelectorOpIn,
					Values:   []string{"yes"},
				},
			},
		},
	}

	// Populate the client with initial data
	objs := []runtime.Object{
		&vitessv1alpha2.VitessCell{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "yes",
				Namespace: namespace,
				Labels: map[string]string{
					"app": "yes",
				},
			},
		},
		&vitessv1alpha2.VitessCell{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "no",
				Namespace: namespace,
				Labels: map[string]string{
					"app": "not",
				},
			},
		},
		vc,
	}

	// Register operator types with the runtime scheme.
	s := scheme.Scheme
	s.AddKnownTypes(vitessv1alpha2.SchemeGroupVersion, &vitessv1alpha2.VitessCluster{})
	s.AddKnownTypes(vitessv1alpha2.SchemeGroupVersion, &vitessv1alpha2.VitessCell{})
	s.AddKnownTypes(vitessv1alpha2.SchemeGroupVersion, &vitessv1alpha2.VitessCellList{})

	// Create a fake client to mock API calls.
	cl := fake.NewFakeClient(objs...)

	// Create a ReconcileVitessCluster object with the scheme and fake client.
	r := &ReconcileVitessCluster{client: cl, scheme: s}

	// Get cells based on the selector
	cells, err := r.GetCellsFromClusterSelector(vc)

	// Check the result of reconciliation to make sure it has the desired state.
	if err != nil {
		t.Fatalf("Error fetching cells from selector: %s", err)
	}

	if err == nil && cells == nil {
		t.Fatal("No error when getting cells but cell object is nil")
	}

	// The fake client doesn't support any list options right now, it just ignores them
	// Skip this test until it supports the LabelSelector option at a minimum
	// if len(cells.Items) != 1 {
	// 	t.Errorf("VitessCluster CellSelector %#v did not match the correct number of cells; got %d, wanted %d", vc.Spec.CellSelector, len(cells.Items), 1)
	// }
}
