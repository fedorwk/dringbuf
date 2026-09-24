package dringbuf_test

import (
	"testing"

	"github.com/fedorwk/dringbuf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDoubleRingBuffer_Contract(t *testing.T) {
	t.Parallel()
	runRingBufferContractTests(t, dringbuf.NewDoubleRingBuffer[int])
}

func TestDoubleRingBuffer_ClearAndReuse(t *testing.T) {
	t.Parallel()
	runRingBufferClearAndReuseTests(t, dringbuf.NewDoubleRingBuffer[int])
}

func TestDoubleRingBuffer_Panics(t *testing.T) {
	t.Parallel()
	runRingBufferCommonPanicTests(t, dringbuf.NewDoubleRingBuffer[int])

	t.Run("constructor overflow panics", func(t *testing.T) {
		t.Parallel()

		maxInt := int(^uint(0) >> 1)
		overflowSize := maxInt/2 + 1

		assert.Panics(t, func() { dringbuf.NewDoubleRingBuffer[int](overflowSize) })
	})
}

func TestDoubleRingBuffer_LastAliasingBehavior(t *testing.T) {
	t.Parallel()

	rb := dringbuf.NewDoubleRingBuffer[int](3)
	rb.Append(1)
	rb.Append(2)
	rb.Append(3)

	view := rb.Tail(3)
	require.Equal(t, []int{1, 2, 3}, view)

	// Double-sized implementation returns a view to internal storage.
	view[0] = 99

	assert.Equal(t, 99, rb.At(0))
	assert.Equal(t, []int{99, 2, 3}, rb.Tail(3))
}

func TestDoubleRingBuffer_Last(t *testing.T) {
	t.Parallel()

	rb := dringbuf.NewDoubleRingBuffer[int](3)

	_, ok := rb.Last()
	assert.False(t, ok)

	rb.Append(1)
	v, ok := rb.Last()
	require.True(t, ok)
	assert.Equal(t, 1, v)

	rb.Append(2)
	rb.Append(3)
	rb.Append(4)

	v, ok = rb.Last()
	require.True(t, ok)
	assert.Equal(t, 4, v)
}

func TestDoubleRingBuffer_LargeCapacitySmoke(t *testing.T) {
	t.Parallel()

	const size = 100_000
	rb := dringbuf.NewDoubleRingBuffer[int](size)

	assert.Equal(t, 0, rb.Len())
	assert.Equal(t, size, rb.Cap())

	for i := range size {
		rb.Append(i)
	}

	assert.Equal(t, size, rb.Len())
	assert.Equal(t, []int{size - 3, size - 2, size - 1}, rb.Tail(3))
}
