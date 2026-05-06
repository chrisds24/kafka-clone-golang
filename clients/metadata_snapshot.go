package clients

import (
	"github.com/chrisds24/kafka-clone-golang/common"
)

// https://github.com/apache/kafka/blob/trunk/clients/src/main/java/org/apache/kafka/clients/MetadataSnapshot.java#L47
type MetadataSnapshot struct {
	// FOR NOW, I only care about the clusterInstance
	// - private Cluster clusterInstance
	// Notice how this is a pointer. I want to avoid copying
	clusterInstance *common.Cluster
}

// For now, I'll make the constructor of MetadataSnapshot default to what
// MetadataSnapshot.empty() does (loosely):
//   - MetadataSnapshot.empty() calls Cluster.empty() to initialize the
//     clusterInstance.
func NewMetadataSnapshot() *MetadataSnapshot {
	return &MetadataSnapshot{
		clusterInstance: common.NewCluster(),
	}
}

// Not unidiomatic to have a getter since the clusterInstance is unexported
// for a good reason. The original Kafka code also throws an exception if
// clusterInstance is null
// It's ok to return the pointer since the Cluster's fields are unexported
// and can't be used in an unsafe way
// Notice the use of a pointer receiver
func (metadataSnapshot *MetadataSnapshot) Cluster() *common.Cluster {
	return metadataSnapshot.clusterInstance
}
