package internal

import (
	"github.com/chrisds24/kafka-clone-golang/clients"
)

// ProducerMetadata: https://github.com/apache/kafka/blob/trunk/clients/src/main/java/org/apache/kafka/clients/producer/internals/ProducerMetadata.java#L37
//   - ProducerMetadata actually extends Metadata: https://github.com/apache/kafka/blob/trunk/clients/src/main/java/org/apache/kafka/clients/Metadata.java#L67
//   - Struct embedding: https://gobyexample.com/struct-embedding
//     -- This is how to loosely extend another type in Go
//     -- This allows ProducerMetadata to use Metadata fields directly
//     -- In my case for now, I can call fetch() which is defined in
//     Metadata
type ProducerMetadata struct {
	// Kafka actually has
	//   private final Map<String, Long> topics = new HashMap<>(), since it
	//   stores expiry time, which I won't do for now
	topics map[string]bool
	*clients.Metadata
}

func NewProducerMetadata() *ProducerMetadata {
	return &ProducerMetadata{
		topics:   make(map[string]bool),
		Metadata: clients.NewMetadata()}
}

// Need a method to add to topics, check data inside it, etc.
