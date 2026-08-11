package bench

import (
	"dringbuf"
	"time"
)

// Kind identifies a goroutine-handoff implementation.
type Kind int

const (
	// ChanUnbuffered hands values over an unbuffered channel.
	ChanUnbuffered Kind = iota
	// ChanBuffered hands values over a channel with Options.Capacity buffer.
	ChanBuffered
	// StreamBlock uses a Stream Subscription with Blocking backpressure, which
	// matches channel semantics: every emitted value is received exactly once.
	StreamBlock
	// StreamDropOldest uses a Stream Subscription that overwrites the oldest
	// value when the buffer is full, so the consumer receives the newest values.
	StreamDropOldest
	// StreamDropNewest uses a Stream Subscription that skips new values when
	// the buffer is full, so the consumer receives the earliest values.
	StreamDropNewest
)

// Options configures a handoff run.
type Options struct {
	Kind     Kind
	Capacity int
}

// RunHandoff transfers `messages` ints from a producer goroutine to a consumer
// goroutine. It returns the elapsed time, the number of values the consumer
// actually received, and whether the received sequence was correct: in order,
// strictly increasing and — for the lossless strategies — carrying the full
// checksum. Drop strategies deliberately deliver at most Options.Capacity
// values, so the checksum does not apply to them.
func RunHandoff(o Options, messages int) (time.Duration, int, bool) {
	if o.Capacity < 1 {
		o.Capacity = 1
	}
	switch o.Kind {
	case ChanUnbuffered:
		return runChannel(o, messages, 0)
	case ChanBuffered:
		return runChannel(o, messages, o.Capacity)
	default:
		return runStream(o, messages)
	}
}

func runChannel(o Options, messages, capacity int) (time.Duration, int, bool) {
	ch := make(chan int, capacity)
	type result struct {
		sum int
		n   int
	}
	done := make(chan result)
	go func() {
		var sum, n int
		for v := range ch {
			sum += v
			n++
		}
		done <- result{sum, n}
	}()

	start := time.Now()
	for i := range messages {
		ch <- i
	}
	close(ch)
	r := <-done

	return time.Since(start), r.n, r.sum == messages*(messages-1)/2
}

func runStream(o Options, messages int) (time.Duration, int, bool) {
	var bp dringbuf.BackpressureStrategy
	switch o.Kind {
	case StreamDropOldest:
		bp = dringbuf.BackpressureStrategyDropOldest
	case StreamDropNewest:
		bp = dringbuf.BackpressureStrategyDropNewest
	default:
		bp = dringbuf.BackpressureStrategyBlock
	}

	s := dringbuf.NewStream[int]()
	sub := s.Subscribe(o.Capacity, bp)

	type result struct {
		sum int
		n   int
		ok  bool
	}
	done := make(chan result)
	go func() {
		var sum, n int
		prev := -1
		ok := true
		for {
			v, err := sub.Next()
			if err != nil {
				break
			}
			sum += v
			n++
			if v <= prev {
				ok = false
			}
			prev = v
		}
		if o.Kind == StreamBlock && sum != messages*(messages-1)/2 {
			ok = false
		}
		done <- result{sum, n, ok}
	}()

	start := time.Now()
	for i := range messages {
		s.Emit(i)
	}
	sub.Close()
	r := <-done

	return time.Since(start), r.n, r.ok
}
