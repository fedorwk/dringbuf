# dringbuf
`[double-sized] ring buffer`  

A high-performance, generic **ring buffer** implementation in Go that uses a **double-sized underlying slice** to provide a continuous `slice` view of its elements at any moment in time.

The key design goal is **zero allocations during runtime operations**.
The only allocation occurs once — at buffer creation.

This makes the structure particularly suitable for **data streaming pipelines**, where consumers expect data as `[]T` and allocation overhead must be avoided.

## Motivation

Traditional circular buffers suffer from one major limitation:

When the logical window wraps around the end of the underlying array, the data becomes physically fragmented. Returning a contiguous `[]T` slice requires:

* either copying,
* or allocating a new slice,
* or exposing two separate slices.

This implementation avoids that entirely.

By maintaining an underlying slice of **2 × capacity**, every element is mirrored at an offset equal to the buffer size. This guarantees that the active window is always represented as a **single contiguous slice** in memory.

As a result:

* `Last(n)` returns a `[]T` without allocation
* No copying is required
* No wrap-around handling is required by the consumer
* The buffer is allocation-free after initialization

## Core Idea

If `capacity = N`, the underlying slice size is `2N`.

For every element written at index `i`, it is stored at:

```
data[i]
data[i + N]
```

This duplication ensures that any logical window of size ≤ N is always contiguous in memory.
