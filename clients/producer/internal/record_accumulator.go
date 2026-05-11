package internal

import (
	"github.com/chrisds24/kafka-clone-golang/clients/producer"
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
	hasKey string, // Not sure if I need this here
	value string,
	cluster common.Cluster,
) RecordAppendResult {
	// If topicInfo for a topic doesn't exist, make one
	// - Remember how the RecordAccumulator constructor above only initializes
	// the topicInfoMap (map of topicInfo for existing topics)
	var topicInfo *topicInfo
	if accumulator.topicInfoMap[topic] != nil {
		topicInfo = accumulator.topicInfoMap[topic]
	} else {
		// Initialize a topicInfo for the topic that has no queue
		// of producer batches for each partition
		accumulator.topicInfoMap[topic] = newTopicInfo()
	}

	// Get the effective partition
	// - FOR NOW, I'm SKIPPING using the BuiltInPartitioner if the
	// record's partition is unknown (not specified)
	// - So this code is pointless for now other than for naming
	effectivePartition := partition

	// TODO: appendNewBatch
}

// Append a new batch to the queue
func (accumulator *RecordAccumulator) appendNewBatch(
	topic string,
	partition int,
	// Passing a []*ProducerBatch here does not copy the whole slice so it's ok
	dq []*ProducerBatch,
	key string,
	hasKey string,
	value string,
	// In the original code, a ByteBuffer is passed. But I'll just pass the
	//   fake BufferPool here so I can update the memory/space used
	bufferPool BufferPool,
) RecordAppendResult {
	// incomplete.add(batch)
	// - SKIPPING this for now, but incomplete is a RecordAccumulator field, so
	//   I need a pointer receiver here
}

//   private RecordAppendResult tryAppend(
// 	long timestamp, byte[] key, byte[] value, Header[] headers,
//     Callback callback, Deque<ProducerBatch> deque, long nowMs) {

// Try to append to a ProducerBatch
func (accumulator *RecordAccumulator) tryAppend(
	key string,
	hasKey string,
	value string,
	deque []*ProducerBatch,
) *RecordAppendResult { // Returning a pointer so I can return nil
	// The queue exists in the first place
	// - Remember that []*ProducerBatch is the queue of ProducerBatches for a
	//   partition
	// - The original Kafka code actually uses last = deque.peekLast(), which
	//   returns null if the queue is empty
	if deque != nil {
		// Here, there's at least 1 batch in the queue
		// TODO Replace []*ProducerBatch with an actual queue of *ProducerBatch
		// - NOTE: Even though len(deque) is used, this is actually an O(1)
		// operation since slices in Go store their length as metadata
		last := deque[len(deque)-1]
		// Go automatically dereferences pointers when calling methods
		recMeta := last.TryAppend(key, value, accumulator.batchSize)
		if recMeta == nil {
			// last.closeForRecordAppends(); WON'T NEED THIS FOR NOW
			return nil
		} else {
			// Return a RecordAppendResult
		}
	}
	return nil
	// So in these cases:
	// - The queue of batches for this partition doesn't exist
	// - The latest batch is full
	// We return nil, so that the code in appendNewBatch knows to create a
	// new batch
}

// Metadata about a record just appended to the record accumulator
type RecordAppendResult struct {
	// This is actually public final FutureRecordMetadata future, but I'll
	// just use RecordMetadata for now
	RecMeta         producer.RecordMetadata
	BatchIsFull     bool
	NewBatchCreated bool
}

// In RecordAccumulator.appendNewBatch, just update bufferPool's fields
//   instead of doing the actual allocation that Kafka does
