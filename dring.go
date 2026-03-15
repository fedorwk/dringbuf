package dringbuf

type RingBuffer[T any] interface {
	Append(elem T)
	Len() int
	Size() int
	At(idx int) T
	Last(n int) []T
	Clear()
}

// Threadsafe version of ring-buffer
// Implementation MUST return copy when `Last` called and lock buffer for reading on Borrow until `release()` call
type SyncRingBuffer[T any] interface {
	RingBuffer[T]
	// Returns last `n` buffer entries
	// Implementation MUST lock buffer for reading until `release()` call
	// Caller SHOULDN'T modify returned slice as it will likely break internal representation consistency
	Borrow(n int) ([]T, release)
}
