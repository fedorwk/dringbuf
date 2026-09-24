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

func TestThreadSafeRingBuffer_BorrowReturnsCopy(t *testing.T) {
	t.Parallel()

	rb := dringbuf.NewThreadSafeRingBuffer[int](3)
	rb.Append(1)
	rb.Append(2)
	rb.Append(3)

	view, release := rb.Borrow(3)
	require.Equal(t, []int{1, 2, 3}, view)

	view[0] = 42
	release()

	// Single-array sync Borrow copies, so mutation is not visible.
	assert.Equal(t, 1, rb.At(0))
	assert.Equal(t, []int{1, 2, 3}, rb.Tail(3))
}

func TestThreadSafeRingBuffer_BorrowBlocksWriterUntilRelease(t *testing.T) {
	t.Parallel()

	rb := dringbuf.NewThreadSafeRingBuffer[int](3)
	rb.Append(1)
	rb.Append(2)
	rb.Append(3)

	_, release := rb.Borrow(3)

	writerStarted := make(chan struct{})
	writerDone := make(chan struct{})

	go func() {
		close(writerStarted)
		rb.Append(4) // must block until release()
		close(writerDone)
	}()

	select {
	case <-writerStarted:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("writer did not start")
	}

	// While read lock is held, writer must still be blocked.
	select {
	case <-writerDone:
		t.Fatal("writer completed before borrow release")
	case <-time.After(50 * time.Millisecond):
	}

	release()

	select {
	case <-writerDone:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("writer did not complete after borrow release")
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

	timeout := 8 * time.Second
	if testing.Short() {
		timeout = 12 * time.Second
	}

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

func TestThreadSafeDoubleRingBuffer_BorrowReturnsAliasedView(t *testing.T) {
	t.Parallel()

	rb := dringbuf.NewThreadSafeDoubleRingBuffer[int](3)
	rb.Append(1)
	rb.Append(2)
	rb.Append(3)

	view, release := rb.Borrow(3)
	require.Equal(t, []int{1, 2, 3}, view)

	view[0] = 42
	release()

	// Double-sized Borrow exposes internal memory, so mutation is visible.
	assert.Equal(t, 42, rb.At(0))
	assert.Equal(t, []int{42, 2, 3}, rb.Tail(3))
}

func TestThreadSafeDoubleRingBuffer_BorrowBlocksWriterUntilRelease(t *testing.T) {
	t.Parallel()

	rb := dringbuf.NewThreadSafeDoubleRingBuffer[int](3)
	rb.Append(1)
	rb.Append(2)
	rb.Append(3)

	_, release := rb.Borrow(3)

	writerStarted := make(chan struct{})
	writerDone := make(chan struct{})

	go func() {
		close(writerStarted)
		rb.Append(4) // must block until release()
		close(writerDone)
	}()

	select {
	case <-writerStarted:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("writer did not start")
	}

	// While read lock is held, writer must still be blocked.
	select {
	case <-writerDone:
		t.Fatal("writer completed before borrow release")
	case <-time.After(50 * time.Millisecond):
	}

	release()

	select {
	case <-writerDone:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("writer did not complete after borrow release")
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

	timeout := 8 * time.Second
	if testing.Short() {
		timeout = 12 * time.Second
	}

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
