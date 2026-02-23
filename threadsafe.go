package dringbuf

import "sync"

type threadSafe[T any] struct {
	sync.RWMutex
	buf *DRingBuffer[T]
}

func NewThreadSafeBuffer[T any](size int) *threadSafe[T] {
	return &threadSafe[T]{
		buf: NewDRingBuffer[T](size),
	}
}

func (b *threadSafe[T]) Append(elem T) {
	b.Lock()
	defer b.Unlock()
	b.Append(elem)
}

func (b *threadSafe[T]) Len() int {
	b.RLock()
	defer b.RUnlock()
	return b.Len()
}

func (b *threadSafe[T]) Size() int {
	b.RLock()
	defer b.RUnlock()
	return b.Size()
}

func (b *threadSafe[T]) At(idx int) T {
	b.RLock()
	defer b.RUnlock()
	return b.At(idx)
}

func (b *threadSafe[T]) Last(n int) []T {
	b.RLock()
	defer b.RUnlock()
	return b.Last(n)
}

func (b *threadSafe[T]) Clear() {
	b.Lock()
	defer b.Unlock()
	b.Clear()
}
