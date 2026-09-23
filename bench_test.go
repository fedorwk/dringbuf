package dringbuf_test

import (
	"dringbuf/internal/bench"
	"testing"
	"time"
)

// Each benchmark iteration has a producer attempt to hand off benchMessages
// values to a consumer that drains the stream/channel until it is closed, so
// ns/op is the per-value throughput of that handoff. The drop backpressure
// stream strategies deliver at most benchMessages (fewer if the consumer
// cannot keep up), so their per-iteration cost is measured over the values
// actually delivered.
const benchMessages = 100_000

// Payload benchmarks transfer larger value types, so they use fewer messages
// to keep the total byte volume (and thus run time) comparable.
const benchPayloadMessages = 10_000

func benchmarkHandoff(b *testing.B, kind bench.Kind) {
	b.ReportAllocs()
	opts := bench.Options{Kind: kind, Capacity: 256}
	for b.Loop() {
		bench.RunHandoff(opts, benchMessages)
	}
}

func benchmarkPayloadHandoff(b *testing.B, kind bench.Kind, run func(bench.Options, int) (time.Duration, int, bool)) {
	b.ReportAllocs()
	opts := bench.Options{Kind: kind, Capacity: 256}
	for b.Loop() {
		run(opts, benchPayloadMessages)
	}
}

func BenchmarkChanUnbuffered(b *testing.B)   { benchmarkHandoff(b, bench.ChanUnbuffered) }
func BenchmarkChanBuffered(b *testing.B)     { benchmarkHandoff(b, bench.ChanBuffered) }
func BenchmarkStreamBlock(b *testing.B)      { benchmarkHandoff(b, bench.StreamBlock) }
func BenchmarkStreamDropOldest(b *testing.B) { benchmarkHandoff(b, bench.StreamDropOldest) }
func BenchmarkStreamDropNewest(b *testing.B) { benchmarkHandoff(b, bench.StreamDropNewest) }

func BenchmarkChanUnbufferedPayload1K(b *testing.B) {
	benchmarkPayloadHandoff(b, bench.ChanUnbuffered, bench.RunPayloadHandoff1K)
}
func BenchmarkChanBufferedPayload1K(b *testing.B) {
	benchmarkPayloadHandoff(b, bench.ChanBuffered, bench.RunPayloadHandoff1K)
}
func BenchmarkStreamBlockPayload1K(b *testing.B) {
	benchmarkPayloadHandoff(b, bench.StreamBlock, bench.RunPayloadHandoff1K)
}
func BenchmarkStreamDropOldestPayload1K(b *testing.B) {
	benchmarkPayloadHandoff(b, bench.StreamDropOldest, bench.RunPayloadHandoff1K)
}
func BenchmarkStreamDropNewestPayload1K(b *testing.B) {
	benchmarkPayloadHandoff(b, bench.StreamDropNewest, bench.RunPayloadHandoff1K)
}

func BenchmarkChanUnbufferedPayload16K(b *testing.B) {
	benchmarkPayloadHandoff(b, bench.ChanUnbuffered, bench.RunPayloadHandoff16K)
}
func BenchmarkChanBufferedPayload16K(b *testing.B) {
	benchmarkPayloadHandoff(b, bench.ChanBuffered, bench.RunPayloadHandoff16K)
}
func BenchmarkStreamBlockPayload16K(b *testing.B) {
	benchmarkPayloadHandoff(b, bench.StreamBlock, bench.RunPayloadHandoff16K)
}
func BenchmarkStreamDropOldestPayload16K(b *testing.B) {
	benchmarkPayloadHandoff(b, bench.StreamDropOldest, bench.RunPayloadHandoff16K)
}
func BenchmarkStreamDropNewestPayload16K(b *testing.B) {
	benchmarkPayloadHandoff(b, bench.StreamDropNewest, bench.RunPayloadHandoff16K)
}
