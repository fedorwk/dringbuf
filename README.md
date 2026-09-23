# dringbuf
`[double-sized] ring buffer`

Generic ring buffer implementations in Go, available in two flavors:

* **Single-array** — a single-array circular buffer (`NewRingBuffer`, `NewThreadSafeRingBuffer`). `Tail(n)` / `Borrow(n)` return a freshly allocated `[]T`.
* **Double-sized** — uses a **2 × capacity** underlying slice to provide a continuous `slice` view of its elements at any moment in time (`NewDoubleRingBuffer`, `NewThreadSafeDoubleRingBuffer`).

The key design goal of the **double-sized** variant is **zero allocations during runtime operations**.
The only allocation occurs once — at buffer creation.

This makes it particularly suitable for **data streaming pipelines**, where consumers expect data as `[]T` and allocation overhead must be avoided.

## Motivation

Traditional circular buffers suffer from one major limitation:

When the logical window wraps around the end of the underlying array, the data becomes physically fragmented. Returning a contiguous `[]T` slice requires:

* either copying,
* or allocating a new slice,
* or exposing two separate slices.

The **double-sized** implementation avoids that entirely.

By maintaining an underlying slice of **2 × capacity**, every element is mirrored at an offset equal to the buffer size. This guarantees that the active window is always represented as a **single contiguous slice** in memory.

As a result, for the double-sized variant (`NewDoubleRingBuffer` / `NewThreadSafeDoubleRingBuffer`):

* `Tail(n)` returns a `[]T` without allocation (base version, `RingBuffer`)
* `Borrow(n)` returns a `[]T` and a `Release` function without allocation (thread-safe version, `ThreadSafeRingBuffer`)
* No copying is required
* No wrap-around handling is required by the consumer
* The buffer is allocation-free after initialization

Constructors return concrete types (`*RingBuffer[T]`, `*DoubleRingBuffer[T]`, `*ThreadSafeRingBuffer[B, T]`),
so method calls are statically dispatched and can be inlined. `NewThreadSafeRingBuffer` and
`NewThreadSafeDoubleRingBuffer` both return the same `ThreadSafeRingBuffer` wrapper parameterized by
the concrete underlying buffer type.

The **single-array** variant (`NewRingBuffer` / `NewThreadSafeRingBuffer`) stores each element once in a single slice of capacity `N`. It uses half the memory of the double-sized variant, but `Tail(n)` and `Borrow(n)` always allocate a new slice and copy.

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
| `Clear()` | Remove all elements. |
| `Borrow(n) ([]T, Release)` | Thread-safe read view; call `Release` when done. |

Ownership: `Tail`/`Last` copy for `RingBuffer` and `ThreadSafeRingBuffer`; `Tail` and `Borrow` return a view into internal storage for `DoubleRingBuffer` (do not modify, and do not retain past the next `Append`/`Clear`).

## Core Idea

If `capacity = N`, the underlying slice size is `2N`.

For every element written at index `i`, it is stored at:

```
data[i]
data[i + N]
```

This duplication ensures that any logical window of size ≤ N is always contiguous in memory.
