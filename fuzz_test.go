package dringbuf_test

import (
	"slices"
	"testing"

	"github.com/fedorwk/dringbuf"
)

// FuzzBuffers runs a random sequence of operations against every buffer
// variant and checks each against a plain-slice model.
func FuzzBuffers(f *testing.F) {
	f.Add(uint8(3), []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9})
	f.Add(uint8(1), []byte{0, 0, 0, 200, 0, 100})
	f.Add(uint8(7), []byte{10, 20, 30, 40, 50, 60, 70, 80, 90, 250, 1, 2})

	f.Fuzz(func(t *testing.T, size uint8, ops []byte) {
		capacity := int(size%16) + 1
		bufs := map[string]dringbuf.Buffer[int]{
			"ring":        dringbuf.NewRingBuffer[int](capacity),
			"double":      dringbuf.NewDoubleRingBuffer[int](capacity),
			"sync ring":   dringbuf.NewThreadSafeRingBuffer[int](capacity),
			"sync double": dringbuf.NewThreadSafeDoubleRingBuffer[int](capacity),
		}
		var model []int

		for i, op := range ops {
			switch {
			case op == 255:
				model = model[:0]
				for _, b := range bufs {
					b.Clear()
				}
			default:
				model = append(model, i)
				if len(model) > capacity {
					model = model[1:]
				}
				for _, b := range bufs {
					b.Append(i)
				}
			}

			n := int(op) % (capacity + 1)
			want := model[len(model)-min(n, len(model)):]
			for name, b := range bufs {
				if b.Len() != len(model) {
					t.Fatalf("%s: Len=%d, want %d", name, b.Len(), len(model))
				}
				if got := b.Tail(n); !slices.Equal(got, want) {
					t.Fatalf("%s: Tail(%d)=%v, want %v", name, n, got, want)
				}
				if got := b.AppendTail(nil, n); !slices.Equal(got, want) {
					t.Fatalf("%s: AppendTail(%d)=%v, want %v", name, n, got, want)
				}
				for j, v := range model {
					if got := b.At(j); got != v {
						t.Fatalf("%s: At(%d)=%d, want %d", name, j, got, v)
					}
				}
				last, ok := b.Last()
				if ok != (len(model) > 0) || (ok && last != model[len(model)-1]) {
					t.Fatalf("%s: Last()=%d,%v with model %v", name, last, ok, model)
				}
			}
		}
	})
}
