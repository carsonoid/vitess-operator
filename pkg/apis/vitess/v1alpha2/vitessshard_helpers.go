package v1alpha2

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetEmbeddedTablets processes the embedded tablet map and returns a slice of tablets
func (shard *VitessShard) GetEmbeddedTablets() (ret []*VitessTablet) {
	for name, spec := range shard.Spec.Tablets {
		ret = append(ret, &VitessTablet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: shard.GetNamespace(),
			},
			Spec: *spec,
		})
	}
	return
}

func (shard *VitessShard) SetParent(keyspace *VitessKeyspace) {
	shard.Spec.parent.Keyspace = keyspace
	shard.Spec.parent.VitessKeyspaceParents = keyspace.Spec.parent
}

func (shard *VitessShard) GetKeyspace() *VitessKeyspace {
	return shard.Spec.parent.Keyspace
}

func (shard *VitessShard) GetCluster() *VitessCluster {
	return shard.Spec.parent.Cluster
}

func (shard *VitessShard) EmbedTablet(tablet *VitessTablet) error {
	// First check the the tablet map is initialized
	if shard.Spec.Tablets == nil {
		shard.Spec.Tablets = make(map[string]*VitessTabletSpec)
	}

	// Then check to make sure it is not already defined in the
	// embedded resources
	if _, exists := shard.Spec.Tablets[tablet.GetName()]; exists {
		return fmt.Errorf("Error merging VitessTablet %s: Already defined in the VitessShard", tablet.GetName())
	}

	// TODO make sure tabletIds don't overlap

	shard.Spec.Tablets[tablet.GetName()] = &tablet.DeepCopy().Spec

	return nil
}

// GetTabletContainers satisfies ConfigProvider
func (shard *VitessShard) GetTabletContainers() *TabletContainers {
	if shard.Spec.Defaults != nil {
		return shard.Spec.Defaults.Containers
	}
	return nil
}
