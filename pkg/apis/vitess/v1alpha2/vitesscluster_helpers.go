package v1alpha2

import (
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetEmbeddedCells processes the embedded cell map and returns a slice of cells
func (cluster *VitessCluster) GetEmbeddedCells() (ret []*VitessCell) {
	for name, spec := range cluster.Spec.Cells {
		ret = append(ret, &VitessCell{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: cluster.GetNamespace(),
			},
			Spec: *spec,
		})
	}
	return
}

func (cluster *VitessCluster) EmbedCell(cell *VitessCell) error {
	// First check the the cells map is initialized
	if cluster.Spec.Cells == nil {
		cluster.Spec.Cells = make(map[string]*VitessCellSpec)
	}

	// Then check to make sure it is not already defined in the
	// embedded resources
	if _, exists := cluster.Spec.Cells[cell.GetName()]; exists {
		return fmt.Errorf("Error merging VitessCell: %s: Already defined in the VitessCluster", cell.GetName())
	}
	cluster.Spec.Cells[cell.GetName()] = &cell.DeepCopy().Spec

	return nil
}

// GetEmbeddedKeyspaces processes the embedded keyspace map and returns a slice of keyspaces
func (cluster *VitessCluster) GetEmbeddedKeyspaces() (ret []*VitessKeyspace) {
	for name, spec := range cluster.Spec.Keyspaces {
		ret = append(ret, &VitessKeyspace{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: cluster.GetNamespace(),
			},
			Spec: *spec,
		})
	}
	return
}

func (cluster *VitessCluster) EmbedKeyspace(keyspace *VitessKeyspace) error {
	// First check the the cells map is initialized
	if cluster.Spec.Keyspaces == nil {
		cluster.Spec.Keyspaces = make(map[string]*VitessKeyspaceSpec)
	}

	// Then check to make sure it is not already defined in the
	// embedded resources
	if _, exists := cluster.Spec.Keyspaces[keyspace.GetName()]; exists {
		return fmt.Errorf("Error merging VitessKeyspace %s: Already defined in the VitessCluster", keyspace.GetName())
	}
	cluster.Spec.Keyspaces[keyspace.GetName()] = &keyspace.DeepCopy().Spec

	return nil
}

// GetEmbeddedShards processes the embedded keyspace map and returns a slice of shards
func (cluster *VitessCluster) GetEmbeddedShards() (ret []*VitessShard) {
	for _, keyspace := range cluster.GetEmbeddedKeyspaces() {
		ret = append(ret, keyspace.GetEmbeddedShards()...)
	}
	return
}

// GetEmbeddedShards processes the embedded keyspace map and returns a slice of shards
func (cluster *VitessCluster) GetEmbeddedTablets() (ret []*VitessTablet) {
	for _, shard := range cluster.GetEmbeddedShards() {
		ret = append(ret, shard.GetEmbeddedTablets()...)
	}
	return
}

func (cluster *VitessCluster) GetLockserver() *VitessLockserver {
	if cluster.Spec.Lockserver == nil {
		return nil
	}
	return &VitessLockserver{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.GetName(),
			Namespace: cluster.GetNamespace(),
		},
		Spec: *cluster.Spec.Lockserver,
	}
}

func (cluster *VitessCluster) GeBackupConfig() *VitessBackupConfig {
	if cluster.Spec.BackupConfig == nil {
		return nil
	}
	return &VitessLockserver{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.GetName(),
			Namespace: cluster.GetNamespace(),
		},
		Spec: *cluster.Spec.BackupConfig,
	}
}

func (cluster *VitessCluster) GetCellByID(cellID string) *VitessCell {
	spec, found := cluster.Spec.Cells[cellID]
	if !found {
		return nil
	}

	return &VitessCell{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cellID,
			Namespace: cluster.GetNamespace(),
		},
		Spec: *spec,
	}
}

func (cluster *VitessCluster) GetScopedName(extra ...string) string {
	return strings.Join(append(
		[]string{
			cluster.GetName(),
		},
		extra...), "-")
}
