package producer

import (
	"github.com/chrisds24/kafka-clone-golang/common"
)

// SOURCE: https://github.com/apache/kafka/blob/trunk/clients/src/main/java/org/apache/kafka/clients/producer/RecordMetadata.java
//   - FutureRecordMetadata actually implements Future<RecordMetadata>
//     -- FutureRecordMetadata is described as:
//     The future result of a record send
//     -- FutureRecordMetadata has a batchIndex field
//     -- It also has a ProduceRequestResult result field, which I'll skip for
//     now until I know more about it
//     -- HOWEVER, I'm skipping FutureRecordMetadata for now and just using
//     RecordMetadata since my clone is currently synchronous
//
// The metadata for a record that has been acknowledged by the server
type RecordMetadata struct {
	offset         int64
	topicPartition common.TopicPartition
}

// Gonna return a value instead of a pointer since the struct
// is small enough
func NewRecordMetadata(
	topicPartition common.TopicPartition,
	baseOffset int64,
	batchIndex int,
) RecordMetadata {
	var tempOffset int64
	if baseOffset == -1 {
		tempOffset = baseOffset
	} else {
		tempOffset = baseOffset + int64(batchIndex)
	}

	return RecordMetadata{
		offset:         tempOffset,
		topicPartition: topicPartition,
	}
}

// There's also hasOffset, which I'll skip for now

// The offset of the record in the topic/partition
// - Returns -1 if hasOffset returns false
func (recordMetadata RecordMetadata) Offset() int64 {
	return recordMetadata.offset
}

// The topic the record was appended to
func (recordMetadata RecordMetadata) Topic() string {
	return recordMetadata.topicPartition.Topic
}

// The partition the record was appended to
func (recordMetadata RecordMetadata) Partition() int {
	return recordMetadata.topicPartition.Partition
}
