package common

// In Go, it's better to put helper types in one file
// However, if it's a big component like a RecordAccumulator, then it's better
//   to put it in its own file.

type ProducerConfig struct {
	LingerMs     int
	BatchSize    int
	BufferMemory int
	Acks         string
}

// TODO: Search "In Go, when to use pointer vs value fields?"
// - Also: "In Go, should I make constructors to enforce default values?"
//   -- "Is this idiomatic in Go if a type has a field that need to be a
//       certain value"
//   -- "Would it be better to just keep fields unexported in these cases?"
//   -- "Won't I just end up doing it the Java way where I have getters and setters?"

// https://github.com/apache/kafka/blob/trunk/clients/src/main/java/org/apache/kafka/clients/producer/ProducerRecord.java#L49
type ProducerRecord struct {
	Topic     string
	Partition *int // Make this a pointer since its optional
	Key       *string
	Value     string
}

// https://github.com/apache/kafka/blob/trunk/clients/src/main/java/org/apache/kafka/clients/producer/RecordMetadata.java#L26
type RecordMetadata struct {
	Offset         int64
	TopicPartition TopicPartition
}

// https://github.com/apache/kafka/blob/trunk/clients/src/main/java/org/apache/kafka/common/TopicPartition.java#L25
// - Like PartitionInfo, mostly getters and setters
type TopicPartition struct {
	Partition int
	Topic     string
}

// https://github.com/apache/kafka/blob/trunk/clients/src/main/java/org/apache/kafka/common/PartitionInfo.java#L25
//   - Also has info about leader, replicas, inSyncReplicas, offlineReplicas
//     -- Won't implement for now since I don't have replication
//   - Putting this in commonTypes.go since it's mostly getters and setters
type PartitionInfo struct {
	Topic     string
	Partition int
}
