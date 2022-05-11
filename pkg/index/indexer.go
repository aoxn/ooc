package index

import (
	pd "github.com/aoxn/wdrip/pkg/iaas/provider"
)

func NewGenericIndexer(
	id string, store pd.ObjectStorage,
) *GenericIndexer {

	return &GenericIndexer{
		SnapshotIndex: NewSnapshotIndex(id, store),
		ClusterIndex:  NewClusterIndex(id, store),
		NodePoolIndex: NewNodePoolIndex(id, store),
	}
}

type GenericIndexer struct {
	*SnapshotIndex
	*ClusterIndex
	*NodePoolIndex
}
