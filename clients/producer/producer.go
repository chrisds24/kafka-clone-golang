package producer

import (
	"github.com/chrisds24/kafka-clone-golang/clients/producer/internal"
	"github.com/chrisds24/kafka-clone-golang/common"
)

// Unexported so clients need to use the constructor
type producer struct {
	// Config such as lingerMs, batchSize, acks, etc.
	producerConfig common.ProducerConfig
	metadata       *internal.ProducerMetadata
	// TODO: RecordAccumulator accumulator
}

// Constructor for a producer
//   - Constructors in Go: https://go.dev/doc/effective_go#composite_literals
//     -- In the link, new returns an address
//     -- Note that if we create a struct, put it in the variable x, then
//     return the address of x, the struct survives after the function returns
//   - Naming "constructors": https://go.dev/doc/effective_go#package-names
//     -- Just call this one new since clients of the package see this as
//     producer.New
//   - https://stackoverflow.com/questions/18125625/constructors-in-go
//   - https://www.reddit.com/r/golang/comments/1nc4nnh/is_using_constructor_in_golang_a_bad_pattern/
//     -- If this returned the struct itself, instead of the pointer, the
//     struct would be copied to the caller's stack frame
func New(producerConfig common.ProducerConfig) *producer {
	// TODO: Logic to initialize RecordAccumulator
	return &producer{
		producerConfig: producerConfig,
		metadata:       internal.NewProducerMetadata()}
}

// FOR NOW: I won't have a doSend, only send
func (prod *producer) Send(record common.ProducerRecord) common.RecordMetadata {
	cluster := prod.waitOnMetadata(record.Topic, record.Partition)
}

// waitOnMetadata()
//   - This returns a ClusterAndWaitTime, but the thing I'm interested in for
//     now is cluster.
//     -- https://github.com/apache/kafka/blob/trunk/clients/src/main/java/org/apache/kafka/common/Cluster.java#L35
//   - Gonna skip a lot of stuff for now. For example, the Sender is the one
//     responsible for the actual fetching, but I don't have a Sender for now
//     -- I'll just
func (prod *producer) waitOnMetadata(
	topic string,
	partition int,
) *common.Cluster {
	cluster := prod.metadata.Fetch()

	// TODO: Just hardcode topics and partitions into the cluster here
	// - Or do it when the Cluster gets initialized in its constructor

	return cluster
}

func (prod *producer) partition(record common.ProducerRecord) int {
	if record.Partition != nil {

	}
}

