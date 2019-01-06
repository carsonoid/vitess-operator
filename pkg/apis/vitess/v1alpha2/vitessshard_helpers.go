package v1alpha2

import (
	"fmt"
)

// GetTabletContainers satisfies ConfigProvider
func (vs *VitessShard) GetTabletContainers() *TabletContainers {
	if vs.Spec.Defaults != nil {
		return vs.Spec.Defaults.Containers
	}
	return nil
}

func (vs *VitessShard) AddTablet(tablet *VitessTablet) error {
	return vs.Spec.AddTablet(tablet)
}

func (vss *VitessShardSpec) AddTablet(tablet *VitessTablet) error {
	// First check the the tablet map is initialized
	if vss.Tablets == nil {
		vss.Tablets = make(map[string]*VitessTabletSpec)
	}

	// Then check to make sure it is not already defined in the
	// embedded resources
	if _, exists := vss.Tablets[tablet.GetName()]; exists {
		return fmt.Errorf("Error merging VitessTablet %s: Already defined in the VitessShard", tablet.GetName())
	}

	// TODO make sure tabletIds don't overlap

	vss.Tablets[tablet.GetName()] = &tablet.DeepCopy().Spec

	return nil
}
