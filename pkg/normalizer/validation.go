package normalizer

import (
	vitessv1alpha2 "vitess.io/vitess-operator/pkg/apis/vitess/v1alpha2"
)

func (n *Normalizer) ValidateCluster(cluster *vitessv1alpha2.VitessCluster) error {
	if cluster.GetLockserver() == nil {
		return ValidationErrorNoLockserver
	}

	if len(cluster.GetEmbeddedCells()) == 0 {
		return ValidationErrorNoCells
	}

	if len(cluster.GetEmbeddedKeyspaces()) == 0 {
		return ValidationErrorNoKeyspaces
	}

	if len(cluster.GetEmbeddedShards()) == 0 {
		return ValidationErrorNoShards
	}

	if len(cluster.GetEmbeddedTablets()) == 0 {
		return ValidationErrorNoTablets
	}

	return nil
}
