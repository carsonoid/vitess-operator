package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (cell *VitessCell) SetParent(cluster *VitessCluster) {
	cell.Spec.parent.Cluster = cluster
}

func (cell *VitessCell) GetCluster() *VitessCluster {
	return cell.Spec.parent.Cluster
}

func (vc *VitessCell) GetLockserver() *VitessLockserver {
	if vc.Spec.Lockserver == nil {
		return nil
	}
	return &VitessLockserver{
		ObjectMeta: metav1.ObjectMeta{Name: vc.GetName(), Namespace: vc.GetNamespace()},
		Spec:       *vc.Spec.Lockserver,
	}
}
