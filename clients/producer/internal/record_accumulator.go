package internal

import (
	"github.com/chrisds24/kafka-clone-golang/common"
)

type RecordAccumulator struct {
	batchSize int
	lingerMs  int
	// ConcurrentMap<String /*topic*/, TopicInfo> topicInfoMap = new CopyOnWriteMap<>();
	// - A TopicInfo actually holds the queue of ProducerBatches for a
	// partition
	// - However, the memory used to create that producer batch is obtained
	//   from the bufferPool, which is a ByteBuffer
	topicInfoMap map[string]*topicInfo

	// Should I change map fields in other structs to be pointer
	// fields instead of value fields?
	// - No, maps are already reference-like types in Go

	// BufferPool free
	// - NOT AN ACTUAL BufferPool. Just simulating memory limits/management
	//   using number of records instead of actual bytes
	free *BufferPool

	// TODO: private final IncompleteBatches incomplete
}

func NewRecordAccumulator(
	batchSize int,
	lingerMs int,
	bufferPool *BufferPool,
) *RecordAccumulator {
	return &RecordAccumulator{
		batchSize: batchSize,
		lingerMs:  lingerMs,
		// NOTE: This only initializes topicInfoMap to a non-nil usable state,
		// but doesn't initialize each topicInfo (which has the batches field,
		// which is a map) to a usable state
		// REMEMBER HOW Kafka has: topicInfo.batches.computeIfAbsent...
		topicInfoMap: make(map[string]*topicInfo),
		free:         bufferPool,
	}
}

type topicInfo struct {
	// public final ConcurrentMap<Integer /*partition*/, Deque<ProducerBatch>> batches = new CopyOnWriteMap<>();
	// TODO: Replace []*ProducerBatch with an actual queue of *ProducerBatch
	Batches map[int][]*ProducerBatch
	// SKIPPING BuiltInPartitioner for now
}

func newTopicInfo() *topicInfo {
	return &topicInfo{
		Batches: make(map[int][]*ProducerBatch),
	}
}

func (accumulator *RecordAccumulator) Append(
	topic string,
	partition int,
	key string,
	hasKey bool, // Not sure if I need this here
	value string,
	cluster *common.Cluster,
) RecordAppendResult {
	// If topicInfo for a topic doesn't exist, make one
	// - Remember how the RecordAccumulator constructor above only initializes
	// the topicInfoMap (map of topicInfo for existing topics)
	if accumulator.topicInfoMap[topic] == nil {
		// Initialize a topicInfo for the topic that has no queue
		// of producer batches for each partition
		accumulator.topicInfoMap[topic] = newTopicInfo()
	}

	// effectivePartition
	// - FOR NOW, I'm SKIPPING using the BuiltInPartitioner if the
	// record's partition is unknown (not specified)
	// - So I'll just use partition for now

	// Deque<ProducerBatch> dq = topicInfo.batches.computeIfAbsent(effectivePartition, k -> new ArrayDeque<>());
	// - Here, topicInfo for the topic already exists since newTopicInfo has
	// been called above.
	// - That topicInfo also has a usable batches field, which is a map
	// - Each []*ProducerBatch for a partition (which is the int key) is also
	// usable since the zero value of a slice is usable in Go (we can append
	// to it)
	// - Below, dq points to the actual []*ProducerBatch for the given partition
	// - So if we edit an existing *ProducerBatch w/ a method that has a pointer
	// receiver, we directly modify that ProducerBatch
	// - HOWEVER, reassigning dq itself (Ex. append new batch to dq) won't
	// change the queue in accumulator.topicInfoMap[topic].Batches[partition]
	// so we need to edit accumulator.topicInfoMap[topic].Batches[partition]
	// itself (SEE appendNewBatch)
	dq := accumulator.topicInfoMap[topic].Batches[partition]

	// RecordAppendResult appendResult = appendNewBatch(topic, effectivePartition, dq, timestamp, key, value, headers, callbacks, buffer, nowMs);
	return accumulator.appendNewBatch(
		topic,
		partition,
		dq,
		key,
		hasKey,
		value,
		// accumulator.free,
	)

	// SKIPPING these:
	// if (appendResult.newBatchCreated)
	//     buffer = null;
	// enableSwitch
	// updatePartitionInfo
}

