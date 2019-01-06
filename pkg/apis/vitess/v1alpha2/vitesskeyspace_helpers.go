package v1alpha2

import (
	"fmt"
)

// GetTabletContainers satisfies ConfigProvider
func (vk *VitessKeyspace) GetTabletContainers() *TabletContainers {
	if vk.Spec.Defaults != nil {
		return vk.Spec.Defaults.Containers
	}
	return nil
}

func (vk *VitessKeyspace) AddShard(shard *VitessShard) error {
	return vk.Spec.AddShard(shard)
}

func (vks *VitessKeyspaceSpec) AddShard(shard *VitessShard) error {
	// First check the the shard map is initialized
	if vks.Shards == nil {
		vks.Shards = make(map[string]*VitessShardSpec)
	}

	// Then check to make sure it is not already defined in the
	// embedded resources
	if _, exists := vks.Shards[shard.GetName()]; exists {
		return fmt.Errorf("Error merging VitessShard %s: Already defined in the VitessKeyspace", shard.GetName())
	}
	vks.Shards[shard.GetName()] = &shard.DeepCopy().Spec

	return nil
}
