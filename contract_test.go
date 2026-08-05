package dringbuf_test

import (
	"testing"

	"dringbuf"

	"github.com/stretchr/testify/assert"
)

type ringBufferFactory func(size int) dringbuf.RingBuffer[int]

func runRingBufferContractTests(t *testing.T, newBuf ringBufferFactory) {
	t.Helper()

	type testCase struct {
		name      string
		ops       []int
		wantLen   int
		wantAt    []int
		wantLast1 []int
		wantLast2 []int
		wantLast3 []int
	}

	tests := []testCase{
		{
			name:      "empty",
			ops:       nil,
			wantLen:   0,
			wantAt:    nil,
			wantLast1: []int{},
			wantLast2: []int{},
			wantLast3: []int{},
		},
		{
			name:      "single append",
			ops:       []int{10},
			wantLen:   1,
			wantAt:    []int{10},
			wantLast1: []int{10},
			wantLast2: []int{10},
			wantLast3: []int{10},
		},
		{
			name:      "partially filled",
			ops:       []int{10, 20},
			wantLen:   2,
			wantAt:    []int{10, 20},
			wantLast1: []int{20},
			wantLast2: []int{10, 20},
			wantLast3: []int{10, 20},
		},
		{
			name:      "full",
			ops:       []int{1, 2, 3},
			wantLen:   3,
			wantAt:    []int{1, 2, 3},
			wantLast1: []int{3},
			wantLast2: []int{2, 3},
			wantLast3: []int{1, 2, 3},
		},
		{
			name:      "overwrite once",
			ops:       []int{1, 2, 3, 4},
			wantLen:   3,
			wantAt:    []int{2, 3, 4},
			wantLast1: []int{4},
			wantLast2: []int{3, 4},
			wantLast3: []int{2, 3, 4},
		},
		{
			name:      "overwrite many",
			ops:       []int{1, 2, 3, 4, 5, 6},
			wantLen:   3,
			wantAt:    []int{4, 5, 6},
			wantLast1: []int{6},
			wantLast2: []int{5, 6},
			wantLast3: []int{4, 5, 6},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rb := newBuf(3)
			assert.Equal(t, 3, rb.Cap())

			for _, v := range tc.ops {
				rb.Append(v)
			}

			assert.Equal(t, tc.wantLen, rb.Len())
			assert.Equal(t, tc.wantLast1, rb.Last(1))
			assert.Equal(t, tc.wantLast2, rb.Last(2))
			assert.Equal(t, tc.wantLast3, rb.Last(3))

			for i, want := range tc.wantAt {
				assert.Equal(t, want, rb.At(i))
			}
		})
	}
}

func runRingBufferCommonPanicTests(t *testing.T, newBuf ringBufferFactory) {
	t.Helper()

	t.Run("at out of bounds panics", func(t *testing.T) {
		t.Parallel()
		rb := newBuf(3)
		rb.Append(1)

		assert.Panics(t, func() { rb.At(3) })
		assert.Panics(t, func() { rb.At(100) })
	})

	t.Run("last greater than capacity panics", func(t *testing.T) {
		t.Parallel()
		rb := newBuf(3)

		assert.Panics(t, func() { rb.Last(4) })
	})
}

func runRingBufferClearAndReuseTests(t *testing.T, newBuf ringBufferFactory) {
	t.Helper()

	rb := newBuf(3)
	rb.Append(1)
	rb.Append(2)
	rb.Append(3)

	rb.Clear()

	assert.Equal(t, 0, rb.Len())
	assert.Equal(t, 3, rb.Cap())
	assert.Equal(t, []int{}, rb.Last(0))
	assert.Equal(t, []int{}, rb.Last(1))
	assert.Equal(t, []int{}, rb.Last(3))

	rb.Append(7)
	rb.Append(8)

	assert.Equal(t, 2, rb.Len())
	assert.Equal(t, 7, rb.At(0))
	assert.Equal(t, 8, rb.At(1))
	assert.Equal(t, []int{7, 8}, rb.Last(3))
}
