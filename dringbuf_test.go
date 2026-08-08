package dringbuf_test

import (
	"dringbuf"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDRingBuffer_Contract(t *testing.T) {
	t.Parallel()
	runRingBufferContractTests(t, dringbuf.NewDRingBuffer[int])
}

func TestDRingBuffer_ClearAndReuse(t *testing.T) {
	t.Parallel()
	runRingBufferClearAndReuseTests(t, dringbuf.NewDRingBuffer[int])
}

func TestDRingBuffer_Panics(t *testing.T) {
	t.Parallel()
	runRingBufferCommonPanicTests(t, dringbuf.NewDRingBuffer[int])

	t.Run("constructor overflow panics", func(t *testing.T) {
		t.Parallel()

		maxInt := int(^uint(0) >> 1)
		overflowSize := maxInt/2 + 1

		assert.Panics(t, func() { dringbuf.NewDRingBuffer[int](overflowSize) })
	})
}

func TestDRingBuffer_LastAliasingBehavior(t *testing.T) {
	t.Parallel()

	rb := dringbuf.NewDRingBuffer[int](3)
	rb.Append(1)
	rb.Append(2)
	rb.Append(3)

	view := rb.Last(3)
	require.Equal(t, []int{1, 2, 3}, view)

	// Double-sized implementation returns a view to internal storage.
	view[0] = 99

	assert.Equal(t, 99, rb.At(0))
	assert.Equal(t, []int{99, 2, 3}, rb.Last(3))
}

func TestDRingBuffer_LargeCapacitySmoke(t *testing.T) {
	t.Parallel()

	const size = 100_000
	rb := dringbuf.NewDRingBuffer[int](size)

	assert.Equal(t, 0, rb.Len())
	assert.Equal(t, size, rb.Cap())

	for i := range size {
		rb.Append(i)
	}

	assert.Equal(t, size, rb.Len())
	assert.Equal(t, []int{size - 3, size - 2, size - 1}, rb.Last(3))
}
