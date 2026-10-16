# dringbuf
`[double-sized] ring buffer`

Generic ring buffer implementations in Go, available in two flavors:

* **Single-array** — a single-array circular buffer (`NewRingBuffer`, `NewThreadSafeRingBuffer`). `Tail(n)` returns a freshly allocated `[]T`.
* **Double-sized** — uses a **2 × capacity** underlying slice to provide a continuous `slice` view of its elements at any moment in time (`NewDoubleRingBuffer`, `NewThreadSafeDoubleRingBuffer`).

The key design goal of the **double-sized** variant is **zero allocations during runtime operations**.
The only allocation occurs once — at buffer creation.

This makes it particularly suitable for **data streaming pipelines**, where consumers expect data as `[]T` and allocation overhead must be avoided. The package also ships a `Stream` that fans values out to buffered subscribers built on the double-sized buffer.

## Motivation

Traditional circular buffers suffer from one major limitation:

When the logical window wraps around the end of the underlying array, the data becomes physically fragmented. Returning a contiguous `[]T` slice requires:

* either copying,
* or allocating a new slice,
* or exposing two separate slices.

The **double-sized** implementation avoids that entirely.

By maintaining an underlying slice of **2 × capacity**, every element is mirrored at an offset equal to the buffer size. This guarantees that the active window is always represented as a **single contiguous slice** in memory.

As a result, for the double-sized variant (`DoubleRingBuffer` / `SyncDoubleRingBuffer`):

* `Tail(n)` on `DoubleRingBuffer` returns a `[]T` view without allocation
* `View(n, fn)` on `SyncDoubleRingBuffer` passes that view to `fn` under a read lock, without allocation
* No copying is required
* No wrap-around handling is required by the consumer
* The buffer is allocation-free after initialization

The **single-array** variant (`RingBuffer` / `SyncRingBuffer`) stores each element once in a single slice of capacity `N`. It uses half the memory of the double-sized variant, but `Tail(n)` allocates a new slice and copies. Use `AppendTail(dst, n)` with a reused `dst` to copy without allocating on either variant.

## Types

| Constructor | Returns |
| --- | --- |
| `NewRingBuffer[T](n)` | `*RingBuffer[T]` |
| `NewDoubleRingBuffer[T](n)` | `*DoubleRingBuffer[T]` |
| `NewThreadSafeRingBuffer[T](n)` | `*SyncRingBuffer[T]` |
| `NewThreadSafeDoubleRingBuffer[T](n)` | `*SyncDoubleRingBuffer[T]` |

`SyncRingBuffer[T]` and `SyncDoubleRingBuffer[T]` are aliases of the generic wrapper
`ThreadSafeRingBuffer[B, T]`, which works with any `Buffer[T]`. Reads share a read lock;
`Append` and `Clear` take the write lock. All four types implement `Buffer[T]`.

## API

Elements are ordered oldest → newest. `At(0)` / `Get(0)` address the oldest element and `At(Len()-1)` / `Get(Len()-1)` the newest.

| Method | Description |
| --- | --- |
| `Append(v)` | Add `v`, overwriting the oldest element when full. |
| `Len()`, `Cap()` | Current element count / maximum capacity. |
| `At(i) T` | Element at index `i`; panics when `i < 0 || i >= Len()`. |
| `Get(i) (T, bool)` | Like `At` but returns `false` instead of panicking. |
| `Last() (T, bool)` | Most recently appended element, or `false` when empty. |
| `Tail(n) []T` | Last `n` elements (clamped to `Len()`) in oldest → newest order. |
| `AppendTail(dst, n) []T` | Appends a copy of the last `n` elements to `dst`. |
| `Clear()` | Remove all elements. |
| `View(n, fn)` | Thread-safe types only: calls `fn` with the last `n` elements under the read lock. |

Ownership: `Tail` copies for `RingBuffer` and for both thread-safe types. For `DoubleRingBuffer`, `Tail` returns a view into internal storage, and so does `View` on `SyncDoubleRingBuffer`: do not modify it, and do not keep it past the next `Append`/`Clear` (or past `fn` returning, for `View`). Calling `append` on a view is safe: it always reallocates.

## Stream

```go
s := dringbuf.NewStream[Event]()
sub := s.Subscribe(1024, dringbuf.BackpressureStrategyBlock)

go func() {
	for sub.ReadView(func(batch []Event) { process(batch) }) == nil {
	}
}()

s.Emit(e)
s.Close() // readers drain what they have, then get io.EOF
```

Every `Emit` is delivered to every subscription. When a subscription's buffer is full, its strategy decides what happens:

| Strategy | When full |
| --- | --- |
| `BackpressureStrategyBlock` | `Emit` waits for the reader. Nothing is lost. |
| `BackpressureStrategyDropOldest` | The oldest unread value is overwritten. |
| `BackpressureStrategyDropNewest` | The new value is discarded. |

`Emit` delivers to subscriptions one after another, so one full `Block` subscription delays every subscription after it.

Reading (all can be used from several goroutines):

| Method | Description |
| --- | --- |
| `Next() (T, error)` | Oldest unread value, blocking until one arrives. |
| `NextContext(ctx) (T, error)` | Like `Next`, but stops waiting when `ctx` is done. |
| `Read(dst) (int, error)` | Copies up to `len(dst)` unread values with one lock. |
| `ReadView(fn) error` | Calls `fn` with all unread values as one contiguous slice, without copying. Emit waits while `fn` runs. |

Closing:

* `Stream.Close()` — the producer is done. Subscriptions can read what they have buffered, then get `io.EOF`. Later `Emit` calls do nothing.
* `Subscription.Close()` — the consumer is leaving. Buffered values are dropped, blocked `Emit` calls return, and reads return `io.EOF`.

`Emit`, `Next`, `Read` and `ReadView` do not allocate.

## Core Idea

If `capacity = N`, the underlying slice size is `2N`.

For every element written at index `i`, it is stored at:

```
data[i]
data[i + N]
```

This duplication ensures that any logical window of size ≤ N is always contiguous in memory.
