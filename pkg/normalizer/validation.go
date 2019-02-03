package normalizer

import (
	vitessv1alpha2 "vitess.io/vitess-operator/pkg/apis/vitess/v1alpha2"
)

func (n *Normalizer) ValidateCluster(cluster *vitessv1alpha2.VitessCluster) error {
	if cluster.Lockserver() == nil {
		return ValidationErrorNoLockserverForCluster
	}

	if len(cluster.Cells()) == 0 {
		return ValidationErrorNoCells
	}

	for _, cell := range cluster.Cells() {
		if cell.Lockserver() == nil {
			return ValidationErrorNoLockserverForCell
		}
	}

	if len(cluster.Keyspaces()) == 0 {
		return ValidationErrorNoKeyspaces
	}

	if len(cluster.Shards()) == 0 {
		return ValidationErrorNoShards
	}

	if len(cluster.Tablets()) == 0 {
		return ValidationErrorNoTablets
	}

	for _, tablet := range cluster.Tablets() {
		if tablet.Cell() == nil {
			return ValidationErrorNoCellForTablet
		}
	}

	return nil
}
