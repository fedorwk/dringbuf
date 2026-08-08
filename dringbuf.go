package dringbuf

// DRingBuffer is a double-sized circular buffer that keeps its logical window
// contiguous in memory.
type DRingBuffer[T any] struct {
	buf  []T
	size int
	len  int
	cur  int
}

// NewDRingBuffer creates a DRingBuffer with the given capacity.
func NewDRingBuffer[T any](size int) *DRingBuffer[T] {
	doubleSize := size * 2
	if doubleSize < size { // integer overflow
		panic("struct size overflow. Max size of buffer is MaxInt/2 for target architecture")
	}
	return &DRingBuffer[T]{
		buf:  make([]T, doubleSize),
		size: size,
		len:  0,
		cur:  0,
	}
}

// Append adds a new element to the buffer. If the buffer is already full, the oldest element is overwritten.
func (b *DRingBuffer[T]) Append(elem T) {
	b.buf[b.cur] = elem
	b.buf[b.cur+b.size] = elem
	b.cur = (b.cur + 1) % b.size

	if b.len < b.size {
		b.len++
	}
}

// Len returns the current number of elements stored in the buffer.
func (b *DRingBuffer[T]) Len() int {
	return b.len
}

// Cap returns the maximum capacity of the buffer (the total number of elements it can hold).
func (b *DRingBuffer[T]) Cap() int {
	return b.size
}

// At retrieves the element at the specified index relative to the logical start of the buffer.
func (b *DRingBuffer[T]) At(idx int) T {
	if idx >= b.size {
		panic("idx out of buffer size")
	}
	return b.buf[b.start()+idx]

}

// Last returns the last n most recently appended elements in order. The result
// is a view into internal storage; do not modify it.
func (b *DRingBuffer[T]) Last(n int) []T {
	if n > b.size {
		panic("n out of buffer size")
	}
	if n > b.len {
		return b.buf[0:b.len]
	}

	end := b.size + b.cur

	return b.buf[end-n : end]
}

// Clear removes all elements from the buffer and resets it to an empty state.
func (b *DRingBuffer[T]) Clear() {
	var zero T
	for i := range b.buf {
		b.buf[i] = zero
	}
	b.cur = 0
	b.len = 0
}

func (b DRingBuffer[T]) start() int {
	return b.cur + b.size - b.len
}
