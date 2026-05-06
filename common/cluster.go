package common

// https://github.com/apache/kafka/blob/trunk/clients/src/main/java/org/apache/kafka/common/Cluster.java#L35
//   - Also has info about nodes, controller, partitionsByNode, nodesById, etc.
//     -- But I'll skip these for now since I don't replication yet
//   - There's also invalidTopics, unauthorizedTopics, etc., but I don't yet
//     know the use of these
//   - I'll just use int for id's for now instead of UUID
type Cluster struct {
	// Map<TopicPartition, PartitionInfo> partitionsByTopicPartition
	partitionsByTopicPartition map[TopicPartition]PartitionInfo
	// Map<String, List<PartitionInfo>> partitionsByTopic
	partitionsByTopic map[string][]PartitionInfo
	// Map<String, Uuid> topicIds
	topicIds map[string]int
	// Map<Uuid, String> topicNames
	topicNames map[int]string
}

// For now, I'll make the default constructor of Cluster do what
// Cluster.empty() does (loosely)
//   - Cluster.empty() is defined as: "Create an empty cluster instance with no
//     nodes and no topic-partitions
//   - It uses the 2nd constructor in the Kafka codebase (the one that takes
//     6 arguments)
//     -- It actually gives the empty cluster a null clusterId
//   - There's all kinds of processing done in Cluster's constructor
//     implementation in the actual Kafka code, but I'll just initialize the
//     fields to empty but usable maps for now.
func NewCluster() *Cluster {
	return &Cluster{
		partitionsByTopicPartition: make(map[TopicPartition]PartitionInfo),
		partitionsByTopic:          make(map[string][]PartitionInfo),
		topicIds:                   make(map[string]int),
		topicNames:                 make(map[int]string),
	}
}

// AFTER THIS, look at how KafkaProducer initializes a ProducerMetadata
