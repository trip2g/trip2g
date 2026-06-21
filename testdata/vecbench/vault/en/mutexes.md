---
title: Mutexes and shared state in Go
lang: en
free: true
---

When multiple goroutines must read and write the same data, channels are not always the right tool. The `sync` package provides mutual exclusion locks for cases where protecting shared memory is cleaner than restructuring around message passing.

## sync.Mutex and sync.RWMutex

`sync.Mutex` exposes two methods: `Lock` and `Unlock`. Any goroutine that calls `Lock` blocks until the mutex is available, then holds exclusive access until `Unlock` is called. A common pattern is to defer `Unlock` immediately after acquiring the lock so it is released even if the function returns early.

`sync.RWMutex` is a reader-writer variant. Multiple goroutines may hold a read lock (`RLock`) simultaneously, but a write lock (`Lock`) requires exclusive access. This is a measurable win for data that is read far more often than it is written, such as in-memory caches or configuration maps.

## Choosing locks over channels

Channels express ownership transfer and pipeline flows. Mutexes protect state that genuinely needs to be shared in place — counters, maps, connection pools. Go's race detector (`go test -race`) catches lock violations and data races at runtime, making it straightforward to verify that a mutex is protecting every access path correctly.
