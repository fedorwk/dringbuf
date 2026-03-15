package dringbuf

import "sync"

type threadSafe[T any] struct {
	mu  sync.RWMutex
	buf RingBuffer[T]
}

func NewThreadSafeRingBuffer[T any](size int) SyncRingBuffer[T] {
	return &threadSafe[T]{
		buf: NewRingBuffer[T](size),
	}
}

type release func()

// Returns underlying data with n last elements
// Locks buffer for reading until `release` call
func (b *threadSafe[T]) Borrow(n int) ([]T, release) {
	b.mu.RLock()
	release := func() {
		b.mu.RUnlock()
	}
	return b.buf.Last(n), release
}

func (b *threadSafe[T]) Append(elem T) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buf.Append(elem)
}

func (b *threadSafe[T]) Len() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.buf.Len()
}

func (b *threadSafe[T]) Size() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.buf.Size()
}

func (b *threadSafe[T]) At(idx int) T {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.buf.At(idx)
}

// Last returns copy of underlying data
// To take advantage of threadsafe implementation use Borrow method instead
func (b *threadSafe[T]) Last(n int) []T {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if n > b.Size() {
		panic("n out of buffer size")
	}
	if l := b.Len(); l < n {
		n = l
	}
	res := make([]T, n)
	copy(res, b.buf.Last(n))
	return res
}

func (b *threadSafe[T]) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buf.Clear()
}
