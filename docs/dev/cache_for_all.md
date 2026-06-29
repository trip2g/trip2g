# Cache-for-all: extend the anonymous page cache to logged-in readers

**TL;DR.** The anonymous page cache is anon-only today. It can be safely extended to logged-in readers on **default-template note pages**, because that server HTML is identical for anonymous and logged-in viewers (the per-user bits render in the browser). Doing so would lift logged-in read throughput from ~937 to near the ~16,400 req/s the anonymous path already reaches on a €6.49/month, 2-vCPU box. **Design only — deferred; the perf ceiling is demonstrated.**

## Why

After the 2026-06-29 read-path work — page cache (#54), cache-lookup-before-Resolve (#55), OG-tag determinism (#56), read-pool prepared-statement cache + driver upgrade (#57) — a CPU profile of the **logged-in** path on cx23 (Hetzner, €6.49/mo, 2 vCPU) under vegeta load showed:

- Logged-in readers bypass the page cache (the `resp.UserToken != nil` gate in `cacheDecision`, `internal/case/rendernotepage/pagecache.go`), so every request runs the full render + DB work: **~937 req/s vs ~16,400 anonymous-cached** on the same box.
- The SQL parse + plan is already gone (#57's statement cache). What remains on the logged-in path: **per-request gzip (~25%)** — no cache, so every response is re-gzipped — plus **query execution `_sqlite3VdbeExec` (~44%)** — the per-view bookkeeping queries.

So the logged-in path still pays the full render + gzip the cache was built to avoid.

## What makes it safe (already de-risked)

- #56 proved the default-template **note** render is **user-independent** for anonymous and non-admin logged-in viewers (byte-identical HTML). The per-user bits — login button, edit badge, paywall-unlock — render client-side via the `$trip2g_user_space` bundle; the live note template (`internal/defaulttemplate`, `StreamNoteContent`) has no per-user server branch. Admins differ only on the onboarding page, which is not a cached note page.
- So a logged-in reader can be served the **same anonymous pre-gzipped cache bytes** on default-template note pages.

## Plan

Extend `serveCachedPageEarly` (added in #55, `internal/case/rendernotepage/pagecache.go`):

1. For a logged-in request on a **default-template note** (NOT a custom Jet layout, NOT a note with subgraph-gated / paywalled sections): check `CanReadNote` (access — already in-memory-cheap after #57), then read the same anonymous page-cache entry (key `{path, host, note_version_id, config_epoch, ui_lang}`; the bytes are user-independent) and serve on hit.
2. Move the per-view bookkeeping — `RecordUserNoteView` / `InsertUserNoteView` / `UpsertUserNoteDailyView` / `IncreaseUserNoteViewCount` / `LastUserNoteView` — to an **async side-effect**, so a cache hit does not block on write-pool queries. This takes both the gzip (~25%) and the execution (~44%) off the response critical path.
3. Keep bypassing the cache for: **custom Jet layouts** (kanban etc. — flagged `Personalized`) and **notes with subgraph-gated / paywalled sections** (per-user server-side unlock).

Expected: logged-in cacheable views go from ~937 → near the anonymous ~16,400 req/s on the same box.

## Open design questions

- **Async view-recording:** ordering and loss-on-crash. Acceptable for analytics (fire-and-forget), but decide the buffer/queue and at-least-once vs best-effort.
- **Logged-in-only pages:** a note never requested anonymously has no cache entry (store is anon-gated today). Either accept it (most pages also get anonymous traffic) or allow logged-in stores too (the bytes are user-independent).
- **Exclusion completeness:** the `Personalized` marker covers custom Jet layouts only. The subgraph-gated / paywalled-section exclusion needs an explicit check — the unlock is client-side today, but confirm no server-side per-user branch sneaks back into the default template.

## Companion simplification (separate, optional)

Drop `UILang` from the cache key. The default template server-translates ~6 UI strings via `ctx.T(ctx.UILang, …)` (widget titles, "read more", Telegram link labels) and embeds `ui_lang`. Plan: SSR those strings in the **note's** language (already in the key via `NoteVersionID`) and let the client swap them to the viewer's UI language if it differs. Removes the `UILang` key dimension (~3× fewer entries; one entry per note-version serves every UI language). **Rationale = SEO:** crawlers and no-JS clients get UI strings in the note's own content language, coherent with the page content, rather than a variant keyed off a visitor header. Graceful-without-JS falls out for free.

## Status

Design only. Not built. The perf ceiling is demonstrated — anonymous ~16,400 req/s, logged-in ~937 req/s, SQL parse eliminated. This is the documented next step for when the logged-in read path needs the headroom.
