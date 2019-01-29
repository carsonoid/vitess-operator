package v1alpha2

import (
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (cell *VitessCell) SetParentCluster(cluster *VitessCluster) {
	cell.Spec.parent.Cluster = cluster
}

func (cell *VitessCell) Cluster() *VitessCluster {
	return cell.Spec.parent.Cluster
}

func (cell *VitessCell) Lockserver() *VitessLockserver {
	if cell.Spec.Lockserver == nil {
		return nil
	}
	return &VitessLockserver{
		ObjectMeta: metav1.ObjectMeta{Name: cell.GetName(), Namespace: cell.GetNamespace()},
		Spec:       *cell.Spec.Lockserver,
	}
}

func (cell *VitessCell) GetScopedName(extra ...string) string {
	return strings.Join(append(
		[]string{
			cell.Cluster().GetName(),
			cell.GetName(),
		},
		extra...), "-")
}
