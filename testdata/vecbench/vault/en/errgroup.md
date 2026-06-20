---
title: Coordinating errors with errgroup
lang: en
free: true
---

The standard library's `sync.WaitGroup` waits for goroutines to finish but has no built-in mechanism for collecting errors. The `golang.org/x/sync/errgroup` package fills that gap, combining goroutine coordination with first-error propagation in a single, ergonomic API.

## Starting tasks with an errgroup

`errgroup.WithContext` returns a `*Group` and a derived `context.Context`. Each concurrent task is launched with `g.Go(func() error {...})`. The group tracks how many tasks are running and, when a task returns a non-nil error, cancels the shared context so sibling tasks can detect the failure and exit early through their own `ctx.Done()` checks.

## Collecting the first error

`g.Wait()` blocks until all goroutines have returned, then returns the first non-nil error encountered. Subsequent errors are silently discarded — the assumption is that once one task fails the others will also be cancelled, so only the root cause matters. This makes `errgroup` ideal for fan-out work where all sub-tasks must succeed for the overall operation to proceed: parallel API calls, concurrent file processing, or batched database writes where a single failure should abort the rest.
