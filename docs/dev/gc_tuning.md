# Go GC tuning (GOMEMLIMIT / GOGC)

**TL;DR.** Never set `GOMEMLIMIT` below the process's real working set. If the live
heap approaches `GOMEMLIMIT`, the Go runtime runs garbage collection *continuously*
to stay under the limit and burns up to **50 % of a CPU core doing nothing but GC**
(the runtime caps GC at 50 % of `GOMAXPROCS`, so the symptom is a process pinned at
~50 % CPU with almost no real work). Set `GOMEMLIMIT` *above* the peak working set
with headroom; keep `GOGC=100` unless you have measured a reason to change it.

## What this is for

Each trip2g instance holds a large in-memory working set: the bleve search index,
note views, layouts, sitemap, and (with vector search) embeddings loaded into RAM.
On a small shared box that working set can be 150–300 MB per instance.

`GOMEMLIMIT` is a **soft total-memory limit** for the Go runtime (heap + stacks +
runtime metadata ≈ RSS minus non-Go). `GOGC` controls how much the heap is allowed
to grow between collections (`GOGC=100` ⇒ collect when heap reaches 2× live;
`GOGC=50` ⇒ 1.5× live).

## The anti-pattern that bit us (2026-06-22)

The landing instance ran with `GOMEMLIMIT=300MiB` and `GOGC=50` while its live heap
was ~194 MB. The GC target (`live × (1 + GOGC/100)` = `194 × 1.5` ≈ **291 MB**) sat
right at the 300 MiB limit, so the runtime collected nonstop. A 45 s CPU profile was
**96.8 % `runtime.gcBgMarkWorker`** — i.e. ~50 % of a core spent only on GC, on a box
with zero external traffic (~0.5 RPS). This starved everything else: slow mutations
(20 s), exhausted SQLite write pool, and a startup-warmup timeout that escalated into
a restart crash-loop.

Raising the limit to `GOMEMLIMIT=700MiB` and `GOGC=100` dropped steady-state CPU from
~50 % to **0.35 %** (next profile showed only trivial job-queue polling). GC was gone.

## How to calculate GOMEMLIMIT correctly

1. **Measure the steady-state live heap** (`H_steady`):
   ```
   curl -s 'http://localhost:<internal_addr>/debug/pprof/heap' -o heap.prof
   go tool pprof -inuse_space -top heap.prof   # read "X total"
   ```
2. **Account for the reload peak** (`H_peak`). Note loading rebuilds the bleve index
   while the old one is still referenced, so the heap roughly **doubles** during a
   reload/warmup: `H_peak ≈ 2 × H_steady`.
3. **Add GC headroom** so the limit is a backstop, not a constant target:
   ```
   GOMEMLIMIT ≈ H_peak × 1.25 … 1.5
   ```
   The hard rule is only that `GOMEMLIMIT > H_peak`; below that you get the death
   spiral. The headroom factor decides how often GC runs near the ceiling.
4. **Check the box budget.** Σ(GOMEMLIMIT over all instances) + OS + other services
   must fit in `RAM + swap` with room to spare. `GOMEMLIMIT` is a ceiling, not a
   reservation — real usage stays near `H_steady × (1 + GOGC/100)` — but size for the
   worst case if instances can spike together. Keep swap configured as a safety net.

Rule of thumb for a dedicated container: set `GOMEMLIMIT` to ~80–90 % of the memory
available to the process and leave `GOGC=100`. On a shared box, size per instance
from its own `H_peak` as above.

## Confirming GC is (not) the bottleneck

```
curl -s 'http://localhost:<internal_addr>/debug/pprof/profile?seconds=30' -o cpu.prof
go tool pprof -top -cum cpu.prof
```
If the top frames are `runtime.gcBgMarkWorker → gcDrain → scanSpan/scanObject`, the
process is GC-bound — raise `GOMEMLIMIT` (or reduce the working set), do not chase
application code.

## Current per-instance values (source of truth: `infra/site.yml`)

| Instance | Domain | GOMEMLIMIT | GOGC | Why |
|----------|--------|-----------|------|-----|
| trip2g_landing | trip2g.com | 700MiB | 100 | largest vault / heaviest index |
| trip2g | demo.trip2g.com | 500MiB | 100 | |
| trip2g_demo2 | simple.trip2g.com | 500MiB | 100 | |
| trip2g_founder | keeper.trip2g.com | 500MiB | 100 | |

These are set as `Environment=` lines in the systemd unit (`infra/service.j2`) so they
are version-controlled. Re-measure and resize when a vault grows substantially.
