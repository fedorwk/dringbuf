package dringbuf

type DRingBuffer[T any] struct {
	buf  []T
	size int
	len  int
	cur  int
}

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

func (b *DRingBuffer[T]) Append(elem T) {
	b.buf[b.cur] = elem
	b.buf[b.cur+b.size] = elem
	b.cur = (b.cur + 1) % b.size

	if b.len < b.size {
		b.len++
	}
}

func (b *DRingBuffer[T]) Len() int {
	return b.len
}

func (b *DRingBuffer[T]) Size() int {
	return b.size
}

func (b *DRingBuffer[T]) At(idx int) T {
	if idx >= b.size {
		panic("idx out of buffer size")
	}
	return b.buf[b.start()+idx]

}

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

func (b *DRingBuffer[T]) Clear() {
	b.buf = b.buf[:0]
	b.cur = 0
	b.len = 0
}

func (b DRingBuffer[T]) start() int {
	return b.cur + b.size - b.len
}
