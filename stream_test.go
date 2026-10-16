package dringbuf_test

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/fedorwk/dringbuf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStream_FIFOOrder(t *testing.T) {
	t.Parallel()

	s := dringbuf.NewStream[int]()
	sub := s.Subscribe(5, dringbuf.BackpressureStrategyDropOldest)

	for i := 1; i <= 3; i++ {
		s.Emit(i)
	}

	for i := 1; i <= 3; i++ {
		v, err := sub.Next()
		require.NoError(t, err)
		assert.Equal(t, i, v)
	}
}

func TestStream_WrapAroundNoPanic(t *testing.T) {
	t.Parallel()

	s := dringbuf.NewStream[int]()
	sub := s.Subscribe(3, dringbuf.BackpressureStrategyDropOldest)

	for i := 1; i <= 6; i++ {
		s.Emit(i)
	}

	for i := 4; i <= 6; i++ {
		v, err := sub.Next()
		require.NoError(t, err)
		assert.Equal(t, i, v)
	}
}

func TestStream_DropNewest(t *testing.T) {
	t.Parallel()

	s := dringbuf.NewStream[int]()
	sub := s.Subscribe(3, dringbuf.BackpressureStrategyDropNewest)

	for i := 1; i <= 5; i++ {
		s.Emit(i)
	}

	for i := 1; i <= 3; i++ {
		v, err := sub.Next()
		require.NoError(t, err)
		assert.Equal(t, i, v)
	}
}

func TestStream_DropOldest(t *testing.T) {
	t.Parallel()

	s := dringbuf.NewStream[int]()
	sub := s.Subscribe(3, dringbuf.BackpressureStrategyDropOldest)

	for i := 1; i <= 5; i++ {
		s.Emit(i)
	}

	for i := 3; i <= 5; i++ {
		v, err := sub.Next()
		require.NoError(t, err)
		assert.Equal(t, i, v)
	}
}

func TestStream_NextBlocksWhenEmpty(t *testing.T) {
	t.Parallel()

	s := dringbuf.NewStream[int]()
	sub := s.Subscribe(3, dringbuf.BackpressureStrategyBlock)

	got := make(chan int)
	go func() {
		v, err := sub.Next()
		if err != nil {
			t.Error(err)
			return
		}
		got <- v
	}()

	select {
	case v := <-got:
		t.Fatalf("Next returned %d on empty stream", v)
	case <-time.After(100 * time.Millisecond):
	}

	s.Emit(42)

	select {
	case v := <-got:
		assert.Equal(t, 42, v)
	case <-time.After(time.Second):
		t.Fatal("Next did not unblock after emit")
	}
}

func TestStream_EmitBlocksWhenFull(t *testing.T) {
	t.Parallel()

	s := dringbuf.NewStream[int]()
	sub := s.Subscribe(3, dringbuf.BackpressureStrategyBlock)

	for i := 1; i <= 3; i++ {
		s.Emit(i)
	}

	done := make(chan struct{})
	go func() {
		s.Emit(4)
		close(done)
	}()

	select {
	case <-done:
		t.Fatal("emit should block when buffer is full")
	case <-time.After(100 * time.Millisecond):
	}

	for i := 1; i <= 4; i++ {
		v, err := sub.Next()
		require.NoError(t, err)
		assert.Equal(t, i, v)
	}

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("emit did not unblock after read")
	}
}

func TestStream_BlockNoLoss(t *testing.T) {
	t.Parallel()

	const total = 6
	s := dringbuf.NewStream[int]()
	sub := s.Subscribe(3, dringbuf.BackpressureStrategyBlock)

	got := make(chan []int)
	go func() {
		vals := make([]int, 0, total)
		for range total {
			v, err := sub.Next()
			if err != nil {
				t.Error(err)
				return
			}
			vals = append(vals, v)
		}
		got <- vals
	}()

	for i := 1; i <= total; i++ {
		s.Emit(i)
	}

	select {
	case vals := <-got:
		assert.Equal(t, []int{1, 2, 3, 4, 5, 6}, vals)
	case <-time.After(2 * time.Second):
		t.Fatal("reader did not consume all values")
	}
}

func TestStream_MultipleSubscribers(t *testing.T) {
	t.Parallel()

	s := dringbuf.NewStream[int]()
	a := s.Subscribe(5, dringbuf.BackpressureStrategyDropOldest)
	b := s.Subscribe(5, dringbuf.BackpressureStrategyDropOldest)

	for i := 1; i <= 3; i++ {
		s.Emit(i)
	}

	for _, sub := range []*dringbuf.Subscription[int]{a, b} {
		for i := 1; i <= 3; i++ {
			v, err := sub.Next()
			require.NoError(t, err)
			assert.Equal(t, i, v)
		}
	}
}

