package dringbuf

type bufferBase[T any] struct {
	buf  []T
	size int
	len  int
	cur  int
}

func NewRingBuffer[T any](size int) RingBuffer[T] {
	doubleSize := size * 2
	if doubleSize < size { // integer overflow
		panic("struct size overflow. Max possible size of buffer is MaxInt/2 for target architecture")
	}
	return &bufferBase[T]{
		buf:  make([]T, doubleSize),
		size: size,
		len:  0,
		cur:  0,
	}
}

func (b *bufferBase[T]) Append(elem T) {
	b.buf[b.cur] = elem
	b.buf[b.cur+b.size] = elem
	b.cur = (b.cur + 1) % b.size

	if b.len < b.size {
		b.len++
	}
}

func (b *bufferBase[T]) Len() int {
	return b.len
}

func (b *bufferBase[T]) Size() int {
	return b.size
}

func (b *bufferBase[T]) At(idx int) T {
	if idx >= b.size {
		panic("idx out of buffer size")
	}
	return b.buf[b.start()+idx]

}

func (b *bufferBase[T]) Last(n int) []T {
	if n > b.size {
		panic("n out of buffer size")
	}
	if n > b.len {
		return b.buf[0:b.len]
	}

	end := b.size + b.cur

	return b.buf[end-n : end]
}

func (b *bufferBase[T]) Clear() {
	b.buf = b.buf[:0]
	b.cur = 0
	b.len = 0
}

func (b bufferBase[T]) start() int {
	return b.cur + b.size - b.len
}
