package common

// In Go, it's better to put helper types in one file
// However, if it's a big component like a RecordAccumulator, then it's better
//   to put it in its own file.

type ProducerConfig struct {
	LingerMs  int
	BatchSize int
	// REPLACE MaxUnsentRecords later once record serialization into bytes
	//   is introduced
	MaxUnsentRecords int64
	// BufferMemory int
	Acks string
}

// TODO: "In Go, should I make constructors enforce default values?"
//   -- "Is this idiomatic in Go if a type has a field that need to be a
//       certain value?"
//   -- "Would it be better to just keep fields unexported in these cases?"
//   -- "Won't I just end up doing it the Java way where I have getters and
//       setters?"

// Regarding value fields vs. pointer fields since ProducerRecord is optional
//
//   - Pointers are usually used for optional fields
//
//   - HOWEVER, there's a catch when it comes to a ProducerRecord
//     -- When passing a pointer to a function, the memory allocated
//     to what the pointer points to is freed once that function
//     returns as long as nothing references that pointer either
//     anywhere within what calls the function or something within
//     the function call (Ex. it gets added to a data structure
//     that persists beyond the function call)
//     -- This is where it gets tricky with ProducerRecord, since it
//     gets added to the RecordAccumulator (which uses a buffer)
//     -- ProducerRecord itself is passed as a value to send, but if pointers
//     are used for Partition and Key, then the memory used for them would
//     remain on the heap
//     -- So even though send is asynchronous and that the method will
//     return immediately (except for rare cases described...SEE KafkaProducer
//     in the repo) once the record has been stored in the buffer of records
//     waiting to be sent, the memory allocated for those two fields would
//     remain on the heap.
//     -- BUT: Since I'm serializing the record anyway and not putting the
//     ProducerRecord itself in the RecordAccumulator, the issue mentioned
//     above shouldn't really happen.
//     -- HOWEVER, I'm still putting a lot of pressue in the heap/GC
//
//   - SOLUTION: Use value fields for Key and Partition. Then use a hasKey
//     existence field for Key and use a negative value (-1) to specify
//     that the Partition is unspecified
//     -- The whole ProducerRecord still "outlives" send since it gets put in
//     the buffer (but serialized), which is the same as using pointers
//     -- However, this way puts less pressure on the heap before the
//     serialization happens
//     -- This method also makes sense since ProducerRecord (although a
//     struct) isn't really an object that is meant to be changed (SEE
//     https://preslav.me/2026/01/08/golang-structs-vs-pointers-pointer-first/)
//     -- Pointers make sense when the struct is either too large or it is
//     meant to be mutable
//
//   - Regarding memory used by the ProducerRecord struct (64-bit system)
//     -- Pointer Key and Partition: 8 bytes (int pointer) + 8 (string
//     pointer) + 8 bytes (actual int, if exists) + 16 bytes (actual string,
//     if exists) = 40 bytes...then add allocation overhead + GC tracking
//     -- Value Key and value Partition (-1 to specifify non-existence)
//     16 (Key) + 1 (hasKey) + 8 (Partition) = 25 bytes...then add
//     padding/alignment
//     -- So the second way has the advantage when the Partition and Key are
//     specified
//
//   - Useful source regarding heap allocation and garbage collection in Go:
//     -- https://alexanderobregon.substack.com/p/go-memory-zeroing-rules
//
//   - NOTE THAT Kafka actually allows empty keys ("") so I can't just
//     use "" to signify lack of Key. -1 does work for Partition since
//     partitions can't be negative and Kafka uses 0-based indexing for
//     partitions.
//
// ORIGINAL SOURCE CODE: https://github.com/apache/kafka/blob/trunk/clients/src/main/java/org/apache/kafka/clients/producer/ProducerRecord.java#L49
type ProducerRecord struct {
	Topic     string // Required
	Partition int    // -1 means partition not specified
	Key       string // Optional. "" (empty string) keys are allowed
	hasKey    bool   // Flag that specifies if Key is specified
	Value     string // Required
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

// My own temporary custom type that represents a record that will be sent
// to the broker.
// Refer to internal/broker/handler.go for the structure of the JSON that
// will be sent to the broker.
type KeyValRecord struct {
	Key   string
	Value string
}
