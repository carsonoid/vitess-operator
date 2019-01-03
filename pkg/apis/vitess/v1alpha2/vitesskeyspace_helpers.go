package v1alpha2

// GetTabletContainers satisfies ConfigProvider
func (vk *VitessKeyspace) GetTabletContainers() *TabletContainers {
	if vk.Spec.Defaults != nil {
		return vk.Spec.Defaults.Containers
	}
	return nil
}
