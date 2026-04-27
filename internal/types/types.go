package types

// Uppercase name means exported
type ProducerRequest struct {
	Acks      string
	TimeoutMs int
	Batches   []Batch
}

type Batch struct {
	Topic     string
	Partition int
	Events    []Event
}

type Event struct {
	// Key       string // TODO: Add keys later
	Value string // TODO: Use []byte later
	// Timestamp int64 // TODO: Add timestamp later
}
