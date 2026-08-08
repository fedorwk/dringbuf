package dringbuf

// RingBuffer is a single-array circular buffer.
type RingBuffer[T any] struct {
	buf  []T
	head int
	len  int
	cap  int
}

// NewRingBuffer creates a RingBuffer with the given capacity.
func NewRingBuffer[T any](size int) *RingBuffer[T] {
	return &RingBuffer[T]{
		buf: make([]T, size),
		cap: size,
	}
}

// Append adds a new element to the buffer. If the buffer is already full, the oldest element is overwritten.
func (b *RingBuffer[T]) Append(elem T) {
	idx := (b.head + b.len) % b.cap
	b.buf[idx] = elem

	if b.len < b.cap {
		b.len++
	} else {
		b.head = (b.head + 1) % b.cap
	}
}

// Len returns the current number of elements stored in the buffer.
func (b *RingBuffer[T]) Len() int {
	return b.len
}

// Cap returns the maximum capacity of the buffer (the total number of elements it can hold).
func (b *RingBuffer[T]) Cap() int {
	return b.cap
}

// At retrieves the element at the specified index relative to the logical start of the buffer.
func (b *RingBuffer[T]) At(idx int) T {
	if idx >= b.cap {
		panic("idx out of buffer size")
	}
	return b.buf[(b.head+idx)%b.cap]
}

// Last returns the last n most recently appended elements in order.
func (b *RingBuffer[T]) Last(n int) []T {
	if n > b.cap {
		panic("n out of buffer size")
	}
	if n > b.len {
		n = b.len
	}

	start := (b.head + b.len - n) % b.cap
	res := make([]T, n)
	for i := 0; i < n; i++ {
		res[i] = b.buf[(start+i)%b.cap]
	}
	return res
}

// Clear removes all elements from the buffer and resets it to an empty state.
func (b *RingBuffer[T]) Clear() {
	var zero T
	for i := range b.buf {
		b.buf[i] = zero
	}
	b.head = 0
	b.len = 0
}
