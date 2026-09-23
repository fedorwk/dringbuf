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
	return runHandoff[Payload8](o, messages, func(i int) Payload8 {
		return Payload8(i)
	}, func(v Payload8) int64 {
		return int64(v)
	}, func(a, b Payload8) bool {
		return a <= b
	})
}

// Payload sizes in bytes. The int benchmarks hand off 8-byte values, so
// Payload8 keeps the checksum logic identical across all variants.
type Payload8 int64

const (
	PayloadSize64  = 64
	PayloadSize1K  = 1024
	PayloadSize4K  = 4096
	PayloadSize16K = 16384
)

// Payload64 is a 64-byte value whose first 8 bytes carry a sequence number.
type Payload64 struct {
	N   int
	Pad [PayloadSize64 - 8]byte
}

// Payload1K is a 1KB value whose first 8 bytes carry a sequence number.
type Payload1K struct {
	N   int
	Pad [PayloadSize1K - 8]byte
}

// Payload4K is a 4KB value whose first 8 bytes carry a sequence number.
type Payload4K struct {
	N   int
	Pad [PayloadSize4K - 8]byte
}

// Payload16K is a 16KB value whose first 8 bytes carry a sequence number.
type Payload16K struct {
	N   int
	Pad [PayloadSize16K - 8]byte
}

// RunPayloadHandoff transfers `messages` 64-byte payloads.
func RunPayloadHandoff64(o Options, messages int) (time.Duration, int, bool) {
	return runPayload(o, messages, PayloadSize64)
}

// RunPayloadHandoff1K transfers `messages` 1KB payloads.
func RunPayloadHandoff1K(o Options, messages int) (time.Duration, int, bool) {
	return runPayload(o, messages, PayloadSize1K)
}

// RunPayloadHandoff4K transfers `messages` 4KB payloads.
func RunPayloadHandoff4K(o Options, messages int) (time.Duration, int, bool) {
	return runPayload(o, messages, PayloadSize4K)
}

// RunPayloadHandoff16K transfers `messages` 16KB payloads.
func RunPayloadHandoff16K(o Options, messages int) (time.Duration, int, bool) {
	return runPayload(o, messages, PayloadSize16K)
}

func runPayload(o Options, messages, size int) (time.Duration, int, bool) {
	switch size {
	case PayloadSize64:
		return runHandoff[Payload64](o, messages,
			func(i int) Payload64 { var p Payload64; p.N = i; p.Pad[0] = byte(i); return p },
			func(v Payload64) int64 { return int64(v.N) },
			func(a, b Payload64) bool { return a.N <= b.N })
	case PayloadSize1K:
		return runHandoff[Payload1K](o, messages,
			func(i int) Payload1K { var p Payload1K; p.N = i; p.Pad[0] = byte(i); return p },
			func(v Payload1K) int64 { return int64(v.N) },
			func(a, b Payload1K) bool { return a.N <= b.N })
	case PayloadSize4K:
		return runHandoff[Payload4K](o, messages,
			func(i int) Payload4K { var p Payload4K; p.N = i; p.Pad[0] = byte(i); return p },
			func(v Payload4K) int64 { return int64(v.N) },
			func(a, b Payload4K) bool { return a.N <= b.N })
	default:
		return runHandoff[Payload16K](o, messages,
			func(i int) Payload16K { var p Payload16K; p.N = i; p.Pad[0] = byte(i); return p },
			func(v Payload16K) int64 { return int64(v.N) },
			func(a, b Payload16K) bool { return a.N <= b.N })
	}
}

func runHandoff[T any](o Options, messages int, gen func(i int) T, sum func(T) int64, less func(a, b T) bool) (time.Duration, int, bool) {
	if o.Capacity < 1 {
		o.Capacity = 1
	}
	switch o.Kind {
	case ChanUnbuffered:
		return runChannel(o, messages, 0, gen, sum)
	case ChanBuffered:
		return runChannel(o, messages, o.Capacity, gen, sum)
	default:
		return runStream(o, messages, gen, sum, less)
	}
}

func runChannel[T any](o Options, messages, capacity int, gen func(i int) T, sum func(T) int64) (time.Duration, int, bool) {
	ch := make(chan T, capacity)
	type result struct {
		sum int64
		n   int
	}
	done := make(chan result)
	go func() {
		var total int64
		n := 0
		for v := range ch {
			total += sum(v)
			n++
		}
		done <- result{total, n}
	}()

	start := time.Now()
	for i := range messages {
		ch <- gen(i)
	}
	close(ch)
	r := <-done

	return time.Since(start), r.n, r.sum == int64(messages)*(int64(messages)-1)/2
}

func runStream[T any](o Options, messages int, gen func(i int) T, sum func(T) int64, less func(a, b T) bool) (time.Duration, int, bool) {
	var bp dringbuf.BackpressureStrategy
	switch o.Kind {
	case StreamDropOldest:
		bp = dringbuf.BackpressureStrategyDropOldest
	case StreamDropNewest:
		bp = dringbuf.BackpressureStrategyDropNewest
	default:
		bp = dringbuf.BackpressureStrategyBlock
	}

	s := dringbuf.NewStream[T]()
	sub := s.Subscribe(o.Capacity, bp)

	type result struct {
		sum int64
		n   int
		ok  bool
	}
	done := make(chan result)
	go func() {
		var total int64
		n := 0
		ok := true
		prev := gen(-1)
		for {
			v, err := sub.Next()
			if err != nil {
				break
			}
			total += sum(v)
			n++
			if less(v, prev) {
				ok = false
			}
			prev = v
		}
		if o.Kind == StreamBlock && total != int64(messages)*(int64(messages)-1)/2 {
			ok = false
		}
		done <- result{total, n, ok}
	}()

	start := time.Now()
	for i := range messages {
		s.Emit(gen(i))
	}
	sub.Close()
	r := <-done

	return time.Since(start), r.n, r.ok
}
