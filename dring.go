package dringbuf

type RingBuffer[T any] interface {
	Append(elem T)
	Len() int
	Size() int
	At(idx int) T
	Last(n int) []T
	Clear()
}
