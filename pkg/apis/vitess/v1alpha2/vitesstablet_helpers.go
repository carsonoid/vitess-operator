package v1alpha2

import (
	"fmt"
	"strings"
)

// GetTabletContainers satisfies ConfigProvider
func (vt *VitessTablet) GetTabletContainers() *TabletContainers {
	return vt.Spec.Containers
}

func (p *VitessTabletParentSet) IsValid() (bool, error) {
	valid := true
	problems := []string{}

	if p.Cluster == nil {
		problems = append(problems, "Parent Cluster not set")
		valid = false
	}
	if p.Cell == nil {
		problems = append(problems, "Parent Cell not set")
		valid = false
	}
	if p.Keyspace == nil {
		problems = append(problems, "Parent Keyspace not set")
		valid = false
	}
	if p.Shard == nil {
		problems = append(problems, "Parent Shard not set")
		valid = false
	}

	return valid, fmt.Errorf(strings.Join(problems, ", "))
}

func (s *VitessTabletSpec) SetParentSet(ps VitessTabletParentSet) error {
	if valid, err := ps.IsValid(); !valid {
		return fmt.Errorf("Invalid Parents for VitessTablet: %s", err)
	}
	s.parentSet = ps
	return nil
}

func (s *VitessTabletSpec) MustSetParentSet(ps VitessTabletParentSet) {
	err := s.SetParentSet(ps)
	if err != nil {
		panic(err)
	}
}

func (s *VitessTabletSpec) GetParents() *VitessTabletParentSet {
	return &s.parentSet
}

func (vt *VitessTablet) GetFullName() string {
	return strings.Join([]string{vt.Spec.parentSet.Cluster.GetName(), vt.Spec.parentSet.Cell.GetName(), vt.Spec.parentSet.Keyspace.GetName(), vt.Spec.parentSet.Shard.GetName(), vt.GetName()}, "-")
}

func (vt *VitessTablet) GetReplicas() *int32 {
	if vt.Spec.Replicas != nil {
		return vt.Spec.Replicas
	}

	if vt.Spec.parentSet.Shard.Spec.Defaults != nil && vt.Spec.parentSet.Shard.Spec.Defaults.Replicas != nil {
		return vt.Spec.parentSet.Shard.Spec.Defaults.Replicas
	}

	var def int32
	return &def
}

func (vt *VitessTablet) GetClusterName() string {
	return vt.Spec.parentSet.Cluster.GetName()
}

func (vt *VitessTablet) GetCellName() string {
	return vt.Spec.parentSet.Cell.GetName()
}

func (vt *VitessTablet) GetKeyspaceName() string {
	return vt.Spec.parentSet.Keyspace.GetName()
}

func (vt *VitessTablet) GetShardName() string {
	return vt.Spec.parentSet.Shard.GetName()
}

func (vt *VitessTablet) GetDBNameAndConfig() (string, *VTContainer) {
	// Inheritance order, with most specific first
	providers := []ConfigProvider{
		vt,
		vt.Spec.parentSet.Shard,
		vt.Spec.parentSet.Keyspace,
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
		vt.Spec.parentSet.Shard,
		vt.Spec.parentSet.Keyspace,
	}

	for _, p := range providers {
		// TODO: More DB providers
		if containers := p.GetTabletContainers(); containers != nil && containers.VTTablet != nil {
			return containers.VTTablet
		}
	}
	return nil
}
