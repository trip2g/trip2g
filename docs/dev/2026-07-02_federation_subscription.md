# Change subscription across federation: feasibility and design

> **TL;DR / verdict.** Yes, a subscriber on hub A can get live change events for notes on
> peer hub B, and we should build it as **cursor pull, not push** (Option 1: a
> `changes_since(cursor)` federated read, optionally upgraded to long-poll). Effort **S**
> for the plain-poll MVP, **M** with long-poll. True push federation (Option 2) is **L**
> and, more importantly, it breaks the one property our Murmur comparison identified as
> the mesh's structural advantage: no participant has to hold a live connection to anyone.
> A relay bridge (Option 3) is **M** but quietly reintroduces a cross-hub write path, the
> exact thing the read-only federation surface protects us from. The honest summary:
> everything push needs (cursor, catch-up query, per-event ACL) pull needs too; push then
> adds a streaming transport, connection lifecycle, and a new auth model on top. Build the
> shared part, skip the expensive part.

Answer to the `ask_note` / `memcli wait` question: **local = SSE `noteChanges`, remote =
federated cursor poll.** Same wait loop, two backends behind one flag.

## 1. What exists today (verified against code)

**Local push.** `noteChanges` is a GraphQL SSE subscription
(`internal/graph/schema.resolvers.go:3362`). Events originate at save time:
`HandleLatestNotesAfterSave` builds a `notebus.Batch` and publishes it
(`cmd/server/webhooks.go:73-89`), hides publish from `internal/case/hidenotes/resolve.go:73`.
The bus is purely in-memory (`internal/notebus/notebus.go`), per-subscriber buffer of 64,
and it **drops batches when the buffer is full** (`notebus.go:112`). No replay, no
persistence: a reconnecting subscriber misses whatever happened while it was away. Events
are enriched from `LatestNoteViews` with `versionId` from the `note_versions.id` space
(`internal/graph/note_changes.go:22-31,48-57`), which matters below: that id is a global
monotonic cursor we already expose to clients.

Per-subscriber ACL exists but is **not on this branch yet**: commit `f36e9fda`
(`feat/notechanges-acl`, PR #96) adds `filterChangesByACL`, which gates every emitted
change through `env.CanReadNote`, fail-closed, and classifies credentials (instance API
key = firehose; admin session, shortapitoken, personal session = per-note ACL). This is
the exact primitive a cross-hub stream would need, so the ACL story composes; only the
transport question remains.

**Federation pull.** Strictly request/response. The outbound client is a single-shot
`fasthttp` POST with `DoTimeout` and a 2s default (`internal/federation/client.go:118,141-167`);
it cannot stream at all. Fan-out runs the peers in parallel and tolerates dead ones
(`internal/case/mcp/federation.go:25-50`). Auth is an HMAC-SHA256 JWT with a **30-second
TTL** (`internal/case/mcp/federation_helpers.go:48`, skew at `:26`); the serving hub maps
`kid` to allowed subgraphs (`verifyInbound`, `federation_helpers.go:80-124`) and every
note the peer sees passes `canReadMCPNote` -> `canreadnote.ResolveWithSubgraphs`
(`internal/case/mcp/resolve.go:584-592`). `federated_graphql_request` (flag-gated,
default off) restricts peers to `note`, `search`, `similarNotes`, `viewer`
(`internal/case/mcp/graphql_tools.go:65-70`). There is no changes-since query anywhere
on the federated surface today.

So the gap is precisely: **no durable "what changed after cursor X" read, and no
streaming transport.** The first is cheap. The second is not, and we may not need it.

## 2. Options

### Option 1: cursor pull (recommended, S; long-poll upgrade M)

The serving hub B gains one read: `changes_since(cursor, limit)` where the cursor is a
`note_versions.id`. Implementation is a sqlc query (`select id, path_id, version from
note_versions where id > ? order by id limit ?`), a join to paths, and a per-note
`canReadMCPNote` filter, identical in shape to how `search` filters results
(`internal/case/mcp/resolve.go:522-528`). Exposed either as a new MCP tool (fits the
`federated_*` family in `federation_handlers.go`) or as one more allowlisted root field
in `graphqlFederatedRootFields`. Response: `[{path, versionId, event}]` plus the next
cursor. Content is deliberately not included; the consumer fetches it with the existing
`federated_note_html`.

The subscriber on A polls at an interval. Latency = poll interval (5-30s is fine for
agent workloads; the kanban live-sync taught us humans notice above ~2s, agents do not).
Cost per empty poll is one authenticated POST, well inside the existing 2s budget.

**Long-poll upgrade:** the serving hub parks the request on a local `SubscribeNoteChanges`
for up to `wait_seconds` and answers early when something matching arrives, else returns
an empty page with the same cursor. Server-side this reuses the notebus machinery the SSE
resolver already uses; wire-side it stays request/response. Two real costs: the federation
client's per-call timeout must be raised for this one call (today it is a single
`c.timeout`, `client.go:118`), and B now holds a parked request per waiting peer, which is
mild but not free. That is why long-poll is the upgrade, not the MVP.

