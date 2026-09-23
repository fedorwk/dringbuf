package dringbuf

// DoubleRingBuffer is a double-sized circular buffer that keeps its logical window
// contiguous in memory.
type DoubleRingBuffer[T any] struct {
	buf  []T
	size int
	len  int
	cur  int
}

// NewDoubleRingBuffer creates a DoubleRingBuffer with the given capacity. It panics if
// size is not positive or if 2*size overflows int.
func NewDoubleRingBuffer[T any](size int) *DoubleRingBuffer[T] {
	if size <= 0 {
		panic("dringbuf: size must be positive")
	}
	doubleSize := size * 2
	if doubleSize < size { // integer overflow
		panic("struct size overflow. Max size of buffer is MaxInt/2 for target architecture")
	}
	return &DoubleRingBuffer[T]{
		buf:  make([]T, doubleSize),
		size: size,
		len:  0,
		cur:  0,
	}
}

// Append adds a new element to the buffer. If the buffer is already full, the oldest element is overwritten.
func (b *DoubleRingBuffer[T]) Append(elem T) {
	b.buf[b.cur] = elem
	b.buf[b.cur+b.size] = elem
	b.cur = (b.cur + 1) % b.size

	if b.len < b.size {
		b.len++
	}
}

// Len returns the current number of elements stored in the buffer.
func (b *DoubleRingBuffer[T]) Len() int {
	return b.len
}

// Cap returns the maximum capacity of the buffer (the total number of elements it can hold).
func (b *DoubleRingBuffer[T]) Cap() int {
	return b.size
}

// At returns the element at idx relative to the logical start of the buffer,
// where At(0) is the oldest element and At(Len()-1) is the most recent. It
// panics if idx is negative or not less than Len.
func (b *DoubleRingBuffer[T]) At(idx int) T {
	if idx < 0 || idx >= b.len {
		panic("dringbuf: index out of range")
	}
	return b.buf[b.start()+idx]
}

// Get returns the element at idx and true, or the zero value and false if idx
// is negative or not less than Len. Unlike At it does not panic.
func (b *DoubleRingBuffer[T]) Get(idx int) (T, bool) {
	if idx < 0 || idx >= b.len {
		var zero T
		return zero, false
	}
	return b.buf[b.start()+idx], true
}

// Last returns the most recently appended element and true, or the zero value
// and false when the buffer is empty.
func (b *DoubleRingBuffer[T]) Last() (T, bool) {
	return b.Get(b.len - 1)
}

// Tail returns the last n most recently appended elements in oldest-to-newest
// order. If n exceeds Len it is clamped to Len. The result is a view into
// internal storage and aliases the live window; do not modify it and do not
// retain it after the next Append or Clear. It panics if n is negative or
// greater than Cap.
func (b *DoubleRingBuffer[T]) Tail(n int) []T {
	if n < 0 || n > b.size {
		panic("dringbuf: n out of buffer size")
	}
	if n > b.len {
		return b.buf[0:b.len]
	}

	end := b.size + b.cur

	return b.buf[end-n : end]
}

// Clear removes all elements from the buffer and resets it to an empty state.
func (b *DoubleRingBuffer[T]) Clear() {
	var zero T
	for i := range b.buf {
		b.buf[i] = zero
	}
	b.cur = 0
	b.len = 0
}

func (b DoubleRingBuffer[T]) start() int {
	return b.cur + b.size - b.len
}
