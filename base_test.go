package dringbuf_test

import (
	"dringbuf"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRingBufferBasic(t *testing.T) {
	assert := assert.New(t)
	t.Parallel()

	t.Run("basic_tests", func(t *testing.T) {
		t.Parallel()
		rbuf := dringbuf.NewRingBuffer[int](3)

		assert.Equal(0, rbuf.Len())
		assert.Equal([]int{}, rbuf.Last(3))
		assert.Panics(func() { rbuf.Last(4) })

		rbuf.Append(1)
		assert.Equal(1, rbuf.Len())
		assert.Equal(1, rbuf.At(0))
		assert.Equal([]int{1}, rbuf.Last(1))
		assert.Equal([]int{1}, rbuf.Last(3))

		rbuf.Append(2)
		assert.Equal(2, rbuf.Len())
		assert.Equal(1, rbuf.At(0))
		assert.Equal(2, rbuf.At(1))
		assert.Equal([]int{2}, rbuf.Last(1))
		assert.Equal([]int{1, 2}, rbuf.Last(2))
		assert.Equal([]int{1, 2}, rbuf.Last(3))

		rbuf.Append(3)
		assert.Equal(3, rbuf.Len())
		assert.Equal(3, rbuf.At(2))
		assert.Equal(2, rbuf.At(1))
		assert.Equal([]int{2, 3}, rbuf.Last(2))
		assert.Equal([]int{1, 2, 3}, rbuf.Last(3))

		// Overwrite old
		rbuf.Append(4)
		assert.Equal(3, rbuf.Len()) // remains 3
		assert.Equal(2, rbuf.At(0))
		assert.Equal([]int{2, 3, 4}, rbuf.Last(3))
		assert.Equal(4, rbuf.At(2))

		rbuf.Append(5)
		rbuf.Append(6)
		assert.Equal(6, rbuf.At(2))
		assert.Equal([]int{4, 5, 6}, rbuf.Last(3))
	})
	t.Run("test_large", func(t *testing.T) {
		t.Parallel()
		bufferSize := 1000000
		rbuf := dringbuf.NewRingBuffer[int](bufferSize)

		assert.Equal(0, rbuf.Len())
		assert.Equal(bufferSize, rbuf.Cap())
		// fill slice with values
		for i := range bufferSize {
			rbuf.Append(i)
		}
		assert.Equal(bufferSize, rbuf.Len())
		t.Log(rbuf.Last(10))
	})
}
