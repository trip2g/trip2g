---
title: Context and cancellation in Go
lang: en
free: true
---

The `context` package gives Go programs a standard way to carry deadlines, timeouts, and cancellation signals across API boundaries. Almost every function that does I/O or calls another service should accept a `context.Context` as its first argument.

## Deadlines and timeouts

`context.WithDeadline` attaches an absolute point in time after which the context is automatically cancelled. `context.WithTimeout` is a convenience wrapper that accepts a duration instead. When the deadline expires, `ctx.Done()` is closed and `ctx.Err()` returns `context.DeadlineExceeded`. Code that blocks on I/O should select on `ctx.Done()` alongside its own channels so it can bail out promptly.

## Propagating cancellation

`context.WithCancel` returns a derived context and a `CancelFunc`. Calling the function cancels the context and every context derived from it, propagating the signal down the call tree without the caller having to track individual goroutines. This makes it straightforward to cancel an entire request subtree — database queries, outbound HTTP calls, background workers — by cancelling a single parent context. Contexts must never be stored in structs; pass them as explicit function arguments to preserve the per-request scope.
