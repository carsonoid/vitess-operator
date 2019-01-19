package vitesscluster

import (
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	// logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"

	vitessv1alpha2 "vitess.io/vitess-operator/pkg/apis/vitess/v1alpha2"
	"vitess.io/vitess-operator/pkg/normalizer"
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

// TestTabletTemplates ensures that tablet templates are generated properly
func TestTabletTemplates(t *testing.T) {
	// Set the logger to development mode for verbose logs.
	// logf.SetLogger(logf.ZapLogger(true))

	var (
		namespace    = "vitess"
		vcName       = "vitess-operator"
		etcd2Address = "etcd2.test.address:12345"
		etcd2Path    = "etcd2/test/path"
	)

	// Define a minimal cluster which matches one of the cells above
	vc := &vitessv1alpha2.VitessCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vcName,
			Namespace: namespace,
		},
		Spec: vitessv1alpha2.VitessClusterSpec{
			Lockserver: &vitessv1alpha2.VitessLockserverSpec{
				Type: vitessv1alpha2.LockserverTypeEtcd2,
				Etcd2: &vitessv1alpha2.Etcd2Lockserver{
					Address: etcd2Address,
					Path:    etcd2Path,
				},
			},
			Cells: map[string]*vitessv1alpha2.VitessCellSpec{
				"default": {},
			},
			Keyspaces: map[string]*vitessv1alpha2.VitessKeyspaceSpec{
				"default": {
					Shards: map[string]*vitessv1alpha2.VitessShardSpec{
						"default": {
							Defaults: &vitessv1alpha2.VitessShardOptions{
								Containers: &vitessv1alpha2.TabletContainers{
									VTTablet: &vitessv1alpha2.VTContainer{
										Image: "test",
									},
									MySQL: &vitessv1alpha2.VTContainer{
										Image: "test",
									},
								},
							},
							Tablets: map[string]*vitessv1alpha2.VitessTabletSpec{
								"default": {
									TabletID: 101,
									Cell:     "default",
								},
							},
						},
					},
				},
			},
		},
	}

	// Populate the client with initial data
	objs := []runtime.Object{
		vc,
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

	// Create a fake client to mock API calls.
	cl := fake.NewFakeClient(objs...)

	// Call the normalize function for the cluster
	err := normalizer.New(cl).NormalizeCluster(vc)
	if err != nil {
		t.Fatalf("Error normalizing cluster: %s", err)
	}

	for _, vt := range vc.GetEmbeddedTablets() {
		vttabletContainers, vttabletInitContainers, err := GetTabletVTTabletContainers(vt)
		if err != nil {
			t.Fatalf("Error generating vttablet container for tablet: %s", err)
		}

		for _, container := range vttabletContainers {
			// make sure that the etcdpath and etcdaddress end up in the generated scripts for the vttablet container
			if container.Name == "vttablet" {
				if !strings.Contains(container.Args[len(container.Args)-1], etcd2Address) {
					t.Fatalf("Generated start script for vttablet container does not contain the etcd address: %s", container.Args[len(container.Args)-1])
				}

				// make sure that the etcdpath and etcdaddress end up in the generated scripts for the vttablet container
				if !strings.Contains(container.Args[len(container.Args)-1], etcd2Path) {
					t.Fatalf("Generated start script for vttablet container does not contain the etcd path")
				}
			}
		}

		for _, container := range vttabletInitContainers {
			// make sure that the etcdpath and etcdaddress end up in the generated scripts for the vttablet container
			if container.Name == "init-vttablet" {
				if !strings.Contains(container.Args[len(container.Args)-1], etcd2Address) {
					t.Fatalf("Generated start script for init-vttablet container does not contain the etcd address")
				}

				// make sure that the etcdpath and etcdaddress end up in the generated scripts for the vttablet container
				if !strings.Contains(container.Args[len(container.Args)-1], etcd2Path) {
					t.Fatalf("Generated start script for init-vttablet container does not contain the etcd path")
				}
			}
		}
	}
}
