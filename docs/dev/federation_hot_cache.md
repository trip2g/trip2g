# Federation hot cache — spec

A trusted peer can pull a **read replica** of a subgraph and serve it (a hot cache), so reads survive the
origin node being unavailable. Incremental version-diff sync, configurable freshness, signed integrity.
Everyone other than the owner reads; the owner writes (single-writer by design).

## What already exists (reuse)
- **Federation:** HMAC `federation_secrets` (a shared kid per peer pair) + per-subgraph scope
  (`federation_secret_subgraphs`); `federated_search` proxies live; the hub stores **no** remote data today.
- **Versioning:** `note_versions.id` is a global autoincrement — a natural high-watermark **cursor**.
  `note_paths.latest_content_hash` per note; soft-delete via `note_paths.hidden_by` / `hidden_at`.
- **Subgraph membership** is declared in note frontmatter.

So the trigger, auth, scope, and a monotonic cursor already exist. Missing: a pull-diff endpoint, a
replica store on the puller, serve-stale, and a replica-pull grant.

## Model
- **Grant:** a peer (one with a `federation_secrets` row) is granted replica-pull rights via a new
  `can_pull_replica` flag on `federation_secrets`. Scope stays gated by `federation_secret_subgraphs`
  (no extra scope surface).
- **Incremental pull (recommended — Option A, pull-poll):** `POST /api/federation/v1/pull_diff
  { subgraph, since_version_id }`, authed by the existing HMAC federation JWT. The origin verifies the
  kid, checks `can_pull_replica` + subgraph scope, queries `note_versions where id > since_version_id`
  for that subgraph's paths, and returns:
  ```json
  { "subgraph": "...", "since_version_id": N, "last_version_id": M, "has_more": false,
    "upserts": [{ "path", "version_id", "content", "content_hash", "updated_at" }],
    "deleted_paths": ["..."], "root_sig": "hmac-sha256:..." }
  ```
  `since_version_id: 0` = full pull. `has_more` paginates (cap the batch, re-poll with the last id).
  Tombstones: hidden paths since the cursor come back in `deleted_paths`. If the origin no longer has the
  cursor in history → `{ "full_resync_required": true }` and the puller resets to 0.
- **Puller store:** the friend keeps the replica locally (`replica_notes` + `replica_deleted_paths`) plus a
  cursor (`subgraph_replica_cursors`: kid, subgraph, `last_version_id`, `sync_interval_seconds`).
- **Sync frequency:** a configurable interval per cursor (default 5 min), pull-poll (self-healing — a
  missed poll catches up on the next tick). Push-on-change is a later option (lower latency, more machinery).
- **Serve-stale / failover:** when the origin is unreachable, the friend serves its replica read-only with
  a freshness field (`last_synced_version_id`, `last_synced_at`, `is_stale`). The federation layer falls
  through to the local replica on an upstream error instead of returning an error.
- **Integrity:** the origin signs the diff with the shared HMAC secret —
  `root_sig = HMAC(secret, subgraph + "|" + last_version_id + "|" + sha256(upserts sorted by path))`. The
  puller recomputes and constant-time-compares before storing/serving; a tampered diff is rejected. A
  friend cannot forge a diff to a third party (no secret). When the Merkle-root attestation lands, the
  `root_sig` field upgrades to a Merkle proof — same field, stronger verification.

## Options compared
| | A: pull-poll (recommended) | B: MCP `pull_diff` tool | C: push-on-change webhook |
|---|---|---|---|
| New auth surface | none (reuses HMAC JWT) | none | friend must expose an inbound endpoint |
| Origin complexity | one handler + a few queries | one MCP tool case | delivery queue + retries |
| Freshness | interval (default 5 min) | same | sub-second |
| Self-healing | yes (next poll) | yes | needs retry logic |
| Timeout risk | low (REST) | real (the MCP client has a short timeout) | n/a |
| v1 | **yes** | bolt-on later | no |

## Build vs reuse
- Migration: add `can_pull_replica` to `federation_secrets`; add `subgraph_replica_cursors`,
  `replica_notes`, `replica_deleted_paths` (the last three live on the puller).
- Read queries: list note_versions since a cursor for a subgraph's paths; list hidden paths since a time.
- HTTP handler `POST /api/federation/v1/pull_diff` (sign/verify reuse the existing federation HMAC helpers).
- A replica-sync client (background ticker per cursor) that pulls, verifies, and upserts.
- Serve-stale fallback in the federation resolve path.

## Test plan
1. **Auth gates:** a kid without `can_pull_replica` → 403; with the flag but no subgraph scope → 403; a
   revoked kid → 401.
2. **Incremental correctness:** write 3 notes, pull from 0 → all 3; write a 4th, pull from the prior max →
   only the 4th.
3. **Tombstones:** hide a note → it appears in `deleted_paths`, not in `upserts`.
4. **Pagination:** insert > batch size → `has_more: true`, second pull returns the rest.
5. **Integrity rejection:** tamper one byte in `upserts` → the puller rejects on `root_sig` mismatch.
6. **Serve-stale:** kill the origin → the friend serves the replica read-only with `is_stale: true`; a
   tampered cached blob is rejected.
7. **Full-resync:** origin returns `full_resync_required` → the puller resets the cursor to 0 and re-pulls.

## Composition
Pairs with the agent runtime (`dev/agent.md`): a fleet sidecar on another machine can hold a hot cache of
a shared knowledge base and keep serving reads when the origin is down. Serve-stale is also the read half
of running a single-writer knowledge network on cheap/preemptible nodes.
