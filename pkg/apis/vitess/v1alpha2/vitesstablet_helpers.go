package v1alpha2

import (
	// "fmt"
	"strconv"
	"strings"
)

// GetTabletContainers satisfies ConfigProvider
func (tablet *VitessTablet) GetTabletContainers() *TabletContainers {
	return tablet.Spec.Containers
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

func (tablet *VitessTablet) SetParents(shard *VitessShard, cell *VitessCell) {
	tablet.Spec.parent.Shard = shard
	tablet.Spec.parent.VitessShardParents = shard.Spec.parent

	tablet.Spec.parent.Cell = cell
}

func (tablet *VitessTablet) GetLockserver() *VitessLockserver {
	if tablet.GetCell() != nil && tablet.GetCell().Spec.Lockserver != nil {
		return tablet.GetCell().GetLockserver()
	}

	if tablet.GetCluster() != nil && tablet.GetCluster().Spec.Lockserver != nil {
		return tablet.GetCluster().GetLockserver()
	}

	return nil
}

func (tablet *VitessTablet) GetCluster() *VitessCluster {
	return tablet.Spec.parent.Cluster
}

func (tablet *VitessTablet) GetCell() *VitessCell {
	return tablet.Spec.parent.Cell
}

func (tablet *VitessTablet) GetKeyspace() *VitessKeyspace {
	return tablet.Spec.parent.Keyspace
}

func (tablet *VitessTablet) GetShard() *VitessShard {
	return tablet.Spec.parent.Shard
}

func (tablet *VitessTablet) GetScopedName(extra ...string) string {
	return strings.Join(append(
		[]string{
			tablet.GetCluster().GetName(),
			tablet.GetCell().GetName(),
			tablet.GetKeyspace().GetName(),
			tablet.GetShard().GetName(),
		},
		extra...), "-")
}

func (tablet *VitessTablet) GetReplicas() *int32 {
	if tablet.Spec.Replicas != nil {
		return tablet.Spec.Replicas
	}

	if tablet.GetShard().Spec.Defaults != nil && tablet.GetShard().Spec.Defaults.Replicas != nil {
		return tablet.GetShard().Spec.Defaults.Replicas
	}

	var def int32
	return &def
}

func (tablet *VitessTablet) GetMySQLContainer() *MySQLContainer {
	// Inheritance order, with most specific first
	providers := []ConfigProvider{
		tablet,
		tablet.Spec.parent.Shard,
		tablet.Spec.parent.Keyspace,
	}

	for _, p := range providers {
		if containers := p.GetTabletContainers(); containers != nil && containers.MySQL != nil {
			// TODO get defaults from full range of providers
			if containers.MySQL.DBFlavor == "" && containers.DBFlavor != "" {
				containers.MySQL.DBFlavor = containers.DBFlavor
			}
			return containers.MySQL
		}
	}
	return nil
}

func (tablet *VitessTablet) GetVTTabletContainer() *VTTabletContainer {
	// Inheritance order, with most specific first
	providers := []ConfigProvider{
		tablet,
		tablet.GetShard(),
		tablet.GetKeyspace(),
	}

	for _, p := range providers {
		if containers := p.GetTabletContainers(); containers != nil && containers.VTTablet != nil {
			// TODO get defaults from full range of providers
			if containers.VTTablet.DBFlavor == "" && containers.DBFlavor != "" {
				containers.VTTablet.DBFlavor = containers.DBFlavor
			}
			return containers.VTTablet
		}
	}
	return nil
}

func (tablet *VitessTablet) GetTabletID() string {
	return strconv.FormatInt(tablet.Spec.TabletID, 10)
}
