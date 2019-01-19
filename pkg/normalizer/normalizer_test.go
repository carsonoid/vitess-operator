package normalizer

import (
	// "context"
	// "strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	// "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	// "sigs.k8s.io/controller-runtime/pkg/reconcile"
	// logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"

	vitessv1alpha2 "vitess.io/vitess-operator/pkg/apis/vitess/v1alpha2"
)

// TestClusterNormalize ensures that selectors and parenting work as expected all the way down
func TestNormalizer(t *testing.T) {
	// Set the logger to development mode for verbose logs.
	// logf.SetLogger(logf.ZapLogger(true))

	var (
		namespace   = "vitess"
		clusterName = "vitess-operator"
	)

	// simple labels for all resources
	labels := map[string]string{
		"app": "yes",
	}

	// simple selector for all resources
	sel := []vitessv1alpha2.ResourceSelector{
		{
			Key:      "app",
			Operator: vitessv1alpha2.ResourceSelectorOpIn,
			Values:   []string{"yes"},
		},
	}

	// Define a minimal cluster which matches one of the cells above
	cluster := &vitessv1alpha2.VitessCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterName,
			Namespace: namespace,
		},
		Spec: vitessv1alpha2.VitessClusterSpec{
			LockserverRef: &corev1.LocalObjectReference{
				Name: "lockserver",
			},
			CellSelector:     sel,
			KeyspaceSelector: sel,
		},
	}

	// Populate the client with initial data
	objs := []runtime.Object{
		&vitessv1alpha2.VitessLockserver{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "lockserver",
				Namespace: namespace,
				Labels:    labels,
			},
		},
		&vitessv1alpha2.VitessCell{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cell",
				Namespace: namespace,
				Labels:    labels,
			},
		},
		&vitessv1alpha2.VitessKeyspace{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "keyspace",
				Namespace: namespace,
				Labels:    labels,
			},
			Spec: vitessv1alpha2.VitessKeyspaceSpec{
				ShardSelector: sel,
			},
		},
		&vitessv1alpha2.VitessShard{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "shard",
				Namespace: namespace,
				Labels:    labels,
			},
			Spec: vitessv1alpha2.VitessShardSpec{
				TabletSelector: sel,
			},
		},
		&vitessv1alpha2.VitessTablet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "tablet",
				Namespace: namespace,
				Labels:    labels,
			},
			Spec: vitessv1alpha2.VitessTabletSpec{
				TabletID: 101,
				Cell:     "cell",
			},
		},
		cluster,
	}

	// Register operator types with the runtime scheme.
	s := scheme.Scheme
	s.AddKnownTypes(vitessv1alpha2.SchemeGroupVersion, &vitessv1alpha2.VitessCluster{})
	s.AddKnownTypes(vitessv1alpha2.SchemeGroupVersion, &vitessv1alpha2.VitessClusterList{})
	s.AddKnownTypes(vitessv1alpha2.SchemeGroupVersion, &vitessv1alpha2.VitessCell{})
	s.AddKnownTypes(vitessv1alpha2.SchemeGroupVersion, &vitessv1alpha2.VitessCellList{})
	s.AddKnownTypes(vitessv1alpha2.SchemeGroupVersion, &vitessv1alpha2.VitessTablet{})
	s.AddKnownTypes(vitessv1alpha2.SchemeGroupVersion, &vitessv1alpha2.VitessTabletList{})
	s.AddKnownTypes(vitessv1alpha2.SchemeGroupVersion, &vitessv1alpha2.VitessShard{})
	s.AddKnownTypes(vitessv1alpha2.SchemeGroupVersion, &vitessv1alpha2.VitessShardList{})
	s.AddKnownTypes(vitessv1alpha2.SchemeGroupVersion, &vitessv1alpha2.VitessKeyspace{})
	s.AddKnownTypes(vitessv1alpha2.SchemeGroupVersion, &vitessv1alpha2.VitessKeyspaceList{})
	s.AddKnownTypes(vitessv1alpha2.SchemeGroupVersion, &vitessv1alpha2.VitessLockserver{})
	s.AddKnownTypes(vitessv1alpha2.SchemeGroupVersion, &vitessv1alpha2.VitessLockserverList{})

	// Create a fake client to mock API calls.
	client := fake.NewFakeClient(objs...)

	n := New(client)

	// Check Sanity
	if err := n.TestClusterSanity(cluster); err != nil {
		t.Fatalf("Cluster Sanity Test failed: %s", err)
	}

	// Call the normalize function for the cluster
	if err := n.NormalizeCluster(cluster); err != nil {
		t.Fatalf("Error normalizing cluster: %s", err)
	}

	// Ensure that all matched objects were embedded properly
	if err := n.ValidateCluster(cluster); err != nil {
		t.Fatalf("Cluster Sanity Test failed: %s", err)
	}
}
