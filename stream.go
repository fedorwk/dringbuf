package dringbuf

import (
	"io"
	"sync"
	"sync/atomic"
)

type BackpressureStrategy int

const (
	BackpressureStrategyBlock BackpressureStrategy = iota
	BackpressureStrategyDropOldest
	BackpressureStrategyDropNewest
)

type Stream[T any] struct {
	subs map[int64]*Subscription[T]
	inc  atomic.Int64
	mu   sync.Mutex
}

func NewStream[T any]() *Stream[T] {
	return &Stream[T]{
		subs: make(map[int64]*Subscription[T]),
	}
}

func (hs *Stream[T]) Emit(val T) {
	for _, s := range hs.subs {
		s.emit(val)
	}
}

type Subscription[T any] struct {
	buf    *ThreadSafeRingBuffer[*RingBuffer[T], T]
	bp     BackpressureStrategy
	offset int

	cancelFn func()
	closed   bool

	mu   sync.Mutex
	cond *sync.Cond
}

// Close stops the subscription: no further values are accepted and a pending
// Next call wakes up. Buffered values can still be read; Next returns io.EOF
// once the subscription is closed and drained. Close is idempotent.
func (s *Subscription[T]) Close() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	s.cond.Broadcast()
	s.mu.Unlock()

	s.cancelFn()
}

func (s *Subscription[T]) emit(v T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	switch s.bp {
	case BackpressureStrategyDropNewest:
		if s.offset >= s.buf.buf.cap { // skip value if full
			return
		}
	case BackpressureStrategyBlock:
		for s.offset >= s.buf.buf.cap { // wait for reader to free space
			s.cond.Wait()
		}
	case BackpressureStrategyDropOldest:
	}
	s.buf.Append(v)
	if s.offset < s.buf.buf.cap {
		s.offset++
	}
	s.cond.Signal()
}

func (s *Subscription[T]) Next() (T, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for s.offset <= 0 {
		if s.closed {
			var zero T
			return zero, io.EOF
		}
		s.cond.Wait()
	}
	val := s.buf.At(s.buf.Len() - s.offset)
	s.offset--
	s.cond.Signal()
	return val, nil
}

func (hs *Stream[T]) Subscribe(bufsize int, bp BackpressureStrategy) *Subscription[T] {
	id := hs.inc.Add(1)
	cancel := func() {
		hs.mu.Lock()
		delete(hs.subs, id)
		hs.mu.Unlock()
	}
	sub := &Subscription[T]{
		buf:      NewThreadSafeRingBuffer[T](bufsize),
		bp:       bp,
		offset:   0,
		cancelFn: cancel,

		mu: sync.Mutex{},
	}
	sub.cond = sync.NewCond(&sub.mu)

	hs.mu.Lock()
	hs.subs[id] = sub
	hs.mu.Unlock()

	return sub
}
