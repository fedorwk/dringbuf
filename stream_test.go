package dringbuf_test

import (
	"dringbuf"
	"io"
	"sync"
	"testing"
	"time"

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

func TestStream_CloseReturnsEOFAfterDrain(t *testing.T) {
	t.Parallel()

	s := dringbuf.NewStream[int]()
	sub := s.Subscribe(3, dringbuf.BackpressureStrategyBlock)

	s.Emit(1)
	s.Emit(2)
	sub.Close()

	for i := 1; i <= 2; i++ {
		v, err := sub.Next()
		require.NoError(t, err)
		assert.Equal(t, i, v)
	}

	_, err := sub.Next()
	assert.ErrorIs(t, err, io.EOF)
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
