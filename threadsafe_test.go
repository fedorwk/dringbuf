package dringbuf_test

import (
	"sync"
	"testing"
	"time"

	"github.com/fedorwk/dringbuf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestThreadSafeRingBuffer_Contract(t *testing.T) {
	t.Parallel()

	runRingBufferContractTests(t, dringbuf.NewThreadSafeRingBuffer[int])
}

func TestThreadSafeRingBuffer_ClearAndReuse(t *testing.T) {
	t.Parallel()

	runRingBufferClearAndReuseTests(t, dringbuf.NewThreadSafeRingBuffer[int])
}

func TestThreadSafeRingBuffer_Panics(t *testing.T) {
	t.Parallel()

	runRingBufferCommonPanicTests(t, dringbuf.NewThreadSafeRingBuffer[int])
}

func TestThreadSafeRingBuffer_TailReturnsCopy(t *testing.T) {
	t.Parallel()

	rb := dringbuf.NewThreadSafeRingBuffer[int](3)
	rb.Append(1)
	rb.Append(2)
	rb.Append(3)

	last := rb.Tail(3)
	require.Equal(t, []int{1, 2, 3}, last)

	last[0] = 99

	assert.Equal(t, []int{1, 2, 3}, rb.Tail(3))
	assert.Equal(t, 1, rb.At(0))
}

func TestThreadSafeRingBuffer_LastAndGet(t *testing.T) {
	t.Parallel()

	rb := dringbuf.NewThreadSafeRingBuffer[int](3)

	_, ok := rb.Last()
	assert.False(t, ok)
	_, ok = rb.Get(0)
	assert.False(t, ok)

	rb.Append(1)
	rb.Append(2)

	v, ok := rb.Last()
	require.True(t, ok)
	assert.Equal(t, 2, v)

	v, ok = rb.Get(0)
	require.True(t, ok)
	assert.Equal(t, 1, v)

	_, ok = rb.Get(2)
	assert.False(t, ok)
}

func TestThreadSafeRingBuffer_ViewReturnsCopy(t *testing.T) {
	t.Parallel()

	rb := dringbuf.NewThreadSafeRingBuffer[int](3)
	rb.Append(1)
	rb.Append(2)
	rb.Append(3)

	rb.View(3, func(view []int) {
		require.Equal(t, []int{1, 2, 3}, view)
		view[0] = 42
	})

	// Single-array View copies, so mutation is not visible.
	assert.Equal(t, 1, rb.At(0))
	assert.Equal(t, []int{1, 2, 3}, rb.Tail(3))
}

func TestThreadSafeRingBuffer_ViewBlocksWriterUntilReturn(t *testing.T) {
	t.Parallel()

	rb := dringbuf.NewThreadSafeRingBuffer[int](3)
	rb.Append(1)
	rb.Append(2)
	rb.Append(3)

	writerDone := make(chan struct{})

	rb.View(3, func([]int) {
		go func() {
			rb.Append(4) // must block until View returns
			close(writerDone)
		}()

		// While the read lock is held, the writer must still be blocked.
		select {
		case <-writerDone:
			t.Fatal("writer completed while View was running")
		case <-time.After(50 * time.Millisecond):
		}
	})

	select {
	case <-writerDone:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("writer did not complete after View returned")
	}

	assert.Equal(t, []int{2, 3, 4}, rb.Tail(3))
}