// Append a new batch to the queue
func (accumulator *RecordAccumulator) appendNewBatch(
	topic string,
	partition int,
	// Passing a []*ProducerBatch here does not copy the whole slice so it's ok
	dq []*ProducerBatch,
	key string,
	hasKey bool,
	value string,
	// In the original code, a ByteBuffer is passed
	// bufferPool *BufferPool,  // NOT NEEDED FOR NOW
) RecordAppendResult {
	appendResult := accumulator.tryAppend(key, hasKey, value, dq)
	// My logic here isn't the same logic as the original for now.
	// But my TEMPORARY logic is that we simply return the appendResult
	// here if an append was successful
	if appendResult != nil {
		return *appendResult
	}

	// Make a new ProducerBatch, add the key-val record to it, then add the
	// batch to dq
	// SKIPPING    MemoryRecordsBuilder recordsBuilder = recordsBuilder(buffer)
	batch := NewProducerBatch(
		common.TopicPartition{
			Topic:     topic,
			Partition: partition,
		},
	)
	// Again, Go automatically dereferences batch (which is a pointer) when
	// calling its method
	batch.TryAppend(key, value, accumulator.batchSize)

	// Add the newly created batch to the queue of batches
	// - Again, slice append in Go creates a new copy instead of modifying
	// what is "pointed to" by dq
	// - UNLIKE in Java, reassigning dq does not modify the []*ProducerBatch
	// for a partition in topicInfo
	// - Need to reassign the []*ProducerBatch for
	// accumulator.topicInfoMap[topic].Batches[partition]
	dq = append(dq, batch)
	accumulator.topicInfoMap[topic].Batches[partition] = dq

	// SKIPPING    incomplete.add(batch);

	// return new RecordAppendResult(future, dq.size() > 1 || batch.isFull(), true, batch.estimatedSizeInBytes());
	return RecordAppendResult{
		// Even though we just created a new batch, adding the record to it
		// could have made the new batch full
		BatchIsFull:     batch.IsFull(accumulator.batchSize),
		NewBatchCreated: true,
	}
}

// Try to append to a ProducerBatch
func (accumulator *RecordAccumulator) tryAppend(
	key string,
	hasKey bool,
	value string,
	deque []*ProducerBatch,
) *RecordAppendResult { // Returning a pointer so I can return nil
	// The queue exists in the first place
	// - Remember that []*ProducerBatch is the queue of ProducerBatches for a
	//   partition
	// if deque != nil { // The original Kafka code uses a null check
	// - But I'll just use a len check since len == 0 for nil slices in Go
	// - This also takes care of the case of a non-nil but empty slice
	// (Which shouldn't really happen in the first place)
	if len(deque) > 0 {
		// Here, there's at least 1 batch in the queue
		// TODO: Replace []*ProducerBatch with an actual queue of *ProducerBatch
		// - NOTE: Even though len(deque) is used, this is actually an O(1)
		// operation since slices in Go store their length as metadata
		// - IMPORTANT: This copies the pointer to the ProducerBatch, so we
		// can directly modify the original ProducerBatch via last
		last := deque[len(deque)-1]
		// NOTE: For now, I'm not returning a FutureRecordMetadata from
		// a ProducerBatch's tryAppend. Instead I'm only returning a bool
		// that indicates if an append to the batch was performed or not
		//   recMeta := last.TryAppend(key, value, accumulator.batchSize)
		//   if recMeta == nil {
		// 	  // last.closeForRecordAppends(); WON'T NEED THIS FOR NOW
		// 	  return nil
		//
		// Go automatically dereferences pointers when calling methods so no
		// need to do *last
		//
		// IMPORTANT: In ProducerBatch's TryAppend method, it uses a pointer
		// receiver and reassigns the batch's records field but with the new
		// record added to the batch.
		// - So the original ProducerBatch itself is modified, not just a copy
		//   of it
		appended := last.TryAppend(key, value, accumulator.batchSize)
		if appended {
			// Return a RecordAppendResult
			// - return new RecordAppendResult(future, deque.size() > 1 || last.isFull(), false, appendedBytes);
			// - I'LL SKIP the deque.size() > 1 logic
			return &RecordAppendResult{
				// BatchIsFull would be true if the append caused the
				// last batch to be full even if it wasn't before the add
				// Though obviously, no new batch is created here
				BatchIsFull:     last.IsFull(accumulator.batchSize),
				NewBatchCreated: false,
			}
		} else {
			// last.closeForRecordAppends(); WON'T NEED THIS FOR NOW
			//
			// In this case, no append happened since the last batch is
			// full
			return nil
		}
	}

	// In the original code, we get here when the queue is empty or the last
	// batch is full. FOR NOW, my code only gets here when the queue is
	// empty
	// - Regardless of the cause, a new batch is created when calling tryAppend
	// from appendNewBatch
	return nil

	// So this function returns nil in two cases:
	// - The queue of batches for this partition doesn't exist
	// - The latest batch is full
}

// Metadata about a record just appended to the record accumulator
type RecordAppendResult struct {
	// This is actually public final FutureRecordMetadata future, but I'll
	// just use RecordMetadata for now
	// - I actually can't even have RecordMetadata here until I have an
	// async Sender
	// RecMeta         producer.RecordMetadata
	BatchIsFull     bool
	NewBatchCreated bool
}

// In RecordAccumulator.appendNewBatch, just update bufferPool's fields
//   instead of doing the actual allocation that Kafka does
