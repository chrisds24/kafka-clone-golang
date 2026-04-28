package producer

/*
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

public Future<RecordMetadata> send(ProducerRecord<K, V> record) { ...
- Asynchronously send a record to a topic
- Just calls the send function below with null callback

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
- private boolean partitionChanged
  -- Check if partition concurrently changed, or we need to complete previously
     disabled partition change
  -- @return 'true' if partition changed and we need to get new partition info
     and retry, 'false' otherwise
	 + ME: Notice how there's a waitOnMetadata function in the
	   KafkaProducer. I think that one is more for requesting metadata on the
       first send to the cluster.
- public RecordAppendResult append
  -- Add a record to the accumulator, return the append result
  -- The append result will contain the future metadata, and flag for whether
     the appended batch is full or a new batch is created
  -- public synchronized SharePartition computeIfAbsent
     + Computes the value for the given key if it is not already present in
	   the cache. Method also updates the group map with the topic-partition
	   for the group id.
     + @param partitionKey The key to compute the value for.
	 + ME: This looks like where the partition a record should go to gets
	   calculated if a partitionKey is specified?
	   * NOT SURE THOUGH, especially since it returns a SharePartition
  -- public class BuiltInPartitioner
     + The class keeps track of various bookkeeping information required for
	   adaptive sticky partitioning (described in detail in KIP-794). There is
	   one partitioner object per topic
	 + ME: So this is just to decide which partition a record should go to for
	   a topic
  -- IMPORTANT: SEE BELOW about how KafkaProducer's doSend calculates the
     partition, but mentions that it might happen in RecordAccumulator if
	 the partition is still RecordMetadata.UNKNOWN_PARTITION
  -- ByteBuffer buffer seems to be where we put the records in (as bytes)
     + NOTE: Look at public final class ProducerBatch below
  -- If the message doesn't have any partition affinity, so we pick a partition
     based on the broker availability and performance.
	 + final BuiltInPartitioner.StickyPartitionInfo partitionInfo
	   * ME: Once again, it uses sticky partitioning
  -- Deque<ProducerBatch> dq
     + There's a synchronized (dq) { ... } code section under this, so that
	   code uses multithreading/concurrency
     + public final class ProducerBatch
       * A batch of records that is or will be sent
	   * This class is not thread safe and external synchronization must be used
	     when modifying it
       * private boolean inflight: Tracks if the batch has been sent to the
	     NetworkClient
	   * public FutureRecordMetadata tryAppend
	     ** Append the record to the current record set and return the relative
		    offset within that record set
		 ** Calls this.recordsBuilder.append(timestamp, key, value, headers)
		    ++ public class MemoryRecordsBuilder (ALSO IMPORTANT)
			++ This class is used to write new log data in memory, i.e. this
			   is the write path for {@link MemoryRecords}.
			++ It transparently handles compression and exposes methods for
			   appending new records, possibly with message format conversion
  -- RecordAppendResult appendResult = tryAppend(timestamp, key, value, headers, callbacks, dq, nowMs);
  -- private RecordAppendResult tryAppend
     + Try to append to a ProducerBatch
     + If it is full, we return null and a new batch is created. We also close
	   the batch for record appends to free up resources like compression
	   buffers. The batch will be fully closed (ie. the record batch headers
	   will be written and memory records built) in one of the following cases
	   (whichever comes first): right before send, if it is expired, or when
	   the producer is closed
	 + RecordAccumulator append method calls this
	 + Then this does    ProducerBatch last = deque.peekLast();
	   * Notice above how a ProducerBatch also has a tryAppend method
  -- private RecordAppendResult appendNewBatch
     + Append a new batch to the queue (***** IMPORTANT ******)
	 + ME: So what exactly is the relationship between a batch and a queue?
	   Why do we need a queue instead of just a batch?
	 + In the code here, once the batch is added to the queue, a new batch
	   is created
  -- Based on the code, seems like an append to the ProducerBatch is
     attempted, then that ProducerBatch is appended to the queue

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
*****************			SUMMARY + new info         ************************
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
*/
