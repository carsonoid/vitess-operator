package v1alpha2

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetEmbeddedCells processes the embedded cell map and returns a slice of cells
func (vc *VitessCluster) GetEmbeddedCells() (ret []*VitessCell) {
	for name, spec := range vc.Spec.Cells {
		ret = append(ret, &VitessCell{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: vc.GetNamespace(),
			},
			Spec: *spec,
		})
	}
	return
}

func (vc *VitessCluster) EmbedCell(cell *VitessCell) error {
	// First check the the cells map is initialized
	if vc.Spec.Cells == nil {
		vc.Spec.Cells = make(map[string]*VitessCellSpec)
	}

	// Then check to make sure it is not already defined in the
	// embedded resources
	if _, exists := vc.Spec.Cells[cell.GetName()]; exists {
		return fmt.Errorf("Error merging VitessCell: %s: Already defined in the VitessCluster", cell.GetName())
	}
	vc.Spec.Cells[cell.GetName()] = &cell.DeepCopy().Spec

	return nil
}

// GetEmbeddedKeyspaces processes the embedded keyspace map and returns a slice of keyspaces
func (vc *VitessCluster) GetEmbeddedKeyspaces() (ret []*VitessKeyspace) {
	for name, spec := range vc.Spec.Keyspaces {
		ret = append(ret, &VitessKeyspace{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: vc.GetNamespace(),
			},
			Spec: *spec,
		})
	}
	return
}

func (vc *VitessCluster) EmbedKeyspace(keyspace *VitessKeyspace) error {
	// First check the the cells map is initialized
	if vc.Spec.Keyspaces == nil {
		vc.Spec.Keyspaces = make(map[string]*VitessKeyspaceSpec)
	}

	// Then check to make sure it is not already defined in the
	// embedded resources
	if _, exists := vc.Spec.Keyspaces[keyspace.GetName()]; exists {
		return fmt.Errorf("Error merging VitessKeyspace %s: Already defined in the VitessCluster", keyspace.GetName())
	}
	vc.Spec.Keyspaces[keyspace.GetName()] = &keyspace.DeepCopy().Spec

	return nil
}

// GetEmbeddedShards processes the embedded keyspace map and returns a slice of shards
func (vc *VitessCluster) GetEmbeddedShards() (ret []*VitessShard) {
	for _, keyspace := range vc.GetEmbeddedKeyspaces() {
		ret = append(ret, keyspace.GetEmbeddedShards()...)
	}
	return
}

// GetEmbeddedShards processes the embedded keyspace map and returns a slice of shards
func (vc *VitessCluster) GetEmbeddedTablets() (ret []*VitessTablet) {
	for _, shard := range vc.GetEmbeddedShards() {
		ret = append(ret, shard.GetEmbeddedTablets()...)
	}
	return
}

func (vc *VitessCluster) GetLockserver() *VitessLockserver {
	if vc.Spec.Lockserver == nil {
		return nil
	}
	return &VitessLockserver{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vc.GetName(),
			Namespace: vc.GetNamespace(),
		},
		Spec: *vc.Spec.Lockserver,
	}
}

func (vc *VitessCluster) GetCellByID(cellID string) *VitessCell {
	spec, found := vc.Spec.Cells[cellID]
	if !found {
		return nil
	}

	return &VitessCell{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cellID,
			Namespace: vc.GetNamespace(),
		},
		Spec: *spec,
	}
}
