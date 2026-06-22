# Render hot-path performance — optimization plan (TODO)

Profiled 2026-06-22 on a Hetzner cpx32 (4 vCPU AMD EPYC-Genoa, 8 GB), **production mode**,
load generated from a **separate machine**, real default-template doc pages (~52 KB) at
escalating load. One node serves **~4 000 rps at 100 %** (p99 ~30 ms); the knee is
~4 000–5 000 rps. Two costs dominate the per-request hot path, both cacheable (pages are
static between writes), so there is room to push the ceiling several×. Not doing this yet —
recording the plan.

Benchmark methodology matters a lot here — three traps each cost ~2× and compounded to a 4×
understatement on the first run: (1) DevMode recomputes asset hashes per request (finding #4
below); (2) running vegeta on the same box steals ~2 cores from the server; (3) cold cache /
single URL. Always: `DEV=false`, separate load generator, warm cache, many real pages.

## Findings (pprof CPU profile, internal port `/debug/pprof`)

1. **gzip level 6 on every response — ~36 % CPU.** `fasthttp.nonblockingWriteGzip` →
   `klauspost/compress/flate.fastEncL6.Encode`. Each ~52 KB page is recompressed per request.
2. **Per-request DB queries in the render path — ~32 % CPU** (`rendernotepage.Resolve` →
   `database/sql.withLock` + SQLite parse/exec): `CanReadNote` (access),
   `GetTelegramPostLinksByNoteVersionID` (Telegram links), `ActiveHTMLInjections`, plus
   view-tracking **writes** (`InsertUserNoteView` / `IncreaseUserNoteViewCount`). These
   serialize on the connection-pool lock under load. (So the "no database on the hot path"
   claim in `docs/*/user/perfomance.md` is not strictly true today.)
3. **GC ~30 %** — driven by the gzip buffers + per-request allocations.
4. **Dev-only artifact:** `assetURL` (`cmd/server/main.go`) recomputes SHA-256 of each asset
   per request in DevMode (the cache is gated on `!DevMode`) — ~19 % CPU in dev, **none in
   prod**. It made the first benchmark ~2× pessimistic.

## Plan (ordered by impact)

- [ ] **Cache the gzipped page bytes.** Pages are pre-rendered and static between writes —
      gzip once when the note view is built, store the compressed bytes next to the HTML, and
      serve them directly (skip per-request compression). Removes ~36 % CPU. Cheap fallback:
      drop the gzip level 6 → 2 (klauspost level 2 is ~3–4× cheaper, output a few % larger).
- [ ] **Move per-render lookups into the in-memory note view.** Telegram links are immutable
      per note version; HTML injections are site-wide and change rarely — precompute/cache
      both instead of querying per request. For public notes, short-circuit `CanReadNote`
      without a query.
- [ ] **Make view tracking async.** `InsertUserNoteView` / `IncreaseUserNoteViewCount` are
      writes on the *read* hot path — batch them through a channel/queue so a render never
      blocks on a write or contends the single writer slot.
- [ ] **`assetURL`: cache unconditionally** (drop the `!DevMode` gate) so dev profiles match
      prod.

Expected: removing gzip-per-request + the per-render DB queries should roughly 2–3× the
per-node ceiling (toward ~5–6k rps) and flatten the latency cliff past the knee.
