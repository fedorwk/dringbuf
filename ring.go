package dringbuf

type RingBuffer[T any] interface {
	// Append adds a new element to the buffer. If the buffer is already full, the oldest element is overwritten
	Append(elem T)
	// Len returns the current number of elements stored in the buffer
	Len() int
	// Cap returns the maximum capacity of the buffer (the total number of elements it can hold)
	Cap() int
	// At retrieves the element at the specified index relative to the logical start of the buffer
	At(idx int) T
	// Last returns the last n most recently appended elements in order
	Last(n int) []T
	// Clear removes all elements from the buffer and resets it to an empty state
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
