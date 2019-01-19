package v1alpha2

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetEmbeddedShards processes the embedded shard map and returns a slice of shards
func (keyspace *VitessKeyspace) GetEmbeddedShards() (ret []*VitessShard) {
	for name, spec := range keyspace.Spec.Shards {
		ret = append(ret, &VitessShard{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: keyspace.GetNamespace(),
			},
			Spec: *spec,
		})
	}
	return
}

func (keyspace *VitessKeyspace) SetParent(cluster *VitessCluster) {
	keyspace.Spec.parent.Cluster = cluster
}

func (keyspace *VitessKeyspace) GetCluster() *VitessCluster {
	return keyspace.Spec.parent.Cluster
}

// GetTabletContainers satisfies ConfigProvider
func (keyspace *VitessKeyspace) GetTabletContainers() *TabletContainers {
	if keyspace.Spec.Defaults != nil {
		return keyspace.Spec.Defaults.Containers
	}
	return nil
}

func (keyspace *VitessKeyspace) EmbedShard(shard *VitessShard) error {
	// First check the the shard map is initialized
	if keyspace.Spec.Shards == nil {
		keyspace.Spec.Shards = make(map[string]*VitessShardSpec)
	}

	// Then check to make sure it is not already defined in the
	// embedded resources
	if _, exists := keyspace.Spec.Shards[shard.GetName()]; exists {
		return fmt.Errorf("Error merging VitessShard %s: Already defined in the VitessKeyspace", shard.GetName())
	}
	keyspace.Spec.Shards[shard.GetName()] = &shard.DeepCopy().Spec

	return nil
}
