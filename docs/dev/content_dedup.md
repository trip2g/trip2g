# Content dedup at the note-version level

**TL;DR:** trip2g stores and embeds every note version independently, keyed by `version_id`. A vault with 1344 notes but only 946 unique bodies (398 byte-identical copies) pays 3× for the duplicates: duplicate `note_versions.content` rows, duplicate embedding compute + vectors, and duplicate search hits crowding the top-K. A usable content hash already exists in two places (`note_paths.latest_content_hash` and `note_version_embeddings.content_hash`), so we don't need to invent one. Recommended order: (1) dedup identical-content results in search — cheap, kills the retrieval noise; (2) dedup embedding *compute* by looking up existing vectors by `content_hash` before calling the model; (3) full content-addressed storage (qmd-style `contents` table) only if storage ever actually hurts — it's the most invasive and least urgent phase.

## 1. Current state

### Content storage

- `note_paths` / `note_versions` schema: `db/migrations/20250402131258_create_note_tables.sql:3-19`. Every version stores the **full raw body** in `note_versions.content` (`text not null`). There is no hash column on `note_versions`.
- `note_paths.latest_content_hash` = base64url(sha256(raw content)), computed on every push at `internal/case/insertnote/resolve.go:42-43` and used only as a "did the latest version change" guard (`resolve.go:80`) and by the git mirror to skip unchanged files (`internal/gitapi/apply.go:61,75-76`).
- So: 398 duplicate notes → 398 duplicate full bodies in `note_versions` (times however many versions each accumulates). SQLite pays the bytes; `noteloader` also loads all of them into memory via `AllLatestNotes` (`queries.read.sql:32-37`, `internal/noteloader/loader.go:199-223`).

### Embeddings

Two tables, both keyed by **version**, not by content:

- `note_version_embeddings(version_id PK, embedding, model_id, content_hash, tokens)` — `db/migrations/20260102093534_create_note_version_embeddings.sql`. One whole-note vector per version.
- `note_version_chunks(version_id, chunk_index, content, embedding, model_id, content_hash)` — `db/migrations/20260318100000_create_note_version_chunks.sql`. Chunk text is **stored again** here (a third copy of the body, sliced), plus one vector per chunk.

The `content_hash` columns are already the right dedup key, but today they're used only as a **change detector within the same version_id**:

- Whole note: `NoteContentHash` = sha256(title + frontmatter-stripped content + model fingerprint) — `internal/case/backjob/generatenoteversionembedding/resolve.go:212-215`. The generator skips re-embedding when the stored row for *this version* has the same hash (`resolve.go:56-69`).
- Chunks: sha256(chunk content + fingerprint), same-version skip at `resolve.go:144-151`.
- The regeneration cron does the same per-version comparison (`internal/case/cronjob/regeneratenoteembeddings/resolve.go:68-75`).

