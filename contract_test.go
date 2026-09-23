package dringbuf_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type ringOps[T any] interface {
	Append(elem T)
	Len() int
	Cap() int
	At(idx int) T
	Get(idx int) (T, bool)
	Last() (T, bool)
	Tail(n int) []T
	Clear()
}

type ringBufferFactory[B ringOps[int]] func(size int) B

func runRingBufferContractTests[B ringOps[int]](t *testing.T, newBuf ringBufferFactory[B]) {
	t.Helper()

	type testCase struct {
		name       string
		ops        []int
		wantLen    int
		wantAt     []int
		wantLast   int
		wantLastOK bool
		wantTail1  []int
		wantTail2  []int
		wantTail3  []int
	}

	tests := []testCase{
		{
			name:       "empty",
			ops:        nil,
			wantLen:    0,
			wantAt:     nil,
			wantLastOK: false,
			wantTail1:  []int{},
			wantTail2:  []int{},
			wantTail3:  []int{},
		},
		{
			name:       "single append",
			ops:        []int{10},
			wantLen:    1,
			wantAt:     []int{10},
			wantLast:   10,
			wantLastOK: true,
			wantTail1:  []int{10},
			wantTail2:  []int{10},
			wantTail3:  []int{10},
		},
		{
			name:       "partially filled",
			ops:        []int{10, 20},
			wantLen:    2,
			wantAt:     []int{10, 20},
			wantLast:   20,
			wantLastOK: true,
			wantTail1:  []int{20},
			wantTail2:  []int{10, 20},
			wantTail3:  []int{10, 20},
		},
		{
			name:       "full",
			ops:        []int{1, 2, 3},
			wantLen:    3,
			wantAt:     []int{1, 2, 3},
			wantLast:   3,
			wantLastOK: true,
			wantTail1:  []int{3},
			wantTail2:  []int{2, 3},
			wantTail3:  []int{1, 2, 3},
		},
		{
			name:       "overwrite once",
			ops:        []int{1, 2, 3, 4},
			wantLen:    3,
			wantAt:     []int{2, 3, 4},
			wantLast:   4,
			wantLastOK: true,
			wantTail1:  []int{4},
			wantTail2:  []int{3, 4},
			wantTail3:  []int{2, 3, 4},
		},
		{
			name:       "overwrite many",
			ops:        []int{1, 2, 3, 4, 5, 6},
			wantLen:    3,
			wantAt:     []int{4, 5, 6},
			wantLast:   6,
			wantLastOK: true,
			wantTail1:  []int{6},
			wantTail2:  []int{5, 6},
			wantTail3:  []int{4, 5, 6},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rb := newBuf(3)
			assert.Equal(t, 3, rb.Cap())

			for _, v := range tc.ops {
				rb.Append(v)
			}

			assert.Equal(t, tc.wantLen, rb.Len())
			assert.Equal(t, tc.wantTail1, rb.Tail(1))
			assert.Equal(t, tc.wantTail2, rb.Tail(2))
			assert.Equal(t, tc.wantTail3, rb.Tail(3))

			gotLast, ok := rb.Last()
			assert.Equal(t, tc.wantLastOK, ok)
			assert.Equal(t, tc.wantLast, gotLast)

			for i, want := range tc.wantAt {
				assert.Equal(t, want, rb.At(i))
				got, ok := rb.Get(i)
				assert.True(t, ok)
				assert.Equal(t, want, got)
			}

			// One past the last element is always out of range.
			_, ok = rb.Get(tc.wantLen)
			assert.False(t, ok)
		})
	}
}

func runRingBufferCommonPanicTests[B ringOps[int]](t *testing.T, newBuf ringBufferFactory[B]) {
	t.Helper()

	t.Run("at out of bounds panics", func(t *testing.T) {
		t.Parallel()
		rb := newBuf(3)
		rb.Append(1)

		assert.Panics(t, func() { rb.At(-1) })
		assert.Panics(t, func() { rb.At(1) })
		assert.Panics(t, func() { rb.At(3) })
		assert.Panics(t, func() { rb.At(100) })
	})

	t.Run("get out of bounds returns false", func(t *testing.T) {
		t.Parallel()
		rb := newBuf(3)
		rb.Append(1)

		_, ok := rb.Get(-1)
		assert.False(t, ok)
		_, ok = rb.Get(1)
		assert.False(t, ok)
		_, ok = rb.Get(3)
		assert.False(t, ok)

		v, ok := rb.Get(0)
		assert.True(t, ok)
		assert.Equal(t, 1, v)
	})

	t.Run("tail greater than capacity panics", func(t *testing.T) {
		t.Parallel()
		rb := newBuf(3)

		assert.Panics(t, func() { rb.Tail(-1) })
		assert.Panics(t, func() { rb.Tail(4) })
	})

	t.Run("non-positive size panics", func(t *testing.T) {
		t.Parallel()

		assert.Panics(t, func() { newBuf(0) })
		assert.Panics(t, func() { newBuf(-1) })
	})
}

func runRingBufferClearAndReuseTests[B ringOps[int]](t *testing.T, newBuf ringBufferFactory[B]) {
	t.Helper()

	rb := newBuf(3)
	rb.Append(1)
	rb.Append(2)
	rb.Append(3)

	rb.Clear()

	assert.Equal(t, 0, rb.Len())
	assert.Equal(t, 3, rb.Cap())
	assert.Equal(t, []int{}, rb.Tail(0))
	assert.Equal(t, []int{}, rb.Tail(1))
	assert.Equal(t, []int{}, rb.Tail(3))

	_, ok := rb.Last()
	assert.False(t, ok)

	rb.Append(7)
	rb.Append(8)

	assert.Equal(t, 2, rb.Len())
	assert.Equal(t, 7, rb.At(0))
	assert.Equal(t, 8, rb.At(1))
	assert.Equal(t, []int{7, 8}, rb.Tail(3))

	v, ok := rb.Last()
	assert.True(t, ok)
	assert.Equal(t, 8, v)
}
