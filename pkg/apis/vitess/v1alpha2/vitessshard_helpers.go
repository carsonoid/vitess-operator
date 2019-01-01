package v1alpha2

// import (
// 	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
// )

func (shard *VitessShard) GetTabletReplicas(tablet *VitessTablet, def int32) *int32 {
	if tablet.Spec.Replicas != nil {
		return tablet.Spec.Replicas
	}

	if shard.Spec.Defaults != nil && shard.Spec.Defaults.Replicas != nil {
		return shard.Spec.Defaults.Replicas
	}

	return &def
}
