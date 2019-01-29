package normalizer

import (
	vitessv1alpha2 "vitess.io/vitess-operator/pkg/apis/vitess/v1alpha2"
)

func (n *Normalizer) ValidateCluster(cluster *vitessv1alpha2.VitessCluster) error {
	if cluster.Lockserver() == nil {
		return ValidationErrorNoLockserver
	}

	if len(cluster.Cells()) == 0 {
		return ValidationErrorNoCells
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

	return nil
}
