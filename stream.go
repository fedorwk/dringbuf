package dringbuf

import (
	"context"
	"io"
	"sync"
	"sync/atomic"
)

// BackpressureStrategy decides what a Subscription does with an emitted value
// when its buffer is full.
type BackpressureStrategy int

const (
	// BackpressureStrategyBlock makes Emit wait until the reader frees space.
	// No value is lost, but a slow subscriber stalls Emit for every
	// subscriber of the stream.
	BackpressureStrategyBlock BackpressureStrategy = iota
	// BackpressureStrategyDropOldest overwrites the oldest unread value.
	BackpressureStrategyDropOldest
	// BackpressureStrategyDropNewest discards the emitted value.
	BackpressureStrategyDropNewest
)

// Stream fans every emitted value out to all of its subscriptions.
type Stream[T any] struct {
	// subs is a copy-on-write snapshot, so Emit reads it without locking.
	subs atomic.Pointer[[]*Subscription[T]]

	mu     sync.Mutex // serializes writers of subs and guards closed
	closed bool
}

// NewStream creates an open Stream with no subscriptions.
func NewStream[T any]() *Stream[T] {
	return &Stream[T]{}
}

// Emit delivers val to every subscription, one after another, applying each
// subscription's backpressure strategy. A subscription with
// BackpressureStrategyBlock and a full buffer makes Emit wait, which also
// delays delivery to the subscriptions after it. Emit after Close does nothing.
func (hs *Stream[T]) Emit(val T) {
	for _, s := range hs.list() {
		s.emit(val)
	}
}

// Subscribe registers a new subscription that buffers up to bufsize unread
// values. It panics if bufsize is not positive or bp is not a known strategy.
// Subscribing to a closed stream returns a subscription that is already at
// io.EOF.
func (hs *Stream[T]) Subscribe(bufsize int, bp BackpressureStrategy) *Subscription[T] {
	if bp < BackpressureStrategyBlock || bp > BackpressureStrategyDropNewest {
		panic("dringbuf: unknown backpressure strategy")
	}
	sub := &Subscription[T]{
		buf:    NewDoubleRingBuffer[T](bufsize),
		bp:     bp,
		stream: hs,
	}
	sub.notEmpty.L = &sub.mu
	sub.notFull.L = &sub.mu

	hs.mu.Lock()
	defer hs.mu.Unlock()
	if hs.closed {
		sub.eof = true
		return sub
	}
	old := hs.list()
	next := append(old[:len(old):len(old)], sub) // always copies
	hs.subs.Store(&next)
	return sub
}

// Close ends the stream: every subscription can still read the values it has
// buffered and then gets io.EOF, blocked Emit calls return, and later Emit
// calls do nothing. Close is idempotent.
func (hs *Stream[T]) Close() {
	hs.mu.Lock()
	if hs.closed {
		hs.mu.Unlock()
		return
	}
	hs.closed = true
	subs := hs.list()
	hs.subs.Store(nil)
	hs.mu.Unlock()

	for _, s := range subs {
		s.end()
	}
}

func (hs *Stream[T]) list() []*Subscription[T] {
	if p := hs.subs.Load(); p != nil {
		return *p
	}
	return nil
}

func (hs *Stream[T]) remove(sub *Subscription[T]) {
	hs.mu.Lock()
	defer hs.mu.Unlock()
	old := hs.list()
	next := make([]*Subscription[T], 0, len(old))
	for _, s := range old {
		if s != sub {
			next = append(next, s)
		}
	}
	if len(next) != len(old) {
		hs.subs.Store(&next)
	}
}

// Subscription is one reader's buffered view of a Stream. Its read methods may
// be called from several goroutines.
type Subscription[T any] struct {
	stream *Stream[T]
	bp     BackpressureStrategy

	mu sync.Mutex
	// The unread values are always buf.Tail(unread), a contiguous slice.
	buf    *DoubleRingBuffer[T]
	unread int
	// eof is set once no more values will arrive: the stream or the
	// subscription was closed.
	eof    bool
	closed bool

	notEmpty sync.Cond // readers wait here for unread > 0 or eof
	notFull  sync.Cond // Block producers wait here for space or eof
}

