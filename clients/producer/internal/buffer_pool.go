package internal

type BufferPool struct {
	// NOTE: For now, I'm not using byte buffers. I'm only simulating the idea
	//   of limiting/managing memory usage.
	// - Just using number of records for now to keep things simple.
	MaxUnsentRecords   int64
	TotalUnsentRecords int64
	AvailableMemory    int64
}
