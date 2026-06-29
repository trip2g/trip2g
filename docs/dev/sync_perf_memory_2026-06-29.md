# pushNotes memory/allocation profiling — 2026-06-29

> **TL;DR.** Every `pushNotes` runs a full `noteloader.Load`. Under sustained push
> load the server allocates ~16 GB over 18 s and spends **~45 % of CPU in GC**.
> The allocation churn is dominated by work that is **recomputed every reload even
> though its inputs rarely change**: re-parsing layout templates (~29 %), rebuilding
> the Jsonnet `std` object per patch eval (53 % of allocation *objects*), and
> re-loading+decoding chunk embeddings (~13 %). All of it is fixable by caching /
> reuse — cutting allocations directly cuts the GC tax.

Profiles: `/tmp/push-cpu.prof`, `/tmp/push-allocs.prof` (`go tool pprof -http=:6060 tmp/main <prof>`).
Load: `scripts/sync-perf-load.mjs --mode api` against the docs vault (588 notes, has
frontmatter patches + layouts + embeddings). Server sustained ~3 push/s vs 10–12 offered
(writes serialize on the per-push reload; p95 latency 5–20 s under backlog).

## Ranked candidates (by allocation share → GC leverage)

### TIER 1

**1. Jet layout templates re-parsed on every reload — ~29 % of bytes.**
`jet.Set.loadFromFile` → `io.ReadAll` (8.1 %) + `Set.parse` (21 % cum). Root causes in
`internal/layoutloader/loader.go`: a fresh `jetLoader` + `jet.NewSet(...)` is built on
**every** `Load` (line 65, 416, `jl.sets[id]=...` line 470), and the Set is created with
**`jet.DevelopmentMode(true)`** (line 416) which disables Jet's compiled-template cache →
templates are re-read and re-parsed from the loader on every access, every push.
*Fix:* cache the compiled `jet.Set` across reloads (rebuild only when a layout's content
hash changes), and/or drop `DevelopmentMode(true)` in production. Layouts change rarely;
this is the single biggest, lowest-risk win. Independent of the patch work.

**2. Jsonnet `std` rebuilt per patch evaluation — 53 % of allocation OBJECTS, ~22 % bytes.**
`buildStdObject` (via `buildInterpreter`→`evaluateStd`) rebuilds the entire Jsonnet `std`
library object on every `ApplyPatches` call (per note, per reload). Object count, not just
bytes, is what drives GC scan time — and Jsonnet is ~70 % of all objects.
*In flight:* the frontmatter-patch **result cache** (branch `perf/frontmatter-patch-cache`,
PR pending) skips eval entirely for unchanged notes — the common case — which removes most
of this. *Residual:* notes that genuinely re-evaluate still rebuild `std` each call; if it
stays hot after the cache lands, reuse/cache the std object or the parsed interpreter
across evaluations.

### TIER 2

**3. Chunk embeddings re-loaded + re-decoded every reload — ~13 % of bytes.**
`noteloader.loadChunks` (called unconditionally, loader.go:324) pulls ALL chunk rows from
SQLite (`columnText` 4.7 % + `columnBlob` 4.1 %) and decodes every embedding via
`model.BytesToFloat32Slice` (4.15 %) on every push reload. Embeddings are only needed for
vector-search queries, don't change on a content push, and aren't used by the post-push
serving render. *Fix:* gate `loadChunks` (skip when vector search isn't being served /
in the push reload), or cache the decoded `[]NoteChunk` and reuse it when the chunk rows
are unchanged.

**4. Render output buffers not pooled — ~13 % of bytes.**
`bytes.growSlice` (6.3 %), `bytes.Clone` (4.0 %), `bytes.Buffer.String` (2.8 %),
`bufio.NewWriterSize` — fresh goldmark/HTML render buffers per note. *Fix:* a `sync.Pool`
of render buffers reused across notes in `mdloader.finishPage`. Classic pooling win;
low risk.

## Cross-cutting
- GC is ~45 % of CPU under load — every candidate above reduces it directly.
- GOGC / GOMEMLIMIT tuning is a *complementary* knob only. Prod runs `GOMEMLIMIT=700MiB`
  and has a documented GC-death-spiral history, so **raising the limit is risky**; cutting
  allocations is the safe lever.
- The per-push *full* reload itself is the structural cost. The AST is already cached
  (`mdloader` `noteCache`, content-hash keyed); the items above are the parts of the reload
  that are NOT yet reused.

## Suggested order
1. **Jet Set cache / drop DevelopmentMode** (Tier 1.1) — biggest, isolated, low risk.
2. Land the **frontmatter-patch result cache** (in flight) — kills the 53 %-objects Jsonnet path for unchanged notes.
3. **Gate/cache chunk embeddings** (Tier 2.3) — skip work the serving reload doesn't need.
4. **Pool render buffers** (Tier 2.4) — steady GC reduction.
</content>
