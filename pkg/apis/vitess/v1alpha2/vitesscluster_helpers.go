package v1alpha2

import (
	"fmt"
)

func (vc *VitessCluster) AddCell(cell *VitessCell) error {
	// First check the the cells map is initialized
	if vc.Spec.Cells == nil {
		vc.Spec.Cells = make(map[string]VitessCellSpec)
	}

	// Then check to make sure it is not already defined in the
	// embedded resources
	if _, exists := vc.Spec.Cells[cell.GetName()]; exists {
		return fmt.Errorf("Error merging VitessCell: %s: Already defined in the VitessCluster", cell.GetName())
	}
	vc.Spec.Cells[cell.GetName()] = cell.DeepCopy().Spec

	return nil
}

func (vc *VitessCluster) AddKeyspace(keyspace *VitessKeyspace) error {
	// First check the the cells map is initialized
	if vc.Spec.Keyspaces == nil {
		vc.Spec.Keyspaces = make(map[string]VitessKeyspaceSpec)
	}

	// Then check to make sure it is not already defined in the
	// embedded resources
	if _, exists := vc.Spec.Keyspaces[keyspace.GetName()]; exists {
		return fmt.Errorf("Error merging VitessKeyspace %s: Already defined in the VitessCluster", keyspace.GetName())
	}
	vc.Spec.Keyspaces[keyspace.GetName()] = keyspace.DeepCopy().Spec

	return nil
}
