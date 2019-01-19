package v1alpha2

import (
	// "fmt"
	"strconv"
	"strings"
)

// GetTabletContainers satisfies ConfigProvider
func (vt *VitessTablet) GetTabletContainers() *TabletContainers {
	return vt.Spec.Containers
}

// func (p *VitessTabletParentSet) IsValid() (bool, error) {
// 	valid := true
// 	problems := []string{}

// 	if p.Cluster == nil {
// 		problems = append(problems, "Parent Cluster not set")
// 		valid = false
// 	}
// 	if p.Cell == nil {
// 		problems = append(problems, "Parent Cell not set")
// 		valid = false
// 	}
// 	if p.Keyspace == nil {
// 		problems = append(problems, "Parent Keyspace not set")
// 		valid = false
// 	}
// 	if p.Shard == nil {
// 		problems = append(problems, "Parent Shard not set")
// 		valid = false
// 	}

// 	return valid, fmt.Errorf(strings.Join(problems, ", "))
// }

func (vt *VitessTablet) SetParents(shard *VitessShard, cell *VitessCell) {
	vt.Spec.parent.Shard = shard
	vt.Spec.parent.VitessShardParents = shard.Spec.parent

	vt.Spec.parent.Cell = cell
}

func (vt *VitessTablet) GetLockserver() *VitessLockserver {
	if vt.GetCell() != nil && vt.GetCell().Spec.Lockserver != nil {
		return vt.GetCell().GetLockserver()
	}

	if vt.GetCluster() != nil && vt.GetCluster().Spec.Lockserver != nil {
		return vt.GetCluster().GetLockserver()
	}

	return nil
}

func (vt *VitessTablet) GetCluster() *VitessCluster {
	return vt.Spec.parent.Cluster
}

func (vt *VitessTablet) GetCell() *VitessCell {
	return vt.Spec.parent.Cell
}

func (vt *VitessTablet) GetKeyspace() *VitessKeyspace {
	return vt.Spec.parent.Keyspace
}

func (vt *VitessTablet) GetShard() *VitessShard {
	return vt.Spec.parent.Shard
}

func (vt *VitessTablet) GetFullName() string {
	return strings.Join([]string{
		vt.GetScopedName(),
	}, "-")
}

func (vt *VitessTablet) GetScopedName() string {
	return strings.Join([]string{
		vt.GetCluster().GetName(),
		vt.GetCell().GetName(),
		vt.GetKeyspace().GetName(),
		vt.GetShard().GetName(),
	}, "-")
}

func (vt *VitessTablet) GetReplicas() *int32 {
	if vt.Spec.Replicas != nil {
		return vt.Spec.Replicas
	}

	if vt.GetShard().Spec.Defaults != nil && vt.GetShard().Spec.Defaults.Replicas != nil {
		return vt.GetShard().Spec.Defaults.Replicas
	}

	var def int32
	return &def
}

func (vt *VitessTablet) GetDBNameAndConfig() (string, *VTContainer) {
	// Inheritance order, with most specific first
	providers := []ConfigProvider{
		vt,
		vt.Spec.parent.Shard,
		vt.Spec.parent.Keyspace,
	}

	for _, p := range providers {
		// TODO: More DB providers
		if containers := p.GetTabletContainers(); containers != nil && containers.MySQL != nil {
			return "mysql", containers.MySQL
		}
	}
	return "", nil
}

func (vt *VitessTablet) GetTabletConfig() *VTContainer {
	// Inheritance order, with most specific first
	providers := []ConfigProvider{
		vt,
		vt.GetShard(),
		vt.GetKeyspace(),
	}

	for _, p := range providers {
		// TODO: More DB providers
		if containers := p.GetTabletContainers(); containers != nil && containers.VTTablet != nil {
			return containers.VTTablet
		}
	}
	return nil
}

func (vt *VitessTablet) GetTabletID() string {
	return strconv.FormatInt(vt.Spec.TabletID, 10)
}
