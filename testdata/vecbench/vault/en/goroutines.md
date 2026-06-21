---
title: Concurrency in Go with goroutines
lang: en
free: true
---

Go makes concurrent programming approachable through goroutines — extremely lightweight threads managed by the Go runtime rather than the operating system.

## Goroutines

Starting a goroutine is as simple as putting `go` in front of a function call. Thousands of them can run at once because each starts with only a few kilobytes of stack that grows on demand. The runtime scheduler multiplexes them onto a small pool of OS threads.

## Channels

Goroutines communicate by passing values over channels instead of sharing memory and locking it. A channel is a typed conduit: one goroutine sends, another receives, and the language guarantees the hand-off is safe. The proverb is "do not communicate by sharing memory; share memory by communicating."

## Synchronisation

For fan-out and fan-in patterns, `sync.WaitGroup` waits for a set of goroutines to finish, while `select` lets a goroutine wait on several channel operations at once. Together these primitives make pipelines and worker pools clean to express.
