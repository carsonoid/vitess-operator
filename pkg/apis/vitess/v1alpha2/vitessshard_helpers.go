package v1alpha2

// GetTabletContainers satisfies ConfigProvider
func (vt *VitessShard) GetTabletContainers() *TabletContainers {
	if vt.Spec.Defaults != nil {
		return vt.Spec.Defaults.Containers
	}
	return nil
}
