package dringbuf_test

import (
	"dringbuf/internal/bench"
	"testing"
)

// Each benchmark iteration has a producer attempt to hand off benchMessages
// values to a consumer that drains the stream/channel until it is closed, so
// ns/op is the per-value throughput of that handoff. The drop backpressure
// stream strategies deliver at most benchMessages (fewer if the consumer
// cannot keep up), so their per-iteration cost is measured over the values
// actually delivered.
const benchMessages = 100_000

func benchmarkHandoff(b *testing.B, kind bench.Kind) {
	b.ReportAllocs()
	opts := bench.Options{Kind: kind, Capacity: 256}
	for b.Loop() {
		bench.RunHandoff(opts, benchMessages)
	}
}

func BenchmarkChanUnbuffered(b *testing.B)   { benchmarkHandoff(b, bench.ChanUnbuffered) }
func BenchmarkChanBuffered(b *testing.B)     { benchmarkHandoff(b, bench.ChanBuffered) }
func BenchmarkStreamBlock(b *testing.B)      { benchmarkHandoff(b, bench.StreamBlock) }
func BenchmarkStreamDropOldest(b *testing.B) { benchmarkHandoff(b, bench.StreamDropOldest) }
func BenchmarkStreamDropNewest(b *testing.B) { benchmarkHandoff(b, bench.StreamDropNewest) }
