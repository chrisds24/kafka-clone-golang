package clients

import (
	"github.com/chrisds24/kafka-clone-golang/common"
)

// https://github.com/apache/kafka/blob/trunk/clients/src/main/java/org/apache/kafka/clients/Metadata.java#L67
type Metadata struct {
	// FOR NOW, I only care about metadataSnapshot
	// - private volatile MetadataSnapshot metadataSnapshot = MetadataSnapshot.empty();
	//   -- The "volatile" keyword is related to multithreading
	metadataSnapshot *MetadataSnapshot
}

func NewMetadata() *Metadata {
	return &Metadata{
		// In actuality, metadataSnapshot is initialized when it is declared as
		//   a field of Metadata, where MetadataSnapshot.empty() is used to
		//   initialize it. But for now, I'll just make the constructor of
		//   MetadataSnapshot create empty fields.
		// - MetadataSnapshot.empty() also calls Cluster.empty() to initialize
		//   clusterInstance
		metadataSnapshot: NewMetadataSnapshot(),
	}
}

func (metadata *Metadata) Fetch() *common.Cluster {
	return metadata.metadataSnapshot.Cluster()
}
