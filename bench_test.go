package dringbuf_test

import (
	"dringbuf/internal/bench"
	"testing"
)

// Each benchmark iteration transfers benchMessages values, so ns/op is the
// cost of handing off that many values, not a single one.
const benchMessages = 100_000

func benchmarkHandoff(b *testing.B, kind bench.Kind) {
	b.ReportAllocs()
	opts := bench.Options{Kind: kind, Capacity: 256}
	for b.Loop() {
		bench.RunHandoff(opts, benchMessages)
	}
}

func BenchmarkChanUnbuffered(b *testing.B) { benchmarkHandoff(b, bench.ChanUnbuffered) }
func BenchmarkChanBuffered(b *testing.B)   { benchmarkHandoff(b, bench.ChanBuffered) }
func BenchmarkSyncBasic(b *testing.B)      { benchmarkHandoff(b, bench.SyncBasic) }
func BenchmarkSyncDring(b *testing.B)      { benchmarkHandoff(b, bench.SyncDring) }
func BenchmarkMutexBasic(b *testing.B)     { benchmarkHandoff(b, bench.MutexBasic) }
func BenchmarkMutexDring(b *testing.B)     { benchmarkHandoff(b, bench.MutexDring) }
