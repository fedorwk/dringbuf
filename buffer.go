package dringbuf

// Buffer is the set of operations shared by RingBuffer and DoubleRingBuffer.
// ThreadSafeRingBuffer wraps any Buffer.
type Buffer[T any] interface {
	Append(elem T)
	Len() int
	Cap() int
	At(idx int) T
	Get(idx int) (T, bool)
	Last() (T, bool)
	Tail(n int) []T
	AppendTail(dst []T, n int) []T
	Clear()
}

var (
	_ Buffer[int] = (*RingBuffer[int])(nil)
	_ Buffer[int] = (*DoubleRingBuffer[int])(nil)
	_ Buffer[int] = (*ThreadSafeRingBuffer[*RingBuffer[int], int])(nil)
)
