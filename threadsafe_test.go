package dringbuf

import (
	"testing"
	"testing/synctest"

	"github.com/stretchr/testify/assert"
)

func TestThreadSafe(t *testing.T) {
	assert := assert.New(t)

	// Base test to ensure same behavior with base implementation
	t.Run("Compatibility", func(t *testing.T) {
		t.Parallel()
		tsbuf := NewThreadSafeRingBuffer[int](3)

		assert.Equal(0, tsbuf.Len())
		assert.Equal([]int{}, tsbuf.Last(3))
		assert.Panics(func() { tsbuf.Last(4) })

		tsbuf.Append(1)
		assert.Equal(1, tsbuf.Len())
		assert.Equal(1, tsbuf.At(0))
		assert.Equal([]int{1}, tsbuf.Last(1))
		assert.Equal([]int{1}, tsbuf.Last(3))

		tsbuf.Append(2)
		assert.Equal(2, tsbuf.Len())
		assert.Equal(1, tsbuf.At(0))
		assert.Equal(2, tsbuf.At(1))
		assert.Equal([]int{2}, tsbuf.Last(1))
		assert.Equal([]int{1, 2}, tsbuf.Last(2))
		assert.Equal([]int{1, 2}, tsbuf.Last(3))

		tsbuf.Append(3)
		assert.Equal(3, tsbuf.Len())
		assert.Equal(3, tsbuf.At(2))
		assert.Equal(2, tsbuf.At(1))
		assert.Equal([]int{2, 3}, tsbuf.Last(2))
		assert.Equal([]int{1, 2, 3}, tsbuf.Last(3))

		// Overwrite old
		tsbuf.Append(4)
		assert.Equal(3, tsbuf.Len()) // remains 3
		assert.Equal(2, tsbuf.At(0))
		assert.Equal([]int{2, 3, 4}, tsbuf.Last(3))
		assert.Equal(4, tsbuf.At(2))

		tsbuf.Append(5)
		tsbuf.Append(6)
		assert.Equal(6, tsbuf.At(2))
		assert.Equal([]int{4, 5, 6}, tsbuf.Last(3))
	})

	t.Run("WriteOnBorrowed", func(t *testing.T) {
		t.Parallel()
		synctest.Test(t, func(t *testing.T) {
			tsbuf := NewThreadSafeRingBuffer[int](3)

			for i := range 3 {
				tsbuf.Append(i + 1)
			}
			assert.Equal([]int{1, 2, 3}, tsbuf.Last(3))

			res, release := tsbuf.Borrow(3)
			assert.Equal([]int{1, 2, 3}, res)

			resCh := make(chan string, 3)
			writerReady := make(chan struct{})

			go func() {
				resCh <- "try_write"
				close(writerReady) // signal we are about to attempt Append
				tsbuf.Append(4)
				resCh <- "write"
			}()

			go func() {
				<-writerReady // wait until writer is about to append
				release()
				resCh <- "release"
			}()

			synctest.Wait()

			assert.Equal("try_write", <-resCh)
			assert.Equal("release", <-resCh)
			assert.Equal("write", <-resCh)
		})
	})
}
