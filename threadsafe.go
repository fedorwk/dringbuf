package dringbuf

import "sync"

// ThreadSafeRingBuffer serializes access to an underlying ring buffer of type
// B. Reads share a read lock; Append and Clear take the write lock.
type ThreadSafeRingBuffer[B Buffer[T], T any] struct {
	mu  sync.RWMutex
	buf B
}

// SyncRingBuffer is a thread-safe RingBuffer.
type SyncRingBuffer[T any] = ThreadSafeRingBuffer[*RingBuffer[T], T]

// SyncDoubleRingBuffer is a thread-safe DoubleRingBuffer.
type SyncDoubleRingBuffer[T any] = ThreadSafeRingBuffer[*DoubleRingBuffer[T], T]

// NewThreadSafeRingBuffer creates a thread-safe wrapper around a RingBuffer.
func NewThreadSafeRingBuffer[T any](size int) *SyncRingBuffer[T] {
	return &SyncRingBuffer[T]{
		buf: NewRingBuffer[T](size),
	}
}

// NewThreadSafeDoubleRingBuffer creates a thread-safe wrapper around a DoubleRingBuffer.
func NewThreadSafeDoubleRingBuffer[T any](size int) *SyncDoubleRingBuffer[T] {
	return &SyncDoubleRingBuffer[T]{
		buf: NewDoubleRingBuffer[T](size),
	}
}

// View calls fn with the last n elements while holding the read lock, so
// writers wait until fn returns. For a DoubleRingBuffer the slice aliases
// internal storage: it must not be modified or retained after fn returns. For a
// RingBuffer it is a copy. It panics if n is negative or greater than Cap.
func (b *ThreadSafeRingBuffer[B, T]) View(n int, fn func([]T)) {
	b.checkTail(n)
	b.mu.RLock()
	defer b.mu.RUnlock()
	fn(b.buf.Tail(n))
}

// Append adds a new element to the buffer. If the buffer is already full, the oldest element is overwritten.
func (b *ThreadSafeRingBuffer[B, T]) Append(elem T) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buf.Append(elem)
}

// Len returns the current number of elements stored in the buffer.
func (b *ThreadSafeRingBuffer[B, T]) Len() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.buf.Len()
}

// Cap returns the maximum capacity of the buffer (the total number of elements
// it can hold). Capacity never changes, so it takes no lock.
func (b *ThreadSafeRingBuffer[B, T]) Cap() int {
	return b.buf.Cap()
}

// At returns the element at idx relative to the logical start of the buffer,
// where At(0) is the oldest element and At(Len()-1) is the most recent. It
// panics if idx is negative or not less than Len.
func (b *ThreadSafeRingBuffer[B, T]) At(idx int) T {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.buf.At(idx)
}

// Get returns the element at idx and true, or the zero value and false if idx
// is out of range. The bounds check and read happen under a single lock.
func (b *ThreadSafeRingBuffer[B, T]) Get(idx int) (T, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.buf.Get(idx)
}

// Last returns the most recently appended element and true, or the zero value
// and false when the buffer is empty.
func (b *ThreadSafeRingBuffer[B, T]) Last() (T, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.buf.Last()
}

// Tail returns a copy of the last n elements. To read the data in place use
// View; to copy into a reused slice use AppendTail.
func (b *ThreadSafeRingBuffer[B, T]) Tail(n int) []T {
	b.checkTail(n)
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.buf.AppendTail(make([]T, 0, min(n, b.buf.Len())), n)
}

// AppendTail appends a copy of the last n elements to dst and returns the
// extended slice. It does not allocate when dst has enough spare capacity.
func (b *ThreadSafeRingBuffer[B, T]) AppendTail(dst []T, n int) []T {
	b.checkTail(n)
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.buf.AppendTail(dst, n)
}

// Clear removes all elements from the buffer and resets it to an empty state.
func (b *ThreadSafeRingBuffer[B, T]) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buf.Clear()
}

// checkTail panics on an out-of-range n before any lock is taken.
func (b *ThreadSafeRingBuffer[B, T]) checkTail(n int) {
	if n < 0 || n > b.buf.Cap() {
		panic("dringbuf: n out of buffer size")
	}
}