**Nothing ever asks "has this hash been embedded under a different version_id?"** Two paths with identical bodies (same title — they're file copies, so titles match and the chunk breadcrumbs match too) produce byte-identical hashes and byte-identical vectors, computed and stored twice. For the flat test vault that's 398 redundant whole-note embeddings plus all their chunks — pure wasted GPU/API calls and duplicate blobs, repeated again whenever the model fingerprint changes.

### Search

Both search paths dedup by **note path only**, never by content:

- Site search: vector candidates deduped via `seen[c.path]` (`internal/case/sitesearch/resolve.go:229-236`), RRF merge keyed by `r.URL` (`resolve.go:300-328`).
- MCP search: same pattern — `seen[s.chunk.NotePath]` at `internal/case/mcp/resolve.go:1217-1222`, merge by URL at `resolve.go:1283-1307`.

Identical bodies produce identical vectors, so all 20 copies of a shared doc score identically against any query and arrive as 20 adjacent results. With `vectorTopK = 50` (`sitesearch/resolve.go:164`) and `hybridResultCap = 20` (`resolve.go:47`), a popular duplicated doc can occupy most of the visible result list, evicting genuinely distinct hits. Same for the MCP `MaxMergedResults` cap. This is the measured "search noise" cost.

### Where the duplication is paid — summary

| Cost | Where | Scale (test vault) |
|---|---|---|
| Raw body storage | `note_versions.content` ×2, chunk text in `note_version_chunks.content` | 398 duplicate bodies + their chunk copies |
| Embedding compute | `generatenoteversionembedding` calls the model per version | 398 redundant whole-note calls + chunk batches, re-paid on every model/fingerprint change |
| Vector storage + RAM | `note_version_embeddings.embedding`, `note_version_chunks.embedding`, all loaded into memory (`loader.go:338-345`) | 398× (whole-note + chunks) duplicate float32 blobs |
| Search quality | top-K crowded with identical bodies | up to K copies of one doc per query |

## 2. Existing content hashes — reuse, don't invent

Yes, two usable hashes already exist:

1. `note_paths.latest_content_hash` — sha256 of the **raw** body (frontmatter included), latest version only. Good identity for "byte-identical file", exactly the qmd notion. Missing for historic versions, but trivially recomputable.
2. `note_version_embeddings.content_hash` / `note_version_chunks.content_hash` — sha256 of the **embedding input** (title + stripped body + model fingerprint). This is the *correct* key for embedding reuse — it already accounts for the fact that the embedded text includes the title and excludes frontmatter, and that a model change must invalidate.

git blob shas via gitapi are not useful here: the mirror is derived *from* the DB (DB-canonical), asynchronous, and hashes with git's `blob <len>\0` prefix — a third incompatible identity. Skip it.

Key subtlety: the two hashes answer different questions. Raw-content hash says "same file bytes" (right for storage dedup and for search collapse). Embedding hash says "same embedding input under this model" (right for vector reuse). Two files with identical bodies but different frontmatter dedup under the *embedding* hash (frontmatter is stripped) but not under the raw hash; two files with the same body but different titles (rename-copies) dedup under the *raw* hash but not the embedding hash. Both cases exist in real vaults; use each hash for its own job.

## 3. Design: content-addressed storage (Phase 3, the qmd shape)

qmd's model: `content(hash PK, body)` + `documents(..., content_hash FK)`. The trip2g equivalent:

### Schema

```sql
create table note_contents (
  hash text primary key,          -- base64url(sha256(raw content)), same encoding as note_paths.latest_content_hash
  content text not null,
  size integer not null,
  created_at datetime not null default current_timestamp
);

-- note_versions: replace content with a reference
alter: note_versions(id, path_id, version, content_hash text not null references note_contents(hash),
                     created_at, created_by_* ...)
```

Embeddings move from version-keyed to content-keyed:

```sql
create table content_embeddings (
  content_hash text not null references note_contents(hash),
  embed_hash blob not null,       -- NoteContentHash(title, content, cfg): title+model fingerprint identity
  embedding blob not null,
  model_id integer not null,
  tokens integer not null,
  primary key (content_hash, embed_hash)
);
-- content_chunks analogous, keyed (content_hash, embed_hash_prefix, chunk_index)
```

(Practically: keep `embed_hash` as the primary lookup and treat `content_hash` as the storage-GC anchor. Because the embedded text includes the title, notes sharing a body but not a title need separate vectors — the two-hash key models that honestly.)

### Write/sync path change

`internal/case/insertnote/resolve.go` already computes the raw hash (line 42-43). Change: `insert or ignore into note_contents(hash, content)` then insert `note_versions` with `content_hash` instead of `content` (`queries.write.sql:27-30` `InsertNoteVersion`). The existing dedup guard at `resolve.go:80` is untouched. Every reader of `note_versions.content` (`AllLatestNotes` at `queries.read.sql:32`, `AllLiveNotes` at `:366`, `AllNoteVersionsByPathID` at `:12`, version-diff/editor paths) gains a join — mechanical but wide; sqlc regeneration makes the compiler find all of them.

### Embedding pipeline change

`generatenoteversionembedding` becomes: compute `embed_hash`; `select ... from content_embeddings where embed_hash = ?`; on hit, done (no per-version copy needed at all — search reads through content). On miss, embed once and insert. The cron (`regeneratenoteembeddings/resolve.go:68-77`) enqueues per unique `embed_hash` instead of per version — 946 jobs instead of 1344. Chunk logic is the same per-chunk (`generatenoteversionembedding/resolve.go:144-151` already has the hash-skip loop; only the lookup scope widens from "this version" to "any").

### Migration / backward compat

1. Additive migration: create `note_contents` + `content_embeddings`; add nullable `note_versions.content_hash`.
2. Backfill in batches: hash each `note_versions.content`, upsert into `note_contents`, set `content_hash`. Copy one embedding row per distinct `embed_hash` from `note_version_embeddings`/`note_version_chunks`.
3. Flip readers to the join; keep `note_versions.content` for one release as a fallback, then drop via SQLite table-rebuild (the codebase already has this pattern, e.g. `db/migrations/20250515071315_add_id_to_note_versions.sql`).
4. Old `note_version_embeddings` tables can be dropped after cutover; nothing external references them.

Backfill rewrites the largest table in the DB — run it as a one-time batched job, not inside a migration transaction, and mind the prod box (single small VM; see prod constraints).

## 4. Cheaper interim win: search-side dedup (no schema change)

Collapse identical-content hits at query time. This alone removes the user-visible pain.

**Key:** a per-`NoteView` raw content hash held in memory. Two options:
- carry `note_paths.latest_content_hash` through `AllLatestNotes`/`AllLiveNotes` (it's already on the joined `note_paths` row — just add the column to the select at `queries.read.sql:32-37` and `:364-373` and thread it into `mdloader.SourceFile`/`NoteView`), or
- compute sha256 in `mdloader` at parse time (content is already in hand; hashing 1-2k notes is microseconds). This also covers live-release versions whose content is *not* the latest (where `latest_content_hash` would be wrong) — **prefer this option**.

**Where:**
- `internal/case/sitesearch/resolve.go` — after `mergeResults` + rerank, in the permission-filter loop (`resolve.go:104-137`): keep the first (highest-ranked) result per content hash, skip the rest. Optionally attach the collapsed siblings' URLs to the canonical hit ("also at: …") so nothing becomes unreachable.
- `internal/case/mcp/resolve.go` — same collapse in `vectorResultsFromChunks` (`resolve.go:1218-1242`, widen `seen` from path to content hash) and in `mergeResults` (`:1283`). Federated search dedups per-KB automatically once the local handler does.

**Canonical-pick rule:** highest fused score wins; tie-break by shortest path (copies usually live deeper than the original). Deterministic and cheap.

**Important:** collapse *after* permission filtering, or pick the canonical among *readable* copies — otherwise a query could collapse 20 copies into the one the user can't read. The current loop order in `sitesearch/resolve.go:104` makes this easy: dedup inside the `canRead` branch.

One catch: dedup before the top-K cap effectively widens recall (`vectorTopK=50` unique contents instead of 50 paths) — that's the desired behavior and needs no tuning.

## 5. Cheap embedding-compute win (Phase 2, minimal schema)

Between "search dedup" and "full storage dedup" there is a very cheap middle step: keep both tables as they are, but before calling the model in `generatenoteversionembedding/resolve.go:84`, look up an existing row by hash:

```sql
-- needs: create index idx_nve_content_hash on note_version_embeddings(content_hash);
select embedding, tokens from note_version_embeddings where content_hash = ? limit 1;
```

On hit, upsert the copied vector for this version_id and skip the API call. Same for chunks (`resolve.go:166`) with an index on `note_version_chunks(content_hash)`. Storage stays duplicated, but the expensive part — GPU/API embedding of 398 redundant notes, re-paid on every model change/re-index — disappears. Two additive index migrations, ~30 lines of Go, no reader changes.

## 6. Risks / edge cases

- **Version history:** untouched in every phase — versions remain distinct rows; only the body (Phase 3) or vector (Phase 2) is shared. Version diff/editor views read through the content reference.
- **Frontmatter/path differ, body identical:** raw-hash dedup treats different frontmatter as different content (correct — frontmatter drives routes/lang/permissions). Embedding-hash dedup strips frontmatter, so those copies still share vectors (also correct). Search collapse uses the raw hash, so notes that differ only in frontmatter are *not* collapsed — safe default; revisit only if it proves too conservative.
- **Same body, different title:** embedding input includes the title (`generatenoteversionembedding/resolve.go:81`), so embed-hash differs — no vector reuse, by design. Search collapse (raw hash) still merges them; the canonical hit shows one title. Acceptable: bodies are byte-identical, but call it out in the changelog.
- **Permissions / subgraphs:** copies may live in subgraphs with different `require_signin` or read patterns. Collapse must respect per-copy `CanReadNote` (§4). Federation is per-instance, no cross-tenant hash sharing exists.
- **GC (Phase 3):** deleting a version must not delete shared content. Either refcount or a periodic `delete from note_contents where hash not in (select content_hash from note_versions)` sweep. `on delete restrict` on the FK prevents accidents.
- **Cache invalidation:** `note_contents` rows are immutable (content-addressed), so no invalidation problem is introduced. Page cache stays keyed by version. The NoteCache content-equality check in `noteloader/loader.go:275` keeps working, and could later compare hashes instead of bytes.
- **Hash collisions:** sha256, ignore.
- **Backfill load:** Phase 3's backfill rewrites `note_versions` — batched, off-peak, resumable (see prod box constraints in ops notes).

## 7. Phased recommendation

| Phase | What | Effort | Payoff |
|---|---|---|---|
| **1. Search dedup** (do first) | Per-NoteView content hash in memory; collapse identical-hash results in sitesearch + mcp after permission check | ~0.5–1 day incl. tests | Kills retrieval noise/confusion — the measured benchmark pain — with zero schema risk |
| **2. Embedding-compute dedup** | Index `content_hash` on both embedding tables; lookup-before-embed in `generatenoteversionembedding` | ~0.5 day | Eliminates redundant model calls (398/1344 ≈ 30% of embed work on this vault), incl. every future re-index |
| **3. Content-addressed storage** (qmd parity) | `note_contents` + content-keyed embeddings, batched backfill, reader joins | 3–5 days + careful rollout | Storage + RAM savings; architecturally clean; only phase that needs a real migration — defer until storage/RAM actually hurts |

Single highest-leverage cheap win: **Phase 1 search collapse keyed on a raw-content sha256 computed at note load**, because it fixes the user-visible problem (the same body returned up to 20×) without touching the schema or the embedding pipeline.