func TestStream_StreamCloseDrainsThenEOF(t *testing.T) {
	t.Parallel()

	s := dringbuf.NewStream[int]()
	sub := s.Subscribe(3, dringbuf.BackpressureStrategyBlock)

	s.Emit(1)
	s.Emit(2)
	s.Close()
	s.Emit(3) // ignored after Close

	for i := 1; i <= 2; i++ {
		v, err := sub.Next()
		require.NoError(t, err)
		assert.Equal(t, i, v)
	}

	_, err := sub.Next()
	assert.ErrorIs(t, err, io.EOF)

	late := s.Subscribe(3, dringbuf.BackpressureStrategyBlock)
	_, err = late.Next()
	assert.ErrorIs(t, err, io.EOF)
}

func TestStream_SubscriptionCloseDropsBuffered(t *testing.T) {
	t.Parallel()

	s := dringbuf.NewStream[int]()
	sub := s.Subscribe(3, dringbuf.BackpressureStrategyBlock)
	other := s.Subscribe(3, dringbuf.BackpressureStrategyBlock)

	s.Emit(1)
	sub.Close()
	s.Emit(2)

	_, err := sub.Next()
	assert.ErrorIs(t, err, io.EOF)

	// Other subscriptions are unaffected.
	for i := 1; i <= 2; i++ {
		v, err := other.Next()
		require.NoError(t, err)
		assert.Equal(t, i, v)
	}
}

func TestStream_CloseUnblocksNext(t *testing.T) {
	t.Parallel()

	s := dringbuf.NewStream[int]()
	sub := s.Subscribe(3, dringbuf.BackpressureStrategyBlock)

	got := make(chan error)
	go func() {
		_, err := sub.Next()
		got <- err
	}()

	select {
	case err := <-got:
		t.Fatalf("Next returned before close: %v", err)
	case <-time.After(100 * time.Millisecond):
	}

	sub.Close()

	select {
	case err := <-got:
		assert.ErrorIs(t, err, io.EOF)
	case <-time.After(time.Second):
		t.Fatal("Next did not unblock after close")
	}
}

func TestStream_EmitAfterCloseIsNoop(t *testing.T) {
	t.Parallel()

	s := dringbuf.NewStream[int]()
	sub := s.Subscribe(3, dringbuf.BackpressureStrategyDropOldest)

	sub.Close()
	s.Emit(1)

	_, err := sub.Next()
	assert.ErrorIs(t, err, io.EOF)
}

func TestStream_ConcurrentEmitAndClose(t *testing.T) {
	t.Parallel()

	s := dringbuf.NewStream[int]()

	const subscribers = 8
	subs := make([]*dringbuf.Subscription[int], 0, subscribers)
	for range subscribers {
		subs = append(subs, s.Subscribe(4, dringbuf.BackpressureStrategyDropNewest))
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := range 10_000 {
			s.Emit(i)
		}
	}()

	go func() {
		defer wg.Done()
		for _, sub := range subs {
			sub.Close()
		}
	}()

	wg.Wait()
}

func TestStream_CloseIdempotent(t *testing.T) {
	t.Parallel()

	s := dringbuf.NewStream[int]()
	sub := s.Subscribe(3, dringbuf.BackpressureStrategyBlock)

	sub.Close()
	sub.Close()

	_, err := sub.Next()
	assert.ErrorIs(t, err, io.EOF)
}

// Regression: a blocked Emit used to ignore Close and hang forever.
func TestStream_CloseUnblocksEmit(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name  string
		close func(*dringbuf.Stream[int], *dringbuf.Subscription[int])
	}{
		{"subscription", func(_ *dringbuf.Stream[int], sub *dringbuf.Subscription[int]) { sub.Close() }},
		{"stream", func(s *dringbuf.Stream[int], _ *dringbuf.Subscription[int]) { s.Close() }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			s := dringbuf.NewStream[int]()
			sub := s.Subscribe(1, dringbuf.BackpressureStrategyBlock)
			s.Emit(1)

			done := make(chan struct{})
			go func() {
				s.Emit(2) // blocks: buffer is full
				close(done)
			}()

			select {
			case <-done:
				t.Fatal("emit should block when buffer is full")
			case <-time.After(50 * time.Millisecond):
			}

			tc.close(s, sub)

			select {
			case <-done:
			case <-time.After(time.Second):
				t.Fatal("Emit did not return after Close")
			}
		})
	}
}

