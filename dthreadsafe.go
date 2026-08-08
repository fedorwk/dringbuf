package dringbuf

import "sync"

// Release unlocks a buffer previously borrowed via ThreadSafeRingBuffer.Borrow.
type Release func()

// innerOps is the set of operations ThreadSafeRingBuffer needs from the
// underlying ring buffer. Instantiations with a concrete B produce direct calls.
type innerOps[T any] interface {
	Append(elem T)
	Len() int
	Cap() int
	At(idx int) T
	Last(n int) []T
	Clear()
}

// ThreadSafeRingBuffer serializes access to an underlying ring buffer of type B.
type ThreadSafeRingBuffer[B innerOps[T], T any] struct {
	mu  sync.Mutex
	buf B
}

// NewThreadSafeRingBuffer creates a thread-safe wrapper around a RingBuffer.
func NewThreadSafeRingBuffer[T any](size int) *ThreadSafeRingBuffer[*RingBuffer[T], T] {
	return &ThreadSafeRingBuffer[*RingBuffer[T], T]{
		buf: NewRingBuffer[T](size),
	}
}

// NewThreadSafeDRingBuffer creates a thread-safe wrapper around a DRingBuffer.
func NewThreadSafeDRingBuffer[T any](size int) *ThreadSafeRingBuffer[*DRingBuffer[T], T] {
	return &ThreadSafeRingBuffer[*DRingBuffer[T], T]{
		buf: NewDRingBuffer[T](size),
	}
}

// Borrow returns underlying data with n last elements.
// Locks buffer for reading until `release` call.
func (b *ThreadSafeRingBuffer[B, T]) Borrow(n int) ([]T, Release) {
	b.mu.Lock()
	release := func() {
		b.mu.Unlock()
	}
	return b.buf.Last(n), release
}

// Append adds a new element to the buffer. If the buffer is already full, the oldest element is overwritten.
func (b *ThreadSafeRingBuffer[B, T]) Append(elem T) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buf.Append(elem)
}

// Len returns the current number of elements stored in the buffer.
func (b *ThreadSafeRingBuffer[B, T]) Len() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Len()
}

// Cap returns the maximum capacity of the buffer (the total number of elements it can hold).
func (b *ThreadSafeRingBuffer[B, T]) Cap() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Cap()
}

// At retrieves the element at the specified index relative to the logical start of the buffer.
func (b *ThreadSafeRingBuffer[B, T]) At(idx int) T {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.At(idx)
}

// Last returns copy of underlying data
// To take advantage of threadsafe implementation use Borrow method instead
func (b *ThreadSafeRingBuffer[B, T]) Last(n int) []T {
	b.mu.Lock()
	defer b.mu.Unlock()

	capacity := b.buf.Cap()
	if n > capacity {
		panic("n out of buffer size")
	}

	length := b.buf.Len()
	if length < n {
		n = length
	}

	res := make([]T, n)
	copy(res, b.buf.Last(n))
	return res
}

// Clear removes all elements from the buffer and resets it to an empty state.
func (b *ThreadSafeRingBuffer[B, T]) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buf.Clear()
}
