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
	Get(idx int) (T, bool)
	Last() (T, bool)
	Tail(n int) []T
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

// Borrow returns the last n elements and locks the buffer until release is
// called. For a RingBuffer the returned slice is a copy; for a DRingBuffer it
// aliases internal storage and must be treated as read-only and released before
// the next Append. The buffer stays locked for writing until release.
func (b *ThreadSafeRingBuffer[B, T]) Borrow(n int) ([]T, Release) {
	b.mu.Lock()
	release := func() {
		b.mu.Unlock()
	}
	return b.buf.Tail(n), release
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

// At returns the element at idx relative to the logical start of the buffer,
// where At(0) is the oldest element and At(Len()-1) is the most recent. It
// panics if idx is negative or not less than Len.
func (b *ThreadSafeRingBuffer[B, T]) At(idx int) T {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.At(idx)
}

// Get returns the element at idx and true, or the zero value and false if idx
// is out of range. The bounds check and read happen under a single lock.
func (b *ThreadSafeRingBuffer[B, T]) Get(idx int) (T, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Get(idx)
}

// Last returns the most recently appended element and true, or the zero value
// and false when the buffer is empty.
func (b *ThreadSafeRingBuffer[B, T]) Last() (T, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Last()
}

// Tail returns a copy of the last n elements. To avoid the copy and read the
// underlying data in place, use Borrow instead.
func (b *ThreadSafeRingBuffer[B, T]) Tail(n int) []T {
	b.mu.Lock()
	defer b.mu.Unlock()

	capacity := b.buf.Cap()
	if n < 0 || n > capacity {
		panic("dringbuf: n out of buffer size")
	}

	length := b.buf.Len()
	if length < n {
		n = length
	}

	res := make([]T, n)
	copy(res, b.buf.Tail(n))
	return res
}

// Clear removes all elements from the buffer and resets it to an empty state.
func (b *ThreadSafeRingBuffer[B, T]) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buf.Clear()
}