// Close unsubscribes: buffered values are dropped, blocked Emit calls for this
// subscription return, and every read returns io.EOF. Close is idempotent.
func (s *Subscription[T]) Close() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	s.eof = true
	s.unread = 0
	s.buf.Clear()
	s.notEmpty.Broadcast()
	s.notFull.Broadcast()
	s.mu.Unlock()

	s.stream.remove(s)
}

// end marks the subscription as finished without dropping buffered values.
func (s *Subscription[T]) end() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.eof = true
	s.notEmpty.Broadcast()
	s.notFull.Broadcast()
}

func (s *Subscription[T]) emit(v T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.eof {
		return
	}
	capacity := s.buf.Cap()
	switch s.bp {
	case BackpressureStrategyDropNewest:
		if s.unread == capacity {
			return
		}
	case BackpressureStrategyBlock:
		for s.unread == capacity && !s.eof {
			s.notFull.Wait()
		}
		if s.eof {
			return
		}
	case BackpressureStrategyDropOldest:
	}
	s.buf.Append(v)
	if s.unread < capacity {
		s.unread++
	}
	s.notEmpty.Signal()
}

// Next returns the oldest unread value, blocking until one is available. It
// returns io.EOF once the subscription has ended and has nothing left to read.
func (s *Subscription[T]) Next() (T, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.waitReadable(context.Background()); err != nil {
		var zero T
		return zero, err
	}
	return s.pop(), nil
}

// NextContext is like Next but stops waiting when ctx is done, returning
// ctx.Err(). A value that is already buffered is returned even if ctx is done.
func (s *Subscription[T]) NextContext(ctx context.Context) (T, error) {
	if ctx.Done() == nil {
		return s.Next()
	}
	stop := context.AfterFunc(ctx, func() {
		s.mu.Lock()
		s.notEmpty.Broadcast()
		s.mu.Unlock()
	})
	defer stop()

	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.waitReadable(ctx); err != nil {
		var zero T
		return zero, err
	}
	return s.pop(), nil
}

// Read copies up to len(dst) of the oldest unread values into dst, blocking
// until at least one is available, and returns how many it copied. It returns
// 0 and io.EOF once the subscription has ended and has nothing left to read.
func (s *Subscription[T]) Read(dst []T) (int, error) {
	if len(dst) == 0 {
		return 0, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.waitReadable(context.Background()); err != nil {
		return 0, err
	}
	n := copy(dst, s.buf.Tail(s.unread))
	s.unread -= n
	s.notFull.Broadcast()
	return n, nil
}

// ReadView blocks until at least one value is available and then calls fn
// with all unread values, oldest first, without copying. The values count as
// read once fn is called. The slice aliases internal storage: it must not be
// modified or retained after fn returns. Emit and other reads on this
// subscription wait while fn runs. It returns io.EOF once the subscription has
// ended and has nothing left to read.
func (s *Subscription[T]) ReadView(fn func([]T)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.waitReadable(context.Background()); err != nil {
		return err
	}
	view := s.buf.Tail(s.unread)
	s.unread = 0
	defer s.notFull.Broadcast()
	fn(view)
	return nil
}

// pop removes and returns the oldest unread value. s.mu must be held and
// unread must be positive.
func (s *Subscription[T]) pop() T {
	v := s.buf.At(s.buf.Len() - s.unread)
	s.unread--
	s.notFull.Signal()
	return v
}

// waitReadable waits until there is an unread value. It returns io.EOF when
// the subscription has ended and is drained, or ctx.Err() when ctx is done.
// s.mu must be held.
func (s *Subscription[T]) waitReadable(ctx context.Context) error {
	for s.unread == 0 {
		if s.eof {
			return io.EOF
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		s.notEmpty.Wait()
	}
	return nil
}