/*
*******************************************************************************
*************			WHAT I'LL DO (APR 28, 2026)        *******************
*******************************************************************************

Won't include ConsoleProducer info. Also not gonna go to deep into the details
I'm skipping multithreading/concurrency/locks concepts for now
I'm also skipping transactions
Basically, I'm skipping things that I don't know about or currently unsure
	how much I need it at the moment

Useful links:
- https://github.com/apache/kafka/blob/trunk/clients/src/main/java/org/apache/kafka/clients/producer/KafkaProducer.java
- https://github.com/apache/kafka/blob/trunk/clients/src/main/java/org/apache/kafka/clients/producer/internals/RecordAccumulator.java
- https://github.com/apache/kafka/blob/trunk/clients/src/main/java/org/apache/kafka/clients/producer/internals/ProducerBatch.java

KafkaProducer.send
- Asynchronously send record to a topic
- Calls doSend

KafkaProducer.doSend
  - The actual thing doing the sending to the buffer (queue of batches for a
    partition)
    -- As will be mentioned later, partition can either be specified in the
    record or calculated later (in doSend or in RecordAccumulator)
  - Wait for metadata about the cluster (so we know if the topic exists, which
    partitions there are, which are the leader brokers for each partition, etc.)
    -- Producer obviously needs this to be able to send to the correct
    destination
    -- SIDE NOTE: Metadata could be cached
	-- This is done by calling
	   waitOnMetadata(record.topic(), record.partition(), nowMs, maxBlockTimeMs)
  - Serialize the key and the value
  - Calculate the partition (could return unknown partition)
    -- If: partition specified in record -> Just return that partition
    -- Else if: Custom partitioner specified -> Use that to compute the partition
    -- Else if: Key is provided and built-in partitioner doesn't ignore keys
    -> Compute from the key
    -- Else: No key or built-in partitioner ignores keys
    -> return that the partition is unknown
  - Ensure the serialized record is of valid size
  - ********** IMPORTANT: Append to the accumulator (RecordAccumulator) by
    calling RecordAccumulator.append:
    RecordAccumulator.RecordAppendResult result = accumulator.append(record.topic(), partition, timestamp, serializedKey,
    serializedValue, headers, appendCallbacks, remainingWaitMs, nowMs, cluster);
    -- SEE DETAILS BELOW in RecordAccumulator.append
  - ........ At this point, the record has been appended to the queue of batches,
    but send/doSend still has some remaining things to do ........
  - Once the append to the accumulator returns, ensure that the record's
    partition is not unknown
  - If batch is full or a new batch has been created, wake up the Sender
    -- The Sender is now responsible for sending the batches to the cluster when
    appropriate)
    -- TODO: How should I implement this concept of being able to send to a
    batch but have some kind "background" logic that sends batches after
    a certain amount of time (Ex. based on linger.ms)?
  - Will I need threads early on in the project after all?
  - TODO: Think of an early simple way to implement linger.ms, but don't
    spend too much time on the logic since I'll replace it anyway
  - Return the result of the append (just a Future<RecordMetadata>)

KafkaProducer.waitOnMetadata
- IMPORTANT: I'll just go over this quick w/o diving much into the details +
  skipping certain details.
  -- Also: Lots of timing related stuff, which I'll skip
- Get metadata.
  -- If the producer already has it (cached), just use that
  -- Otherwise, fetch it from the broker
     + The Sender is the one responsible for the actual fetching
	 + This "fetching" happens in a loop, where we keep on asking for metadata
	   as long as we don't have it.
- This returns a ClusterAndMetadata.
  -- FOR NOW: I'm only interested in Cluster
- *** IMPORTANT: ProducerMetadata metadata field of KafkaProducer
  https://github.com/apache/kafka/blob/trunk/clients/src/main/java/org/apache/kafka/clients/producer/internals/ProducerMetadata.java#L37
- *** Metadata (which ProducerMetadata extends)
  -- https://github.com/apache/kafka/blob/trunk/clients/src/main/java/org/apache/kafka/clients/Metadata.java#L67
	 + The fetch() function used in "Cluster cluster = metadata.fetch()" in
	   waitOnMetadata() is actually in Metadata
	 + fetch() calls metadataSnapshot.cluster()
- *** MetadataSnapshot
  https://github.com/apache/kafka/blob/trunk/clients/src/main/java/org/apache/kafka/clients/MetadataSnapshot.java#L47
  -- Has controller, topicId, topicNames, nodes, partitions, etc.
  -- Has the cluster function called in metadataSnapshot.cluster()
     + cluster() simply returns clusterInstance, which is a Cluster (SEE BELOW)

Cluster
- https://github.com/apache/kafka/blob/trunk/clients/src/main/java/org/apache/kafka/common/Cluster.java#L35
- I won't go over this, but here are some fields (NOT ALL):
  -- private final List<Node> nodes
  -- private final Set<String> invalidTopics
  -- private final Node controller
  -- private final Map<TopicPartition, PartitionInfo> partitionsByTopicPartition;
  -- private final Map<String, List<PartitionInfo>> partitionsByTopic;
  -- private final Map<String, Uuid> topicIds;
  -- private final Map<Uuid, String> topicNames;
  -- *** There's other interesting ones, but I'll just skip those.
     + Looking at this, it's basically just metadata about the cluster

RecordAccumulator (some important fields)
  - **** I'll skip some, since I either don't understand it or don't need it at
    the moment
  - private final int batchSize;
  - private final Compression compression;
  - private final int lingerMs;
  - private final ExponentialBackoff retryBackoff;
  - private final int deliveryTimeoutMs;
  - private final long partitionAvailabilityTimeoutMs;
  - private final BufferPool free;
    -- This is where we get a buffer from when appending a new batch since the
    latest for a partition's queue is either full or non-existent(queue has
    no batch to begin with)
    -- TODO: I probably won't need a BufferPool early on, since the reason for
    that is that the producer has memory bounds on how much memory it could
    use and we have different threads sharing that memory
  - Instead, maybe I can just have a maximum total number of unsent records
    the producer can hold
  - private final ConcurrentMap<String topic, TopicInfo> topicInfoMap
    -- This is where we store info about each topic
  - private final IncompleteBatches incomplete;
    -- private final Set<ProducerBatch> incomplete;
  - Basically a set of incomplete batches
  - **********
  - A RecordAccumulator can be created with a custom partitioner or using the
    built-in (default) partitioner config

-------------- Not inside RecordAccumulator, but used there ---------
private static class TopicInfo
  - public final ConcurrentMap<Integer partition, Deque<ProducerBatch>> batches
    -- This is where we store the queue of batches for each partition
  - public final BuiltInPartitioner builtInPartitioner

public final class ProducerBatch
  - final TopicPartition topicPartition
    -- Obviously, a batch needs info about which topic and partition it is for
  - private final MemoryRecordsBuilder recordsBuilder
    -- This is where a ProducerBatch instance stores memory/data about records
    -- As will be shown later, RecordAccumulator.appendNewBatch creates a
    MemoryRecordsBuilder using an allocated buffer in
    RecordAccumulator.append, then passes this MemoryRecordsBuilder to
    ProducerBatch's constructor when making the new ProducerBatch instance
  - This is when creating a new batch
  - int recordCount;
  - int maxRecordSize;

-------------------------------------------

RecordAccumulator.append
  - DESCRIPTION: Add a record to the accumulator.
    -- Returns the append result, which will contain the future metadata, and
    flag for whether the appended batch is full or a new batch is created
  - This flag will be used to wake up the sender in KafkaProducer.doSend
    if the batch is indeed full or a new batch has been created
  - Get TopicInfo of the topic from the topicInfoMap, or create one if it
    doesn't exist
  - Initialize a null ByteBuffer (used later when we need to allocate a buffer
    that will be used by a new batch)
    -- TODO: As mentioned above, I won't be using a BufferPool so I won't need
    a ByteBuffer here
  - ********************* Loop **************************
  - ******* IMPORTANT: As mentioned above, I'll be skipping multithreading,
    concurrency, and locks related details.
    -- So these won't actually be run inside a loop, I JUST PUT THE LOOP
    HERE SO I CAN EASILY COMPARE WITH THE ACTUAL CODE
  - If partition is still unknown, find the effective partition using the
    BuiltInPartitioner, which uses sticky partitioning
    -- REMINDER: This BuiltInPartitioner is obtained from the TopicInfo
  - Check if we have an in-progress batch for the specified partition
    -- Obtain this from TopicInfo.batches
  - This is ConcurrentMap<Integer partition, Deque<ProducerBatch>> batches
    from above
    -- ME: It would be more correct to say: "Check if we have a queue of batches
    for the specified partition"
    -- We'll try to append (RecordAccumulator.tryAppend) to this queue later.
  - /////////// IMPORTANT CHANGE !!! /////////////
  - Since I won't be dealing with any multithreading and buffers yet, I'll just
    skip the first synchronized(dq) { ... tryAppend... } since it gets called
    again in RecordAccumulator.appendNewBatch
  - I won't need the if (buffer == null) { ... }
  - Basically, I'm just calling appendNewBatch to call
    RecordAccumulator.tryAppend
  - //////////////////////////////////////////////
  - Call RecordAccumulator.appendNewBatch, which calls
    RecordAccumulator.tryAppend, which then calls ProducerBatch.tryAppend
    -- RecordAppendResult appendResult = appendNewBatch(topic, effectivePartition, dq, timestamp, key, value, headers, callbacks, buffer, nowMs);
    -- This returns RecordAppendResult
    -- *** SEE DETAILS about RecordAccumulator.appendNewBatch,
    RecordAccumulator.tryAppendBelow, and ProducerBatch.tryAppend below
  - ////// IMPORTANT ////////: I'll be changing some code for these to fit
    what I'm doing for now
  - Finally, return the RecordAppendResult from RecordAccumulator.appendNewBatch
  - ****************** end of Loop **********************

RecordAppendResult
  - public final FutureRecordMetadata future
    -- Basically, just the metadata about the added record.
  - public final boolean batchIsFull
  - public final boolean newBatchCreated
  - public final int appendedBytes
  - *** NOTE: batchIsFull and newBatchCreated are the flags that doSend uses to
    wake up the Sender

public final class FutureRecordMetadata implements Future<RecordMetadata>

	private final ProduceRequestResult result;
	private final int batchIndex;
	private final long createTimestamp;
	private final int serializedKeySize;
	private final int serializedValueSize;
	private final Time time;

- Notice how it implements Future<RecordMetadata>

RecordAccumulator.appendNewBatch
  - ///// IMPORTANT /////: Changing some code from the original
  - DESCRIPTION: Append a new batch to the queue
  - Calls RecordAccumulator.tryAppend
  - If the append result is not null, just return it
    -- HOWEVER: The comment in the original code: "/ Somebody else found us a
    batch, return the one we waited for! Hopefully this doesn't happen often"
    won't be applicable in my case
  - If the append result is null, such as when the batch is full or there's no
    available batch in the queue, then create a new ProducerBatch. Then:
    -- Add the record to that newly created batch
    -- Add this batch to the queue
    -- Add this batch to the set of incomplete batches
    -- Return RecordAppendResult with:
  - batchIsFull: dq.size() > 1 || batch.isFull()
  - newBatchCreated: true
  - ******** NOTE *******: The result of appendNewBatch is the FINAL RETURN VALUE
    of RecordAccumulator.append (which was called by KafkaProducer.send)
  - NOTE: I should probably just call this something else. The main purpose
    based on my edits is that this attempts to add to a batch in the queue
    at least one batch exists in it and the last batch isn't full. Then if
    the last batch is full or we didn't have any batches in the first place,
    then we create a new batch

RecordAccumulator.tryAppend
  - ///// IMPORTANT /////: Changing some code from the original
  - This tries to append a record to the last ProducerBatch in the queue
  - First, get the last ProducerBatch
  - If there's no batch to begin with, return null
    -- RecordAccumulator.appendNewBatch can then create the batch where
    we can add the record to
  - If there is batch, try to append to it
    -- CODE: FutureRecordMetadata future = last.tryAppend(timestamp, key, value, headers, callback, nowMs);
    -- So we call ProducerBatch.tryAppend here
    -- If the batch is full, ProducerBatch.tryAppend returns null
  - We can then close this batch for record appends
  - Then we just return null so RecordAccumulator.appendNewBatch can then
    create the batch as already mentioned before
    -- If not full, just return RecordAppendResult with:
  - batchIsFull: deque.size() > 1 || last.isFull()
  - newBatchCreated: false

ProducerBatch.tryAppend
  - DESCRIPTION: Append the record to the current record set and return the
    relative offset within that record set
  - First, check if the batch has space.
  - If not, return null
  - If it does, add the record to the batch
    -- Create the FutureRecordMetadata
    -- Increase the recordCount for this ProducerBatch
    -- Then return the FutureRecordMetadata

IMPORTANT: I'm skipping a lot of metadata request things for now, but here are
   some useful links:
- private ClusterAndWaitTime waitOnMetadata
  https://github.com/apache/kafka/blob/trunk/clients/src/main/java/org/apache/kafka/clients/producer/KafkaProducer.java#L1214
- public class FetchMetadata (the type of metadata field in KafkaProducer)
  https://github.com/apache/kafka/blob/trunk/clients/src/main/java/org/apache/kafka/common/requests/FetchMetadata.java#L25
- Map<TopicPartition, FetchResponseData.PartitionData> fetch(FetchRequest.Builder fetchRequest)
  https://github.com/apache/kafka/blob/trunk/server/src/main/java/org/apache/kafka/server/LeaderEndPoint.java#L65
  - This is the function fetch() called inside waitOnMetadata
- FOR NOW: The producer has the metadata (since the topics and partitions are
  hardcoded anyway)

IMPORTANT: I'm also skipping a lot of the timing related things, except the
  ones I need like those related to linger.

SEE LONGER NOTES section for more details on step by step analysis of the
  actual Kafka code.
*/

