package bench

import (
	"dringbuf"
	"sync"
	"time"
)

// Kind identifies a goroutine-handoff implementation.
type Kind int

const (
	// ChanUnbuffered hands values over an unbuffered channel.
	ChanUnbuffered Kind = iota
	// ChanBuffered hands values over a channel with Options.Capacity buffer.
	ChanBuffered
	// SyncBasic uses NewThreadSafeRingBuffer (own Mutex) plus the queue's Mutex.
	SyncBasic
	// SyncDring uses NewThreadSafeDRingBuffer (own Mutex) plus the queue's Mutex.
	SyncDring
	// MutexBasic uses NewRingBuffer guarded only by the queue's single Mutex.
	MutexBasic
	// MutexDring uses NewDRingBuffer guarded only by the queue's single Mutex.
	MutexDring
)

// Options configures a handoff run.
type Options struct {
	Kind     Kind
	Capacity int
}

// RunHandoff transfers `messages` ints from a producer goroutine to a consumer
// goroutine, delivering every value exactly once. It returns the elapsed time
// and whether the consumer received the correct total
// (sum == messages*(messages-1)/2), proving no message was lost or duplicated.
func RunHandoff(o Options, messages int) (time.Duration, bool) {
	if o.Capacity < 1 {
		o.Capacity = 1
	}
	if o.Kind == ChanUnbuffered || o.Kind == ChanBuffered {
		return runChannel(o, messages)
	}
	return runQueue(o, messages)
}

func runChannel(o Options, messages int) (time.Duration, bool) {
	capacity := 0
	if o.Kind == ChanBuffered {
		capacity = o.Capacity
	}

	ch := make(chan int, capacity)
	result := make(chan int)
	go func() {
		var sum int
		for i := 0; i < messages; i++ {
			sum += <-ch
		}
		result <- sum
	}()

	start := time.Now()
	for i := 0; i < messages; i++ {
		ch <- i
	}
	close(ch)
	sum := <-result

	return time.Since(start), sum == messages*(messages-1)/2
}

func runQueue(o Options, messages int) (time.Duration, bool) {
	var storage dringbuf.RingBuffer[int]
	switch o.Kind {
	case MutexBasic:
		storage = dringbuf.NewRingBuffer[int](o.Capacity)
	case MutexDring:
		storage = dringbuf.NewDRingBuffer[int](o.Capacity)
	case SyncDring:
		storage = dringbuf.NewThreadSafeDRingBuffer[int](o.Capacity)
	default:
		storage = dringbuf.NewThreadSafeRingBuffer[int](o.Capacity)
	}
	q := newQueue(storage)

	result := make(chan int)
	go func() {
		var sum int
		for i := 0; i < messages; i++ {
			sum += q.pop()
		}
		result <- sum
	}()

	start := time.Now()
	for i := 0; i < messages; i++ {
		q.push(i)
	}
	sum := <-result

	return time.Since(start), sum == messages*(messages-1)/2
}

// buffer is the subset of the ring-buffer interface the queue needs. Both the
// raw RingBuffer and the Mutex-guarded SyncRingBuffer satisfy it; for the
// raw variants q.mu is the only lock protecting storage.
type buffer interface {
	Append(elem int)
	Len() int
	Cap() int
	At(idx int) int
}

// queue wraps a ring buffer with channel-like blocking semantics: push blocks
// while the buffer is full, pop blocks while it is empty. Every pushed value
// is popped exactly once, in order, so the handoff is lossless.
//
// A ring buffer never shrinks on read, so the adapter tracks how many values
// were pushed and popped and reads the oldest unread one via At. Reads and
// writes are serialized by q.mu, so the buffer's window is stable during pop.
type queue struct {
	mu       sync.Mutex
	notFull  *sync.Cond
	notEmpty *sync.Cond
	buf      buffer
	capacity int
	pushed   int
	popped   int
}

func newQueue(buf buffer) *queue {
	q := &queue{buf: buf, capacity: buf.Cap()}
	q.notFull = sync.NewCond(&q.mu)
	q.notEmpty = sync.NewCond(&q.mu)
	return q
}

func (q *queue) push(v int) {
	q.mu.Lock()
	defer q.mu.Unlock()

	for q.pushed-q.popped == q.capacity {
		q.notFull.Wait()
	}
	q.buf.Append(v)
	q.pushed++
	q.notEmpty.Signal()
}

func (q *queue) pop() int {
	q.mu.Lock()
	defer q.mu.Unlock()

	for q.pushed == q.popped {
		q.notEmpty.Wait()
	}
	unread := q.pushed - q.popped
	v := q.buf.At(q.buf.Len() - unread)
	q.popped++
	q.notFull.Signal()
	return v
}