### Option 2: true push (L, and it spends our best property)

Peer B streams matching events to subscriber A over a held connection. Inventory of what
it requires against current code:

- **Transport.** The federation client cannot stream (`DoTimeout` buffers the whole
  response). A push channel needs a second client path (SSE reader or chunked reader),
  plus proxy behavior (Caddy timeouts, LB idle cuts) handled on both hubs.
- **Auth lifecycle.** The 30s JWT is designed for one-shot requests. A persistent stream
  needs re-auth on the wire or per-connection session tokens, a new model either way.
- **Per-event ACL.** The good news: this part is already built. A stream handler on B
  would subscribe to notebus and run `filterChangesByACL` with a federation ctx, and
  `canReadMCPNote` already branches to `ResolveWithSubgraphs` when federation auth is in
  ctx (`resolve.go:588-589`). Solved by composition.
- **Backpressure and missed events.** notebus drops on a full buffer (`notebus.go:112`).
  Over a WAN link that is a guarantee of loss, so push needs a resume cursor anyway,
  which means building the exact `changes_since` query from Option 1 first, then the
  stream on top of it.
- **Reconnect logic** on A: detect drop, re-auth, replay from cursor, dedupe.

That last pair is the punchline: **push is a superset of pull.** You build Option 1 and
then keep building. And the prize for the extra work is negative on the isolation axis
(section 3).

### Option 3: relay bridge (M in code, standing infra in ops)

A process near B subscribes locally (SSE with a scoped credential; with PR #96 a
shortapitoken subscription gets exactly the per-note ACL we want) and forwards to A,
either by writing notes on A via `updateNotes` or by calling a webhook on A. This is the
fleet-agent pattern and it works today with zero server changes, which is its honest
appeal: it is the only option shippable without touching trip2g.

Costs: the relay is a daemon somebody runs and monitors; it needs egress to A and
credentials on both hubs; a relay crash loses events unless it keeps its own cursor,
which again means wanting `changes_since` on B for catch-up. And the note-writing variant
gives A inbound cross-hub **writes**. The Murmur comparison scored blast radius in the
mesh's favor precisely because the federation surface is read-only
(`docs/dev/2026-07-02_murmur_vs_trip2g.md`, section 4); a compromised peer can serve lies
but push nothing. A relay with `updateNotes` rights on A deletes that sentence.

## 3. The three judgments

**Isolation.** The mesh's win (Murmur doc, section 5) is that a provider can be
inbound-only, a consumer egress-only, and nobody holds a live connection to shared infra.
Option 1 preserves this exactly: the consumer initiates every exchange, connections are
short-lived (bounded even in long-poll flavor), the provider stays a passive HTTPS
endpoint with zero per-subscriber state. Option 2 breaks it: A holds a persistent egress
stream, B holds N open inbound streams and per-connection state; we would be rebuilding
the bus posture we argued against, one edge at a time. Option 3 is mixed: B stays clean,
but A must accept inbound traffic from the relay, and the relay itself is a small piece
of shared infrastructure with credentials to two hubs.