/*
===============================================================================
============================    LONGER NOTES    ===============================
===============================================================================

FROM KafkaProducer: https://github.com/apache/kafka/blob/trunk/clients/src/main/java/org/apache/kafka/clients/producer/KafkaProducer.java
- NOTE: There's a lot of other useful info here, but I'll just include the ones
  I'll need for now
- EXAMPLE USAGE: https://kafka.apache.org/quickstart/
  -- Here, they're writing events to a specific topic
- The {@link #send(ProducerRecord) send()} method is asynchronous. When called,
  it adds the record to a buffer of pending record sends and immediately
  returns. This allows the producer to batch together individual records for
  efficiency.
- The producer maintains buffers of unsent records for each partition.
- These buffers are of a size specified by the <code>batch.size</code> config.
  Making this larger can result in more batching, but requires more memory
  (since we will generally have one of these buffers for each active partition)
  -- As per Kafka producer configuration documentation batch.size may be set to
     0 to explicitly disable batching which in practice actually means using a
	 batch size of 1.
- By default a buffer is available to send immediately even if there is
  additional unused space in the buffer. However if you want to reduce the
  number of requests you can set <code>linger.ms</code> to something greater
  than 0. This will instruct the producer to wait up to that number of
  milliseconds before sending a request in hope that more records will
  arrive to fill up the same batch.
  + This setting would add latency to our request waiting for more records to
    arrive if we didn't fill up the buffer.
  + Note that records that arrive close together in time will generally batch
    together even with <code>linger.ms=0</code>. So, under heavy load, batching
	will occur regardless of the linger configuration
  + However setting this to something larger than 0 can lead to fewer, more
    efficient requests when not under maximal load at the cost of a small
	amount of latency

public Future<RecordMetadata> send(ProducerRecord<K, V> record, Callback callback) { ...
- Asynchronously send a record to a topic and invoke the provided callback
  when the send has been acknowledged
- This method will return immediately (except for rare cases described below)
  once the record has been stored in the buffer of records waiting to be sent.
- This allows sending many records in parallel without blocking to wait for the
  response after each one
- Can block for the following cases:
  1) For the first record being sent to the cluster by this client for the
     given topic. In this case it will block for up to {@code max.block.ms}
	 milliseconds while waiting for topic's metadata if Kafka cluster is
	 unreachable
  2) Allocating a buffer if buffer pool doesn't have any free buffers.
- The result of the send is a {@link RecordMetadata} specifying the partition
  the record was sent to, the offset it was assigned and the timestamp of the
  record.
  -- If the producer is configured with acks = 0, the {@link RecordMetadata}
     will have offset = -1 because the producer does not wait for the
	 acknowledgement from the broker.
  -- If {@link org.apache.kafka.common.record.TimestampType#CREATE_TIME CreateTime}
     is used by the topic, the timestamp will be the user provided timestamp or
	 the record send time if the user did not specify a timestamp for the
	 record.
  -- If {@link org.apache.kafka.common.record.TimestampType#LOG_APPEND_TIME LogAppendTime}
     is used for the topic, the timestamp will be the Kafka broker local time
	 when the message is appended.
- Since the send call is asynchronous it returns a
  {@link java.util.concurrent.Future Future} for the {@link RecordMetadata}
  that will be assigned to this record.
  -- Invoking {@link java.util.concurrent.Future#get() get()} on this future
     will block until the associated request completes and then return the
	 metadata for the record or throw any exception that occurred while sending
	 the record
	 + In send from ConsoleProducer, get() is called if sync is true (from
	   opts.sync()). So opts.sync() is an option to make things synchronous
- @param record: The record to send. If the topic or the partition specified in
  it cannot be found in metadata within {@code max.block.ms}, the returned
  future will time out when retrieved
  + ME: So the broker must return metadata that matches the topic and
    partition specified in the record so they're in agreement to where
	the record has been added in the broker
- ----------------- CODE ------------------
1.) calls a ProducerInterceptor's onSend method
  https://github.com/apache/kafka/blob/trunk/clients/src/main/java/org/apache/kafka/clients/producer/ProducerInterceptor.java
  -- Purpose is to intercept the record, which can be potentially modified
     + This is the description from the KafkaProducer file
  -- onSend is called before key and value get serialized and partition is
     assigned (if partition is not specified in ProducerRecord).
	 + IMPORTANT: So the partition can be specified in the record when the
	   producer receives it or the producer can assign the partition if not
	   specified.
     + This method is allowed to modify the record, in which case, the new
	   record will be returned. The implication of modifying key/value is that
	   partition assignment (if not specified in ProducerRecord) will be done
	   based on modified key/value, not key/value from the client.
	   Consequently, key and value transformation done in onSend() needs to be
	   consistent: same key and value should mutate to the same (modified) key
	   and value. Otherwise, log compaction would not work as expected
2.) calls doSend(interceptedRecord, callback)
- -------------------------------------------

public class RecordAccumulator
- https://github.com/apache/kafka/blob/trunk/clients/src/main/java/org/apache/kafka/clients/producer/internals/RecordAccumulator.java#L69
- This class acts as a queue that accumulates records into
  {@link MemoryRecords} instances to be sent to the server.
  -- The accumulator uses a bounded amount of memory and append calls will
     block when that memory is exhausted, unless this behavior is explicitly
	   disabled.
  -- @param lingerMs An artificial delay time to add before declaring a records
     instance that isn't full ready for sending. This allows time for more
	   records to arrive. Setting a non-zero lingerMs will trade off some
     latency for potentially better throughput due to more batching (and hence
	   fewer, larger requests).

------------------ IMPORTANT (regarding sending records) -------------
private Future<RecordMetadata> doSend(ProducerRecord<K, V> record, Callback callback)
- Implementation of asynchronously send a record to a topic
- private ClusterAndWaitTime waitOnMetadata(String topic, Integer partition, long nowMs, long maxWaitMs)
  -- Wait for cluster metadata including partitions for the given topic to be
     available.
     + ME: Remember above in send. It mentions that send blocks: "For the first
	   record being sent to the cluster by this client for the given topic. In
	   this case it will block for up to {@code max.block.ms} milliseconds
	   while waiting for topic's metadata if Kafka cluster is unreachable"
  -- ME: Remember how the producer has metadata about the cluster and its
     topics/partitions? I think this is where it gets updated. It seems like
	 the producer also has a cluster metadata cache of some sort
- So it first gets metadata about the cluster
- Then it serializes the key and the value
- int partition = partition(record, serializedKey, serializedValue, cluster);
  -- Try to calculate partition, but note that after this call it can be
     RecordMetadata.UNKNOWN_PARTITION, which means that the RecordAccumulator
	 would pick a partition using built-in logic (which may take into account
	 broker load, the amount of data produced to each partition, etc.).
  -- private int partition(ProducerRecord<K, V> record, byte[] serializedKey, byte[] serializedValue, Cluster cluster)
  -- computes partition for given record.
     + if the record has partition returns the value otherwise
     + if custom partitioner is specified, call it to compute partition
     + otherwise try to calculate partition based on key.
     + If there is no key or key should be ignored return
       RecordMetadata.UNKNOWN_PARTITION to indicate any partition
     + can be used (the partition is then calculated by built-in
       partitioning logic).

- RecordAccumulator.RecordAppendResult result = accumulator.append
  -- Append the record to the accumulator.  Note, that the actual partition may
     be calculated there and can be accessed via appendCallbacks.topicPartition
  -- ME: The notes about RecordAccumulator are above
- The sender is woken up if the topic partition is either full or getting a
  new batch
---------------------------------

public void flush()
- Seems to be used only for transactions

public void close()
- Called in a constructor inside a catch branch

*******************************************************************************
************************			CODE ANALYSIS         ***************************
*******************************************************************************
When ConsoleProducer is started:
1.) Using command line args, creates ConsoleProducerOptions
2.) Creates an instance of a KafkaProducer, passing in the options to be used
    by the producer
3.) (Overly simplfied) Loop as long as there's input from the user in the
    terminal. In this loop, the KafkaProducer will keep sending any input
	given by the user in the terminal
NOTE: The quickstart example sends records to the specified topic in the
  options. (https://kafka.apache.org/quickstart/#step-4-write-some-events-into-the-topic)

send (KafkaProducer.send)
- Summarized high-level explanation:
  -- This is to send a record to a topic, then it runs the provided callback
     once the send has been acknowledged. Though as mentioned below, the send
	 is asynchronous
  -- Asynchronous: Will return once the record has been stored in the buffer
     of records waiting to be sent
  -- Can block on first record being sent by this client to the cluster for the
     given topic (which is done to get metadata about a topic and it blocks for
	 at most the specified amount in the max.block.ms setting)
	 + ME: To keep it simple for now, it needs info about the topic, its
	   partitions, which are the leader brokers, which brokers are in the
	   cluster, etc.
	 + ********* IMPORTANT ***********: Before records can be batched, the
	   producer needs metadata about the cluster first. So above, it
	   literally means that it could block on the FIRST RECORD being sent to
	   the cluster to get metadata.
  -- Can also block when allocating a buffer if the buffer pool has no free
     buffers
  -- ME: So once the record has been put in the queue of ProducerBatch (in
     RecordAccumulator's append method), then the send can return. Obviously,
	 the rest of the code after the call to RecordAccumulator's tryAppend
	 method (inside it's append method) which adds to the queue needs to run,
	 just like the rest of the code in doSend (which send calls).
	 + The loose meaning of "Will return once the record has been stored in the
	   buffer of records waiting to be sent" is simply to say that once the
	   record is in the queue of batches, send's responsibility is done
	   and something else (the Sender) is responsible for actually sending
	   the batches.
  -- Result is RecordMetadata which contains the partition the record was sent
     to, the offset it was assigned, and the timestamp of the record
  -- doSend: Send actually calls doSend to do the actual send
- ----------------- Steps for doSend --------------------
- (Once again, a summarized and overly simplified explanation)
- First, append callback takes care of:
  -- call interceptors and user callback on completion
  -- remember partition that is calculated in RecordAccumulator.append
- Second, get metadata about the cluster, especially the topic and partition (if
  specified) we want metadata about
  -- Note that cluster refers to all brokers, not just the brokers in which
     the topic's partitions (and their replicas) belong in
     + SIDE NOTE: Also, replication factor = 3 does NOT mean that we only have
       3 brokers. It simply means that partitions (for all topics) are
	   replicated in 3 of those brokers.
  -- REMINDER: As mentioned above, the send can block on the first record being
     sent to the cluster.
  -- IMPORTANT: There's also a metadata cache
     + We just get the metadata here if we have it
	 + WARNING: This could be stale
- Serialize the key and the value
- Calculate the partition
  + Just return the partition specified in the record if it has one
  + Use custom partitioner to compute the partition if one was specified
  + Otherwise, calculate partition based on key using the BuiltInPartitioner
  + Finally, if there's no key or the key should be ignored, just
    return RecordMetadata.UNKNOWN_PARTITION
	* IMPORTANT: In this case, the RecordAccumulator would pick a partition
	  using built-in logic.
	  ** In RecordAccumulator.append: Partition is picked based on the broker
	  availability and performance.
- Ensure the serialized record is valid size. So not larger than
  ProducerConfig.MAX_REQUEST_SIZE_CONFIG and
  ProducerConfig.BUFFER_MEMORY_CONFIG (not larger than the configured buffer
  size)
- Append the record to the accumulator
  -- Again, partition may actually be calculated in the RecordAccumulator
  -- ME: This is where we add the record to a batch.
  -- NOTE: The RecordAccumulator isn't the batch itself.
  -- In the KafkaProducer's constructor, it gives the RecordAccumulator's
     constructor a BufferPool
  -- ********IMPORTANT********: Relationship between KafkaProducer,
     RecordAccumulator, BufferPool, ByteBuffer, and ProducerBatch:
     + KafkaProducer creates a RecordAccumulator, and gives it a BufferPool.
     + The RecordAccumulator can allocate a ByteBuffer from that pool when it
	   needs to create a new ProducerBatch
- ********IMPORTANT********: RecordAccumulator
  -- Acts as a queue that accumulates records into MemoryRecords instances to
     be sent to the server.
	 + Uses a bounded amount of memory and blocks when memory is full
  -- When the instance was created in the KafkaProducer constructor, the
     following were passed:
	 + batchSize
	 + compression codec
	 + lingerMs
	 + retryBackoffMs
	 + retryBackoffMaxMs
	 + deliveryTimeoutMs
	 + partitionerConfig (Or not, if default partitionerConfig is wanted)
	 + bufferPool
	   * public class BufferPool: A pool of ByteBuffers kept under a given
	     memory limit
		 ** private final Deque<ByteBuffer> free: Seems like just a deque of
		    available ByteBuffers
		 ** private final long totalMemory: The maximum amount of memory that
		    this buffer pool can allocate
		 ** private long nonPooledAvailableMemory
  -- It also has the following fields:
     * private final ConcurrentMap<String topic, TopicInfo> topicInfoMap
	   + Basically a map that contains info about a topic
	     ** SEE BELOW for TopicInfo
	 * private final IncompleteBatches incomplete
     * private final BufferPool free (already mentioned above)
  -- ++++++++++++++ RecordAccumulator.append ++++++++++++++++
  -- NOTE: Multithreading/concurrency is involved here
     + Which is why we perform the operations inside ****** Loop ****** inside
	   a loop to deal with race conditions
  -- Description: Add a record to the accumulator, return the append result
     + Append result will contain the metadata and a flag if the appended
	   batch is full or a new batch has been created
  -- First get info about the topic
     + A TopicInfo has a ConcurrentMap<Integer partition, Deque<ProducerBatch>> batches
       * Notice how the map contains the queue of batches to be sent for each
	     partition
  -- ByteBuffer buffer = null
     + If null later, this will be allocated.
	 + This will be used when appending a new batch
	 + ME: I initially thought that this buffer is used to store records in
	   the queue of batches. IT IS NOT
  -- ********** Loop **********
  -- Pick a partition using BuiltInPartitioner.StickyPartitionInfo if
     partition == RecordMetadata.UNKNOWN_PARTITION
  -- Deque<ProducerBatch> dq = topicInfo.batches.computeIfAbsent(effectivePartition, k -> new ArrayDeque<>());
     + Comment in code: "Check if we have an in-progress batch"
	   * ME: Based on code later (in tryAppend), this in-progress batch could
	     be full. But we won't know until we call tryAppend
	 + Remember that topicInfo has a map of the queue of batches for each
	   partition
	 + computeIfAbsent is a ConcurrentMap method
	   https://docs.oracle.com/javase/8/docs/api/java/util/concurrent/ConcurrentMap.html#computeIfAbsent-K-java.util.function.Function-
	 + Basically, we create the queue of batches for that partition if
	   it doesn't exist
  -- RecordAppendResult appendResult = tryAppend(timestamp, key, value, headers, callbacks, dq, nowMs);
     + We try to append to a ProducerBatch in the queue of batches for this
	   partition
  -- RecordAccumulator.tryAppend:
     + If the ProducerBatch is full, we return null and a new batch is created.
	   The batch is also closed for record appends.
	 + ProducerBatch last = deque.peekLast();
	   * if last is null (peekLast returned null), it means the queue is empty
	     ** tryAppend ends up returning null just like if the batch is full
		    ++ However, nothing is closed since there's no batch in the
			   first place
	     ** ME: Later (in appendNewBatch), we actually create a new batch to
		    put the record into. So basically it's the same behavior if the
			batch is full or there's no existing batch in the queue.
			++ Only difference is if a batch is full, we close it for record
			   appends
	   * If not, we try to append to the last batch in the queue in
	     FutureRecordMetadata future = last.tryAppend(timestamp, key, value, headers, callback, nowMs)
	 + ProducerBatch.tryAppend:
	   * Append the record to the current record set and return the relative
	     offset within that record set
	   * If this batch is full, it returns null.
	   * Also mentions something about the batch possibly needing to be split
	     (I'LL SKIP THIS FOR NOW)
	   * Returns a FutureRecordMetadata
	   * ******* IMPORTANT *********: A ProducerBatch actually uses
	     MemoryRecordsBuilder to "store" records
     + If the returned FutureRecordMetadata is null, then the batch is closed
	   for record appends (last.closeForRecordAppends()) since it's full
	 + Otherwise, we create a RecordAppendResult instance using the returned
	   FutureRecordMetadata
	   * When calling the constructor, we pass in:
	     ** newBatchCreated is false
		 ** batchIsFull is true based on deque.size() > 1 || last.isFull()
	   * HOWEVER: It's weird how it uses deque.size() > 1 || last.isFull() as
	     the boolean condition to determine if the batch is full. Why does
		 deque.size() > 1 indicate that the batch is full?
		 ** From ChatGPT (DOUBLE CHECK THIS)
		    ++ deque.size() > 1 means there is already more than one batch
			   queued for this topic-partition, which means there is backlog,
			   so the partition should be treated as “ready enough” even if
			   the current tail batch itself is not full. As a result, we can
			   wake the sender
	 + IMPORTANT: It says above that a new batch is created if the batch is
	   full. However, I don't see it anywhere in RecordAccumulator.tryAppend
	   and ProducerBatch.tryAppend
	   * It's actually in appendNewBatch later
  -- At this point RecordAccumulator.tryAppend has returned a
     RecordAppendResult (could be null).
	 + REMINDER: RecordAppendResult contains the future(metadata), batchIsFull,
	   newBatchCreated, and appendedBytes
	 + Based on the code, only two cases where null could happen:
	   1.) Batch is full    OR    2.) queue was empty
	 + If the append result is null, then the switch is disabled if there are
	   any incomplete batches
	   (GONNA SKIP what the switch means for now)
  -- buffer = free.allocate(size, maxTimeToBlock)
     + If buffer is null, allocate a new {size} byte message buffer for
       topic {topic} partition {effectivePartition} with remaining timeout
	   {maxTimeToBlock}ms
	 + This is used below in appendNewBatch
  -- ******** CHECKPOINT *********
     + At this point, we have added the record to the queue of batches if
	   the last batch isn't full
	 + HOWEVER, if it's full or there's no available batch in the queue to
	   begin with, then we haven't added it to the queue of batches for that
	   partition yet
  -- RecordAppendResult appendResult = appendNewBatch(topic, effectivePartition, dq, timestamp, key, value, headers, callbacks, buffer, nowMs);
     + ////// IMPORTANT SIDE NOTE /////: NOT FROM appendNewBatch code, but from
	   earlier readings:
	   * One batch would be for one partition in a topic, where the producer
	     can send multiple batches that could be for different partitions
		 (which could be from different topics)
	 + This appends a new batch to the queue of batches for the specified
	   partition
	 + We pass in the allocated buffer and dq (the Deque<ProducerBatch>)
	 + Calls RecordAccumulator.tryAppend again
	   * If the last batch in deque is full or there's no batch in the queue
	     to begin with (so RecordAccumulator.tryAppend returns null), then its
		 obvious that we need to add a new batch to the queue
	   * ME: HOWEVER, if there's a batch and it's not full, then don't we end
	     up adding the record twice to the last batch?
	     ** The comment says: "Somebody else found us a batch, return the one we
	        waited for! Hopefully this doesn't happen often..."
		    ++ ME: What exactly does this mean ???
	 + If the result of the tryAppend is not null, just return the
	   RecordAppendResult
	 + If the result of the tryAppend is null, a new ProducerBatch is created
	   * MemoryRecordsBuilder recordsBuilder = recordsBuilder(buffer)
	     ** So we use the allocated buffer from RecordAccumulator.append's
		    code as the new ProducerBatch's MemoryRecordsBuilder (used to
			store records)
	   * Create a new ProducerBatch
	   * Add the record to this new batch
	   * Add this batch to dq (the Deque<ProducerBatch>) as the last batch
	   * Add this batch to the incomplete batches
	   * Return a RecordAppendResult with newBatchCreated set to true
	     ** ANALYSIS of batchIsFull: dq.size() > 1 || batch.isFull()
		    ++ If there was no batch in the queue to begin with, then the very
			   first batch for the queue would be created here
		    ++ dq.size() would be == 1
			++ Normally, that batch would also not be full.
			    - UNLESS the batch size can only hold one record (based on
				  bytes where that one record is enough to hit the bytes
				  limit)
			++ THEREFORE, the Sender would not be woken up in this case
  -- RecordAccumulator.append <RETURNS> a RecordAppendResult (from appendNewBatch)
     + The append result will contain the future metadata, and flag for whether
	   the appended batch is full or a new batch is created
  -- *********** end Loop ***************
  -- FINALLY, deallocate the buffer
  -- +++++++++++++++ end RecordAccumulator.append ++++++++++++++++++
- ********IMPORTANT********: After the call to RecordAccumulator.append, then
  if batch is full or a new batch was created, wake up the Sender
  -- Sender: The background thread that handles the sending of produce requests
     to the Kafka cluster. This thread makes metadata requests to renew its
	 view of the cluster and then sends produce requests to the appropriate
	 nodes.
	 + https://github.com/apache/kafka/blob/trunk/clients/src/main/java/org/apache/kafka/clients/producer/internals/Sender.java#L79
  -- ME: If a new batch is created, but it's the very first batch in the queue,
     woudln't it not make sense to wake up the sender since we don't want to
	 send that one and only + incomplete batch yet?
- LASTLY, return the Future<RecordMetadata>
- ------------ end of Steps for doSend -----------------

//////////////////////////////////////////
So far, I have 2 things I'm unsure about:
1.) Inside appendNewBatch, we call RecordAccumulator.tryAppend
    - But if there's a batch and it's not full, then don't we end up adding the
	  record twice to the last batch?
	- The comment also says: "Somebody else found us a batch, return the one we
	  waited for! Hopefully this doesn't happen often..."
	- From ChatGPT (DOUBLE CHECK):
	  + The second tryAppend(...) inside appendNewBatch(...) is there because
	    Kafka may have released the deque lock while allocating memory from
		the buffer pool
	  + During that gap, another producer thread might append to the same
	    topic-partition and create a usable batch. So Kafka checks again
	  + ME: So I was right about multithreading/concurrency/locks being at play
	    here. I remember having to order my code a certain way from CSE 130
		when dealing with multiple threads
2.) In doSend's code (in KafkaProducer), if a new batch is created, but it's
    the very first batch in the queue, wouldn't it not make sense to wake up
	the sender since we don't want to send that one and only + incomplete
	batch yet?
	- This situation would happen in RecordAccumulator.tryAppend's code, where
	  ProducerBatch last = deque.peekLast(). tryAppend ends up returning null
	  when last is null
    - From ChatGPT (DOUBLE CHECK THIS in Sender's code):
	  + Even if the Sender was woken up, it's readiness logic could decide if
	    the batch is ready. So even if we end up in the situation above, we
		won't end up sending something prematurely. NOTE: This DOES NOT mean
		that incomplete batches aren't allowed to be sent. Remember that there
		are also other options such as linger.ms, etc.
//////////////////////////////////////////

*******************************************************************************
*****************		end of SUMMARY + new info      ************************
*******************************************************************************

*/
