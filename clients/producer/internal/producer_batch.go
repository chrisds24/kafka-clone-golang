package internal

import (
	"github.com/chrisds24/kafka-clone-golang/common"
)

type ProducerBatch struct {
	topicPartition common.TopicPartition

	// SKIPPING MemoryRecordsBuilder, which takes in the ByteBuffer to
	//   create space for a new ProducerBatch
	// - https://github.com/apache/kafka/blob/trunk/clients/src/main/java/org/apache/kafka/common/record/internal/MemoryRecordsBuilder.java
	// recordCount int  // SKIPPING this too since I can just do len(records)
	// maxRecordSize int  // NOT SURE what this does yet

	// For now, the records are actually stored here
	// - But in actuality they are stored in memory obtained from the
	// BufferPool (...At least that's how I think it works)
	records []common.KeyValRecord
}

func NewProducerBatch(tp common.TopicPartition) *ProducerBatch {
	return &ProducerBatch{
		topicPartition: tp,
		// records is initialized to its zero value
		// - The zero value of a slice is usable safely
	}
}

// SOURCE: https://github.com/apache/kafka/blob/trunk/clients/src/main/java/org/apache/kafka/clients/producer/internals/ProducerBatch.java#L148
func (producerBatch *ProducerBatch) TryAppend(
	key string,
	value string,
	// I'm passing this so each ProducerBatch doesn't need to track a
	// max batch size field
	batchSize int,
	// ) *producer.RecordMetadata {
) bool { // TEMPORARILY returning bool for early synchronous implementation
	// producerBatch.records should actually never be > batchSize, only equal
	// at most
	if len(producerBatch.records) >= batchSize {
		// return nil
		return false
	}

	// In Go, append returns a new slice but doesn't modify the original slice
	// - Need to assign to the ProducerBatch's records field
	producerBatch.records = append(
		producerBatch.records,
		common.KeyValRecord{
			Key:   key,
			Value: value,
		},
	)

	// this.recordCount is batchIndex in FutureRecordMetadata's constructor
	// - For my case, this would just be len(producerBatch.records) - 1, which
	// is literally just the index in which the new record has been added to
	// - Refer to https://github.com/apache/kafka/blob/trunk/clients/src/main/java/org/apache/kafka/clients/producer/internals/FutureRecordMetadata.java#L19
	//
	// batchIndex is used in RecordMetadata's constructor to set the offset
	// field, but RecordMetadata is not initialized anywhere in
	// FutureRecordMetadata (which implements Future<RecordMetadata>)
	//
	// Regarding the offset, I also don't see how I can get the offset here,
	// which the broker returns (So I can only get it once the actual send
	// to the broker is done)
	//
	// IMPORTANT: The reason for this is that the original code uses a Future
	// which is asynchronous and the async send in the original can fill in
	// these info once the promise returns.
	// - Since I don't have an async sender yet, I have no way of setting
	//   these info here.
	// - I can't really get an actual RecordMetadata, which is something that
	//   should come from the broker after the send
	//
	// TODO: So instead of returning RecordMetadata here, just return some
	// other value to signify if an append was performed on the batch vs if
	// no append was done
	//
	// AT THIS POINT, the only thing remaining in ProducerBatch's tryAppend in
	// the original code is creating + returning a RecordMetadata (since I'm
	// skipping stuff like thunks, maxRecordSize, etc.)
	// - INSTEAD: I'll just return a boolean (false if no append happened and
	// true if an append was indeed performed)
	return true
}

// MY TEMPORARY implementation of isFull. The original is actually defined
// in the MemoryRecordsBuilder class
func (producerBatch *ProducerBatch) IsFull(
	batchSize int,
) bool {
	return len(producerBatch.records) >= batchSize
}