**ACL correctness.** The rule: an event crosses the boundary only if the same peer would
receive that note on a normal federated query. Option 1 gets this by construction, since
every `changes_since` row passes `canReadMCPNote` under the caller's federation ctx, the
same gate as `search`. Option 2 gets it via `filterChangesByACL` with federation ctx,
also correct, once PR #96 lands. Option 3 is only as correct as the credential the relay
uses on B; a scoped shortapitoken is right, an instance API key (firehose,
`noteChangesAuth` returns bypassACL=true for it) would leak everything B knows to
whatever the relay forwards. One subtlety for all options: when a peer is **granted new
subgraph access**, changes in those subgraphs from before the grant sit below its cursor
and will not replay. Document it: after an ACL grant, the consumer resets its cursor or
runs a backfill `search`. Fail direction is closed, which is the right direction.

**Durability.** SSE has no replay and notebus drops under pressure, so any cross-hub
design that leans on the live stream alone will lose events on the first bad night.
Murmur solved this with an outbox plus ACK plus a dedupe table per edge, the
"distributed-messaging tax" the comparison doc describes. The cursor gives us the same
guarantee for free: `note_versions` **is** the outbox, already written transactionally
with the change itself, and the consumer-held cursor is the ACK. No per-subscriber server
state, no dedupe table (ids are monotonic, "greater than cursor" is the dedupe), restarts
on either side are a non-event. This is the same shape as Murmur's `murmur_request`
("durable store-poll accelerated by a tap"): the store-poll is the correctness layer, the
tap is a latency optimization. Locally our tap is SSE; remotely the long-poll is the tap.

**Known gap, stated plainly: hides.** `note_versions` records create/update only; hiding
sets `note_paths.hidden_by` and never writes a version row, so a pure version-id cursor
misses remove events. MVP answer: the consumer discovers hides when `federated_note_html`
stops returning the note. Real answer, if remote hide events turn out to matter: a small
append-only change-log table (path_id, event, created_at, id) written next to the
existing publish site in `HandleLatestNotesAfterSave` and `hidenotes`, and the cursor
moves to that table. That is the one piece of genuinely new infrastructure on the road
map, and it is optional for v1.

## 4. Effort sizing

| Option | Size | What it buys | What it costs |
|---|---|---|---|
| 1a. Plain cursor poll | **S** | Durable cross-hub change feed, correct ACL, no new posture | Latency = poll interval; hides invisible in v1 |
| 1b. + long-poll | **M** | Seconds-level latency, still request/response | Parked requests on B; per-call timeout plumbing |
| 2. True push | **L** | Sub-second latency | Streaming transport, auth lifecycle, reconnect/replay (which is Option 1 again), and the isolation property |
| 3. Relay bridge | **M** | Ships with zero server changes | A daemon to run; write path into A; ACL only as good as the relay's credential |

MVP = 1a: one sqlc query, one federated tool with the standard ACL filter, one poll loop
in the consumer (memcli/fleet). No migration, no new tables, no new connection shapes.

## 5. Minimal implementation sketch (Option 1a)

Server (hub B):
1. sqlc read in `queries.read.sql`: `NoteVersionsSince(id, limit)` joining
   `note_versions` to `note_paths` for the current path, ordered by `nv.id`.
2. New handler `handleFederatedChangesSince` in `internal/case/mcp/federation_handlers.go`
   following the `callFederatedSingleKB` pattern; on the serving side a local
   `changes_since` tool that maps rows to `{path, versionId, event}` and filters each
   through `canReadMCPNote` (as `filterSearchResults` does at `resolve.go:507-531`).
   `event` is `create` when `version == 1`, matching `cmd/server/webhooks.go:62-64`.
3. Response carries `next_cursor` = max id seen **before** ACL filtering, so denied notes
   still advance the cursor and cannot wedge it.
4. Client method on `internal/federation/client.go` plus the `model.Federation` interface
   entry; recursion (`FederatedChangesSince`) can wait.

Consumer (hub A side, memcli/fleet `wait`):
5. `wait --kb <kb_id> --path <glob>`: if the target is local, subscribe to `noteChanges`
   SSE as today; if the target is a federation KB, loop `changes_since` with a persisted
   cursor (a file next to memcli state is enough), interval configurable, default 10s.
6. First call with no stored cursor starts from the current head (fetch latest id, do not
   replay history) unless `--from-beginning` is passed.

Later, in order of value: 1b long-poll (`wait_seconds` param, notebus parking on B),
change-log table for hide events, and only then reconsider push if a workload actually
demands sub-second cross-hub latency. Nothing today does; the `ask_note` flow is
agent-paced, and a 10s poll is invisible at that pace.
