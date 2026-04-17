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
	Key       string
	Value     string // TODO: Use []byte later
	Timestamp int64
}
