package dringbuf

// RingBuffer is a single-array circular buffer.
type RingBuffer[T any] struct {
	buf  []T
	head int
	len  int
}

// NewRingBuffer creates a RingBuffer with the given capacity. It panics if size
// is not positive.
func NewRingBuffer[T any](size int) *RingBuffer[T] {
	if size <= 0 {
		panic("dringbuf: size must be positive")
	}
	return &RingBuffer[T]{
		buf: make([]T, size),
	}
}

// Append adds a new element to the buffer. If the buffer is already full, the oldest element is overwritten.
func (b *RingBuffer[T]) Append(elem T) {
	b.buf[b.wrap(b.head+b.len)] = elem

	if b.len < len(b.buf) {
		b.len++
	} else {
		b.head = b.wrap(b.head + 1)
	}
}

// Len returns the current number of elements stored in the buffer.
func (b *RingBuffer[T]) Len() int {
	return b.len
}

// Cap returns the maximum capacity of the buffer (the total number of elements it can hold).
func (b *RingBuffer[T]) Cap() int {
	return len(b.buf)
}

// At returns the element at idx relative to the logical start of the buffer,
// where At(0) is the oldest element and At(Len()-1) is the most recent. It
// panics if idx is negative or not less than Len.
func (b *RingBuffer[T]) At(idx int) T {
	if idx < 0 || idx >= b.len {
		panic("dringbuf: index out of range")
	}
	return b.buf[b.wrap(b.head+idx)]
}

// Get returns the element at idx and true, or the zero value and false if idx
// is negative or not less than Len. Unlike At it does not panic.
func (b *RingBuffer[T]) Get(idx int) (T, bool) {
	if idx < 0 || idx >= b.len {
		var zero T
		return zero, false
	}
	return b.buf[b.wrap(b.head+idx)], true
}

// Last returns the most recently appended element and true, or the zero value
// and false when the buffer is empty.
func (b *RingBuffer[T]) Last() (T, bool) {
	return b.Get(b.len - 1)
}

// Tail returns the last n most recently appended elements in oldest-to-newest
// order. If n exceeds Len it is clamped to Len. The result is a freshly
// allocated copy and may be modified freely. It panics if n is negative or
// greater than Cap.
func (b *RingBuffer[T]) Tail(n int) []T {
	if n < 0 || n > len(b.buf) {
		panic("dringbuf: n out of buffer size")
	}
	return b.AppendTail(make([]T, 0, min(n, b.len)), n)
}

// AppendTail appends the last n most recently appended elements, in
// oldest-to-newest order, to dst and returns the extended slice. If n exceeds
// Len it is clamped to Len. It does not allocate when dst has enough spare
// capacity. It panics if n is negative or greater than Cap.
func (b *RingBuffer[T]) AppendTail(dst []T, n int) []T {
	if n < 0 || n > len(b.buf) {
		panic("dringbuf: n out of buffer size")
	}
	n = min(n, b.len)

	start := b.wrap(b.head + b.len - n)
	if end := start + n; end <= len(b.buf) {
		return append(dst, b.buf[start:end]...)
	}
	dst = append(dst, b.buf[start:]...)
	return append(dst, b.buf[:n-(len(b.buf)-start)]...)
}

// Clear removes all elements from the buffer and resets it to an empty state.
func (b *RingBuffer[T]) Clear() {
	clear(b.buf)
	b.head = 0
	b.len = 0
}

// wrap maps i in [0, 2*Cap) onto [0, Cap) without a division.
func (b *RingBuffer[T]) wrap(i int) int {
	if i >= len(b.buf) {
		i -= len(b.buf)
	}
	return i
}
