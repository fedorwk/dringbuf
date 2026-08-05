package dringbuf

type buffer[T any] struct {
	buf  []T
	head int
	len  int
	cap  int
}

func NewRingBuffer[T any](size int) RingBuffer[T] {
	return &buffer[T]{
		buf: make([]T, size),
		cap: size,
	}
}

func (b *buffer[T]) Append(elem T) {
	idx := (b.head + b.len) % b.cap
	b.buf[idx] = elem

	if b.len < b.cap {
		b.len++
	} else {
		b.head = (b.head + 1) % b.cap
	}
}

func (b *buffer[T]) Len() int {
	return b.len
}

func (b *buffer[T]) Cap() int {
	return b.cap
}

func (b *buffer[T]) At(idx int) T {
	if idx >= b.cap {
		panic("idx out of buffer size")
	}
	return b.buf[(b.head+idx)%b.cap]
}

func (b *buffer[T]) Last(n int) []T {
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

func (b *buffer[T]) Clear() {
	var zero T
	for i := range b.buf {
		b.buf[i] = zero
	}
	b.head = 0
	b.len = 0
}
