package dringbuf_test

import (
	"dringbuf"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRingBuffer_Contract(t *testing.T) {
	t.Parallel()
	runRingBufferContractTests(t, func(size int) dringbuf.RingBuffer[int] {
		return dringbuf.NewRingBuffer[int](size)
	})
}

func TestRingBuffer_ClearAndReuse(t *testing.T) {
	t.Parallel()
	runRingBufferClearAndReuseTests(t, func(size int) dringbuf.RingBuffer[int] {
		return dringbuf.NewRingBuffer[int](size)
	})
}

func TestRingBuffer_Panics(t *testing.T) {
	t.Parallel()
	runRingBufferCommonPanicTests(t, func(size int) dringbuf.RingBuffer[int] {
		return dringbuf.NewRingBuffer[int](size)
	})
}

func TestRingBuffer_LastReturnsCopy(t *testing.T) {
	t.Parallel()

	rb := dringbuf.NewRingBuffer[int](3)
	rb.Append(1)
	rb.Append(2)
	rb.Append(3)

	view := rb.Last(3)
	require.Equal(t, []int{1, 2, 3}, view)

	// Basic implementation copies in Last, so mutation must not affect the buffer.
	view[0] = 99

	assert.Equal(t, []int{1, 2, 3}, rb.Last(3))
	assert.Equal(t, 1, rb.At(0))

	// Also verify copy behavior across a wrap-around boundary.
	rb.Append(4)
	rb.Append(5)
	rb.Append(6)
	require.Equal(t, []int{4, 5, 6}, rb.Last(3))

	wrapped := rb.Last(3)
	wrapped[2] = 99
	assert.Equal(t, []int{4, 5, 6}, rb.Last(3))
}

func TestRingBuffer_LargeCapacitySmoke(t *testing.T) {
	t.Parallel()

	const size = 100_000
	rb := dringbuf.NewRingBuffer[int](size)

	assert.Equal(t, 0, rb.Len())
	assert.Equal(t, size, rb.Cap())

	for i := range size {
		rb.Append(i)
	}

	assert.Equal(t, size, rb.Len())
	assert.Equal(t, []int{size - 3, size - 2, size - 1}, rb.Last(3))
}
