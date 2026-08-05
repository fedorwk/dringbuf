package dringbuf

type dbuffer[T any] struct {
	buf  []T
	size int
	len  int
	cur  int
}

func NewDRingBuffer[T any](size int) RingBuffer[T] {
	doubleSize := size * 2
	if doubleSize < size { // integer overflow
		panic("struct size overflow. Max size of buffer is MaxInt/2 for target architecture")
	}
	return &dbuffer[T]{
		buf:  make([]T, doubleSize),
		size: size,
		len:  0,
		cur:  0,
	}
}

func (b *dbuffer[T]) Append(elem T) {
	b.buf[b.cur] = elem
	b.buf[b.cur+b.size] = elem
	b.cur = (b.cur + 1) % b.size

	if b.len < b.size {
		b.len++
	}
}

func (b *dbuffer[T]) Len() int {
	return b.len
}

func (b *dbuffer[T]) Cap() int {
	return b.size
}

func (b *dbuffer[T]) At(idx int) T {
	if idx >= b.size {
		panic("idx out of buffer size")
	}
	return b.buf[b.start()+idx]

}

func (b *dbuffer[T]) Last(n int) []T {
	if n > b.size {
		panic("n out of buffer size")
	}
	if n > b.len {
		return b.buf[0:b.len]
	}

	end := b.size + b.cur

	return b.buf[end-n : end]
}

func (b *dbuffer[T]) Clear() {
	var zero T
	for i := range b.buf {
		b.buf[i] = zero
	}
	b.cur = 0
	b.len = 0
}

func (b dbuffer[T]) start() int {
	return b.cur + b.size - b.len
}
