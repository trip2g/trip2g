# Federation hot cache — spec

> **Status: deferred — not worth building yet (no concrete need).** The realistic case is already covered
> more cheaply by hand: an owner who knows their node is going down can hand a trusted peer a replica in
> advance. The automated, continuously-syncing pull machinery below only earns its keep for *unplanned* or
> *permanent* disappearance, or for hands-off freshness — none of which is pressing now. Keep this as a
> design reference; revisit when a live federated peer or the agent runtime (`agent.md`) actually needs
> continuous, hands-off caching over a neighbor KB. The design below (admin-only, isolated cache tables)
> stands — it is simply not scheduled.

**TL;DR.** A continuously-updated, **admin-only** local copy ("hot cache") of a neighbor's shared federated
**subgraph**, kept in its **own cache tables + a dedicated vector index** — never in your note-store. Two
purposes: don't fear losing the source (a live local backup that auto-refreshes while the neighbor is up),
and let the admin search / vector-retrieve a neighbor's knowledge locally — including when the neighbor is
down or **gone for good**. Single-writer: the owner writes; peers hold a read copy.

**Hard constraint — never re-transmit a neighbor's private data.** The cache exists for the admin's own
consumption only. Consequences that shape the whole design:
- **No public serving** of a neighbor's content, ever.
- **No permission-role machinery.** Admin-only ⇒ there is nobody to regulate access for, so the entire
  cross-instance access-policy problem (mirroring the origin's paywall, who is a subscriber, etc.)
  disappears.
- **No forwarding of private content to a third peer.** Transitive propagation (phase 2) is allowed **only**
  for content the origin explicitly marked public.

**Storage decision — separate cache tables + a dedicated vector index (NOT materialized into `note_paths`).**
Because "private foreign data must never leak" is a hard invariant, **physical isolation beats flag
discipline**: if the replica lives in its own `replica_*` tables, the publishing / serving / sitemap / git
mirror / webhook / Telegram / public-site-search pipelines **structurally cannot see it** — there is nothing
to suppress and nothing to forget. The only cost is that the cache's *one* consumer (admin search + vector)
must read the cache store explicitly; at admin-only scope that is far cheaper and safer than gating five
outbound pipelines on a flag. (This reverses an earlier "materialize into the note-store" idea, which was
attractive for "free" indexing but required suppressing every outbound handler and carried real leak risk.)

## Why — and why not just a whole-DB replica
The value is **the admin's federated search + a local vector base over a neighbor's subgraph**, plus a
**self-refreshing backup** so you don't lose the source:
- **A live proxy can't be embedded.** Today `federated_search` proxies live to the neighbor's origin. So
  when the neighbor is down, search over it breaks; and you **cannot vector-index** its content — you can
  only embed text you hold locally.
- **A whole-DB replica (LiteFS / Litestream) doesn't solve this.** Those copy *your own whole DB* for *your
  own nodes*. They don't pull a *neighbor's partial* subgraph across a trust boundary, nor give you a local,
  embeddable copy of just the part a peer shared.
- **So the hot cache is a local materialization of a neighbor's subgraph** — but kept isolated. Once it's
  local you can embed it, search it, and you still have it if the neighbor vanishes. Surviving the
  neighbor's outage or permanent disappearance is a consequence of holding a local copy.

This is the read half of a single-writer knowledge network on cheap/disposable peers: the knowledge survives
any one node because trusted peers hold copies (see `../en/thoughts/surviving-node-loss.md`). It pairs with
the agent runtime (`agent.md`): an admin's fleet sidecar can retrieve over a cached neighbor KB.

## What already exists (reuse)
- **Federation trust + transport.** HMAC-SHA256 JWT (`alg=HS256`, `kid`, `exp=iat+30s`); secrets in
  `federation_secrets` (`kid`, `secret_crypt`, `kb_url`, `revoked_at`) + per-subgraph scope in
  `federation_secret_subgraphs`. Outbound: `internal/federation/client.go`. Inbound:
  `internal/case/mcp/federation_helpers.go:verifyInbound()` → `(kid, allowed_subgraphs)`; entry via
  `internal/case/mcp/endpoint.go`. `federated_search` proxies live today; the hub stores no remote data.
- **Versioning cursor.** `note_versions.id` is a global autoincrement — a natural high-watermark cursor
  (migration `20250515071315_add_id_to_note_versions.sql`). `note_paths.latest_content_hash`;
  soft-delete via `note_paths.hidden_by` / `hidden_at`.
- **Subgraph membership** in frontmatter (`subgraph` / `subgraphs` → `NoteView.SubgraphNames`,
  `internal/model/note.go`).
- **Embedding mechanics to mirror (not reuse directly).** Chunking via `mdchunk.Split`, OpenAI passage
  embeddings, brute-force dot-product scoring (`internal/case/sitesearch/resolve.go`,
  `internal/case/backjob/generatenoteversionembedding/resolve.go`). The cache builds a *parallel* vector
  index over its own tables using the same primitives.

So trigger, auth, scope, the cursor, and the embedding primitives exist. Missing for v1: a pull-diff
endpoint, the cache store + its own sync loop and vector index, an admin search over it, and a pull grant.

## v1 design (admin-only, isolated cache)

### Grant
The origin grants a peer replica-pull rights via a new `can_pull_replica` flag on its `federation_secrets`
row for that peer. Scope stays gated by `federation_secret_subgraphs`.

### Origin — pull-diff endpoint
`POST /api/federation/v1/pull_diff { subgraph, since_version_id }`, authed by the existing HMAC federation
JWT. New `Endpoint` under `internal/case/federation/pull_diff/`, registered in
`internal/router/endpoints_gen.go`. It:
1. `verifyInbound()` → `(kid, allowed_subgraphs)`; reject if `subgraph` ∉ scope or `can_pull_replica` unset
   (403); revoked kid → 401.
2. Queries `note_versions where id > since_version_id` for the subgraph's paths + hidden paths since the
   cursor (tombstones). Membership lives in frontmatter, so maintain a derived
   `federation_note_subgraphs(path_id, subgraph_id)` table (updated by the frontmatter-change hook) for an
   O(1) join instead of re-parsing notes per request.
3. Returns upserts + `deleted_paths` + a signed envelope:
   ```json
   { "subgraph": "...", "since_version_id": N, "last_version_id": M, "has_more": false,
     "upserts": [{ "path", "version_id", "content", "content_hash", "updated_at" }],
     "deleted_paths": ["..."], "root_sig": { "type": "hmac", "value": "..." } }
   ```
   `since_version_id: 0` = full pull. `has_more` paginates. Origin lost the cursor (vacuum) →
   `{ "full_resync_required": true }` → puller resets to 0. Sign reuses `hmacSHA256()` +
   `subtle.ConstantTimeCompare`.

### Puller — cache store + sync loop
Separate tables, owned entirely by the puller:
- `subgraph_replica_cursors(origin_kid, subgraph, last_version_id, sync_interval_seconds, last_synced_at)`
  — one row per cached subgraph; default interval 5 min, pull-poll (self-healing).
- `replica_notes(origin_kid, subgraph, path, version_id, content, content_hash, updated_at)`,
  PK `(origin_kid, subgraph, path)`. The `origin_kid` prefix makes two peers (or a peer + a local note of
  the same path) collision-free by construction.
- `replica_deleted_paths(origin_kid, subgraph, path, hidden_at)` — tombstones.
- `replica_note_chunks(origin_kid, subgraph, path, chunk_idx, content, content_hash, embedding)` — the
  **dedicated cache vector index**, mirroring `note_version_chunks`.

Each tick: pull → recompute and constant-time-compare `root_sig` (reject tampered diffs) → upsert
`replica_notes`, apply tombstones → re-chunk + re-embed changed paths into `replica_note_chunks`
**immediately** (don't wait for any cron) → advance the cursor only after a successful write.

### Admin consumer surface
An **admin-scoped** federated search reads the cache: vector dot-product over `replica_note_chunks` (+
optional keyword over `replica_notes`). The existing `federated_search` is repointed to **prefer the local
cache** (fast, survives origin-down) and may fall back to a live proxy for non-cached subgraphs. No new
public surface; no roles.

### Isolation by construction (what we no longer need)
Because the cache is not in `note_paths`/`note_versions`, none of these can ever see it — so there is nothing
to gate: git mirror (`internal/gitapi/materialize.go`), change_webhooks / agents, Telegram publish, sitemap,
custom-domain routing, and the **public site search / vector** (Bleve from `AllLatestNotes`,
`LatestNoteChunks`). The neighbor's content cannot leak through any of them. This is the structural payoff of
separate tables.

## Out of v1 — hard limits (not just deferred)
- **Public serving of a neighbor's content** — forbidden by the never-re-transmit invariant.
- **Cross-instance permission roles** — unnecessary at admin-only scope; do not build them.
- **Forwarding private content to third peers** — forbidden. Only origin-marked-public content may ever
  propagate further (phase 2).

## Phase 2 — and forward-compat seams to land in v1
- **`root_sig` is a versioned envelope from day one** (`{type, value}`), so phase 2 can upgrade to a Merkle
  attestation (`{type:"merkle", merkle_root, signature, proof}`) — same field, stronger verification.
- **Adoption** (origin gone for good). Promote a cached subgraph to your own notes: a real migration from
  `replica_*` into `note_paths`/`note_versions` (deliberately explicit, not a flag flip, since the stores
  are separate). Only legitimate for content you are permitted to keep/own; private foreign data stays a
  private admin backup, not a republished vault.
- **Transitive (Merkle) verification — public content only.** A third peer can verify a cached subgraph
  without the origin's shared secret if the origin signed a Merkle root with an asymmetric `origin_pubkey`
  (Ed25519) added to `federation_secrets`: the holder serves a note onward with an inclusion proof + the
  signed root; the third peer verifies with the origin's public key, even if the origin is dead. This is
  what would turn the cache into a P2P preservation network — but **gated to content the origin marked
  public**, never private subgraphs. Background: Certificate Transparency (RFC 9162) is the closest
  production model of a signed Merkle root.

## Options compared (transport)
| | A: pull-poll (recommended) | B: MCP `pull_diff` tool | C: push-on-change webhook |
|---|---|---|---|
| New auth surface | none (reuses HMAC JWT) | none | friend must expose an inbound endpoint |
| Origin complexity | one handler + a few queries | one MCP tool case | delivery queue + retries |
| Freshness | interval (default 5 min) | same | sub-second |
| Self-healing | yes (next poll) | yes | needs retry logic |
| Timeout risk | low (REST) | real (short MCP client timeout) | n/a |
| v1 | **yes** | bolt-on later | no |

## Test plan
1. **Auth gates:** kid without `can_pull_replica` → 403; flag set but subgraph out of scope → 403; revoked
   kid → 401.
2. **Incremental correctness:** write 3 notes, pull from 0 → all 3; write a 4th, pull from prior max → only
   the 4th.
3. **Tombstones:** hide a note → it appears in `deleted_paths` and is removed from `replica_notes` +
   `replica_note_chunks`, not in `upserts`.
4. **Pagination:** insert > batch size → `has_more: true`, second pull returns the rest.
5. **Integrity rejection:** tamper one byte → puller rejects on `root_sig` mismatch.
6. **Full-resync:** origin returns `full_resync_required` → puller resets the cursor to 0 and re-pulls.
7. **Isolation (the core invariant):** sync a subgraph → assert it is **absent** from `note_paths`, the
   public site search, the public vector index, the sitemap, the git mirror, and that no change_webhook /
   agent / Telegram publish fired. It must appear **only** in the admin federated search.
8. **Vector freshness:** sync after boot → the new content is immediately retrievable via the admin cache
   vector search (no restart, no cron wait).
9. **Collision safety:** two origins sharing a path, and a local note of the same path, all coexist (keyed
   by `origin_kid`), none overwrites another.

## Composition
Pairs with the agent runtime (`agent.md`): an admin fleet sidecar retrieves (vector/keyword) over a cached
neighbor KB and keeps working when the origin is down or gone. Adoption (phase 2) is how a surviving peer
keeps *permitted* knowledge alive permanently; private caches stay private backups.