func TestThreadSafeRingBuffer_ConcurrentAccessCompletes(t *testing.T) {
	const (
		capacity      = 32
		writers       = 2
		readers       = 2
		perWriterOps  = 80
		perReaderOps  = 120
		maxWrittenVal = writers * perWriterOps
	)

	const timeout = 8 * time.Second

	rb := dringbuf.NewThreadSafeRingBuffer[int](capacity)

	var wg sync.WaitGroup
	wg.Add(writers + readers)

	start := make(chan struct{})

	for w := range writers {
		go func() {
			defer wg.Done()
			<-start
			base := w * perWriterOps
			for i := range perWriterOps {
				rb.Append(base + i)
			}
		}()
	}

	for range readers {
		go func() {
			defer wg.Done()
			<-start
			for range perReaderOps {
				_ = rb.Len()
				_ = rb.Cap()
				_ = rb.Tail(0)
			}
		}()
	}

	close(start)

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(timeout):
		t.Fatal("concurrent access timed out")
	}

	assert.Equal(t, capacity, rb.Cap())
	assert.LessOrEqual(t, rb.Len(), capacity)

	last := rb.Tail(rb.Len())
	assert.Len(t, last, rb.Len())

	for _, v := range last {
		assert.GreaterOrEqual(t, v, 0)
		assert.Less(t, v, maxWrittenVal)
	}
}

func TestThreadSafeDoubleRingBuffer_Contract(t *testing.T) {
	t.Parallel()

	runRingBufferContractTests(t, dringbuf.NewThreadSafeDoubleRingBuffer[int])
}

func TestThreadSafeDoubleRingBuffer_ClearAndReuse(t *testing.T) {
	t.Parallel()

	runRingBufferClearAndReuseTests(t, dringbuf.NewThreadSafeDoubleRingBuffer[int])
}

func TestThreadSafeDoubleRingBuffer_Panics(t *testing.T) {
	t.Parallel()

	runRingBufferCommonPanicTests(t, dringbuf.NewThreadSafeDoubleRingBuffer[int])
}

func TestThreadSafeDoubleRingBuffer_TailReturnsCopy(t *testing.T) {
	t.Parallel()

	rb := dringbuf.NewThreadSafeDoubleRingBuffer[int](3)
	rb.Append(1)
	rb.Append(2)
	rb.Append(3)

	last := rb.Tail(3)
	require.Equal(t, []int{1, 2, 3}, last)

	last[0] = 99

	assert.Equal(t, []int{1, 2, 3}, rb.Tail(3))
	assert.Equal(t, 1, rb.At(0))
}

func TestThreadSafeDoubleRingBuffer_LastAndGet(t *testing.T) {
	t.Parallel()

	rb := dringbuf.NewThreadSafeDoubleRingBuffer[int](3)

	_, ok := rb.Last()
	assert.False(t, ok)
	_, ok = rb.Get(0)
	assert.False(t, ok)

	rb.Append(1)
	rb.Append(2)

	v, ok := rb.Last()
	require.True(t, ok)
	assert.Equal(t, 2, v)

	v, ok = rb.Get(0)
	require.True(t, ok)
	assert.Equal(t, 1, v)

	_, ok = rb.Get(2)
	assert.False(t, ok)
}

func TestThreadSafeDoubleRingBuffer_ViewReturnsAliasedView(t *testing.T) {
	t.Parallel()

	rb := dringbuf.NewThreadSafeDoubleRingBuffer[int](3)
	rb.Append(1)
	rb.Append(2)
	rb.Append(3)

	rb.View(3, func(view []int) {
		require.Equal(t, []int{1, 2, 3}, view)
		view[0] = 42
	})

	// Double-sized View exposes internal memory, so mutation is visible.
	assert.Equal(t, 42, rb.At(0))
	assert.Equal(t, []int{42, 2, 3}, rb.Tail(3))
}

func TestThreadSafeDoubleRingBuffer_ViewBlocksWriterUntilReturn(t *testing.T) {
	t.Parallel()

	rb := dringbuf.NewThreadSafeDoubleRingBuffer[int](3)
	rb.Append(1)
	rb.Append(2)
	rb.Append(3)

	writerDone := make(chan struct{})

	rb.View(3, func([]int) {
		go func() {
			rb.Append(4) // must block until View returns
			close(writerDone)
		}()

		// While the read lock is held, the writer must still be blocked.
		select {
		case <-writerDone:
			t.Fatal("writer completed while View was running")
		case <-time.After(50 * time.Millisecond):
		}
	})

	select {
	case <-writerDone:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("writer did not complete after View returned")
	}

	assert.Equal(t, []int{2, 3, 4}, rb.Tail(3))
}