// Regression: producers and consumers used to share one sync.Cond and Signal,
// which could wake the wrong side and deadlock.
func TestStream_ManyProducersManyConsumers(t *testing.T) {
	t.Parallel()

	const (
		producers = 4
		consumers = 4
		per       = 2000
		trials    = 20
	)
	for range trials {
		s := dringbuf.NewStream[int]()
		sub := s.Subscribe(1, dringbuf.BackpressureStrategyBlock)

		var prod sync.WaitGroup
		for range producers {
			prod.Add(1)
			go func() {
				defer prod.Done()
				for i := range per {
					s.Emit(i)
				}
			}()
		}

		var mu sync.Mutex
		total := 0
		var cons sync.WaitGroup
		for c := range consumers {
			cons.Add(1)
			go func() {
				defer cons.Done()
				buf := make([]int, 3)
				for {
					var n int
					var err error
					switch c % 3 {
					case 0:
						_, err = sub.Next()
						n = 1
					case 1:
						n, err = sub.Read(buf)
					default:
						err = sub.ReadView(func(v []int) { n = len(v) })
					}
					if err != nil {
						return
					}
					mu.Lock()
					total += n
					mu.Unlock()
				}
			}()
		}

		done := make(chan struct{})
		go func() {
			prod.Wait()
			s.Close()
			cons.Wait()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("deadlock")
		}
		require.Equal(t, producers*per, total)
	}
}

func TestStream_Read(t *testing.T) {
	t.Parallel()

	s := dringbuf.NewStream[int]()
	sub := s.Subscribe(4, dringbuf.BackpressureStrategyDropOldest)
	for i := 1; i <= 6; i++ {
		s.Emit(i)
	}

	buf := make([]int, 3)
	n, err := sub.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, []int{3, 4, 5}, buf[:n])

	n, err = sub.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, []int{6}, buf[:n])

	n, err = sub.Read(nil)
	assert.NoError(t, err)
	assert.Zero(t, n)

	s.Close()
	n, err = sub.Read(buf)
	assert.ErrorIs(t, err, io.EOF)
	assert.Zero(t, n)
}

func TestStream_ReadView(t *testing.T) {
	t.Parallel()

	s := dringbuf.NewStream[int]()
	sub := s.Subscribe(3, dringbuf.BackpressureStrategyDropOldest)
	for i := 1; i <= 5; i++ {
		s.Emit(i)
	}

	var got []int
	require.NoError(t, sub.ReadView(func(v []int) { got = append(got, v...) }))
	assert.Equal(t, []int{3, 4, 5}, got)

	s.Emit(6)
	got = got[:0]
	require.NoError(t, sub.ReadView(func(v []int) { got = append(got, v...) }))
	assert.Equal(t, []int{6}, got)

	sub.Close()
	assert.ErrorIs(t, sub.ReadView(func([]int) { t.Fatal("fn called after Close") }), io.EOF)
}

func TestStream_NextContext(t *testing.T) {
	t.Parallel()

	s := dringbuf.NewStream[int]()
	sub := s.Subscribe(3, dringbuf.BackpressureStrategyBlock)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := sub.NextContext(ctx)
	assert.True(t, errors.Is(err, context.DeadlineExceeded), "got %v", err)

	// A buffered value wins over a done context.
	s.Emit(7)
	v, err := sub.NextContext(ctx)
	require.NoError(t, err)
	assert.Equal(t, 7, v)

	s.Emit(8)
	v, err = sub.NextContext(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 8, v)
}

func TestStream_SubscribeValidates(t *testing.T) {
	t.Parallel()

	s := dringbuf.NewStream[int]()
	assert.Panics(t, func() { s.Subscribe(0, dringbuf.BackpressureStrategyBlock) })
	assert.Panics(t, func() { s.Subscribe(3, dringbuf.BackpressureStrategy(99)) })
	assert.Panics(t, func() { s.Subscribe(3, dringbuf.BackpressureStrategy(-1)) })
}

func TestStream_ZeroAllocs(t *testing.T) {
	s := dringbuf.NewStream[int]()
	sub := s.Subscribe(16, dringbuf.BackpressureStrategyDropOldest)
	for range 8 {
		s.Subscribe(16, dringbuf.BackpressureStrategyDropOldest)
	}
	buf := make([]int, 4)
	view := func([]int) {}

	assert.Zero(t, testing.AllocsPerRun(100, func() { s.Emit(1) }))
	assert.Zero(t, testing.AllocsPerRun(100, func() { s.Emit(1); _, _ = sub.Next() }))
	assert.Zero(t, testing.AllocsPerRun(100, func() { s.Emit(1); _, _ = sub.Read(buf) }))
	assert.Zero(t, testing.AllocsPerRun(100, func() { s.Emit(1); _ = sub.ReadView(view) }))
}
