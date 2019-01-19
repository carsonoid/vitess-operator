package normalizer

import (
	"errors"

	vitessv1alpha2 "vitess.io/vitess-operator/pkg/apis/vitess/v1alpha2"
)

func (n *Normalizer) ValidateCluster(cluster *vitessv1alpha2.VitessCluster) error {

	// Child tests from the top down
	lockserver := cluster.GetLockserver()
	if lockserver == nil {
		return errors.New("No embeddded lockserver from cluster after normalization")
	}

	cells := cluster.GetEmbeddedCells()
	if len(cells) == 0 {
		return errors.New("No embedded cells from cluster after normalization")
	}

	shards := cluster.GetEmbeddedShards()
	if len(shards) == 0 {
		return errors.New("No embedded shards from cluster after normalization")
	}

	tablets := cluster.GetEmbeddedTablets()
	if len(tablets) == 0 {
		return errors.New("No embedded tablets from cluster after normalization")
	}

	keyspaces := cluster.GetEmbeddedKeyspaces()
	if len(keyspaces) == 0 {
		return errors.New("No embedded keyspaces from cluster after normalization")
	}

	for _, keyspace := range keyspaces {
		shards := keyspace.GetEmbeddedShards()
		if len(shards) == 0 {
			return errors.New("No embedded shards from keyspace after normalization")
		}

		for _, shard := range shards {
			tablets := shard.GetEmbeddedTablets()
			if len(tablets) == 0 {
				return errors.New("No embedded tablets from shard after normalization")
			}
		}
	}

	// Parent tests from the bottom up

	// every tablet should have a parent cell, cluster, keyspace, and shard
	for _, tablet := range tablets {
		if tablet.GetCell() == nil {
			return errors.New("No parent cell in tablet after normalization")
		}
		if tablet.GetCluster() == nil {
			return errors.New("No parent cluster in tablet after normalization")
		}
		if tablet.GetKeyspace() == nil {
			return errors.New("No parent keyspace in tablet after normalization")
		}
		if tablet.GetShard() == nil {
			return errors.New("No parent shard in tablet after normalization")
		}
	}

	// every shard should have a parent keyspace and cluster
	for _, shard := range shards {
		if shard.GetKeyspace() == nil {
			return errors.New("No parent keyspace in shard after normalization")
		}

		if shard.GetCluster() == nil {
			return errors.New("No parent cluster in shard after normalization")
		}
	}

	// every keyspace should have a parent cluster
	for _, keyspace := range keyspaces {
		if keyspace.GetCluster() == nil {
			return errors.New("No parent cluster in keyspace after normalization")
		}
	}

	// every cell should have a parent cluster
	for _, cell := range cells {
		if cell.GetCluster() == nil {
			return errors.New("No parent cluster in cell after normalization")
		}
	}

	return nil
}