func TestThreadSafeDoubleRingBuffer_ConcurrentAccessCompletes(t *testing.T) {
	const (
		capacity      = 32
		writers       = 2
		readers       = 2
		perWriterOps  = 80
		perReaderOps  = 120
		maxWrittenVal = writers * perWriterOps
	)

	const timeout = 8 * time.Second

	rb := dringbuf.NewThreadSafeDoubleRingBuffer[int](capacity)

	var wg sync.WaitGroup
	wg.Add(writers + readers)

	start := make(chan struct{})

	for w := range writers {
		go func() {
			defer wg.Done()
			<-start
			base := w * perWriterOps
			for i := range perWriterOps {
				rb.Append(base + i)
			}
		}()
	}

	for range readers {
		go func() {
			defer wg.Done()
			<-start
			for range perReaderOps {
				_ = rb.Len()
				_ = rb.Cap()
				_ = rb.Tail(0)
			}
		}()
	}

	close(start)

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(timeout):
		t.Fatal("concurrent access timed out")
	}

	assert.Equal(t, capacity, rb.Cap())
	assert.LessOrEqual(t, rb.Len(), capacity)

	last := rb.Tail(rb.Len())
	assert.Len(t, last, rb.Len())

	for _, v := range last {
		assert.GreaterOrEqual(t, v, 0)
		assert.Less(t, v, maxWrittenVal)
	}
}

func TestThreadSafe_PanicInViewReleasesLock(t *testing.T) {
	t.Parallel()

	rb := dringbuf.NewThreadSafeDoubleRingBuffer[int](3)
	rb.Append(1)

	assert.Panics(t, func() { rb.View(1, func([]int) { panic("boom") }) })
	assert.Panics(t, func() { rb.View(4, func([]int) {}) })
	assert.Panics(t, func() { rb.Tail(-1) })
	assert.Panics(t, func() { rb.AppendTail(nil, 4) })

	done := make(chan struct{})
	go func() {
		rb.Append(2)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("buffer stayed locked after a panic")
	}
	assert.Equal(t, []int{1, 2}, rb.Tail(3))
}

func TestThreadSafe_ConcurrentViews(t *testing.T) {
	t.Parallel()

	rb := dringbuf.NewThreadSafeDoubleRingBuffer[int](3)
	rb.Append(1)

	// Two Views must be able to hold the read lock at the same time.
	inside := make(chan struct{})
	done := make(chan struct{})
	go func() {
		rb.View(1, func([]int) {
			close(inside)
			<-done
		})
	}()
	<-inside

	second := make(chan struct{})
	go func() {
		rb.View(1, func([]int) {})
		close(second)
	}()
	select {
	case <-second:
	case <-time.After(time.Second):
		t.Fatal("second View blocked behind the first")
	}
	close(done)
}

func TestThreadSafe_ZeroAllocs(t *testing.T) {
	rb := dringbuf.NewThreadSafeDoubleRingBuffer[int](8)
	for i := range 8 {
		rb.Append(i)
	}
	dst := make([]int, 0, 8)
	sum := 0
	view := func(v []int) { sum += len(v) }

	assert.Zero(t, testing.AllocsPerRun(100, func() { rb.Append(1) }))
	assert.Zero(t, testing.AllocsPerRun(100, func() { rb.View(4, view) }))
	assert.Zero(t, testing.AllocsPerRun(100, func() { dst = rb.AppendTail(dst[:0], 4) }))
	assert.Equal(t, 1.0, testing.AllocsPerRun(100, func() { _ = rb.Tail(4) }))

	ring := dringbuf.NewThreadSafeRingBuffer[int](8)
	for i := range 8 {
		ring.Append(i)
	}
	assert.Zero(t, testing.AllocsPerRun(100, func() { dst = ring.AppendTail(dst[:0], 8) }))
	assert.Equal(t, 1.0, testing.AllocsPerRun(100, func() { _ = ring.Tail(8) }))
}
