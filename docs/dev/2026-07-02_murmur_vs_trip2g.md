# Murmur V2 event bus vs trip2g MCP federation — substrate comparison

> **TL;DR / verdict.** Thesis under test: "a mesh of knowledge bases is more stable, more
> secure, and better for network isolation than an event bus." At the substrate level it
> holds on two of three counts and needs an honest correction on the first. Stability: the
> bus does NOT lose data (proven by test: outbox + ACK + dedupe recovered exactly-once after
> killing the recipient), so the weak form of the thesis is false; the strong form holds
> because the mesh gets idempotency and recovery from its model, while the bus has to build
> them per edge out of SQLite. Security: mesh wins on enforcement point (per-hub ACL applied
> to every query, always on; Murmur's ingress auth ships default-OFF) and on blast radius
> (the federation surface is read-only, a compromised peer can serve lies but push nothing).
> Network isolation: mesh wins on posture options (a provider can be inbound-only, a consumer
> egress-only, an unfederated hub fully air-gapped); a bus participant's coordination IS a
> live connection to shared infrastructure. Where the bus is genuinely better: blocking
> request/reply, presence, large-payload streaming, and cross-org relay where the middle box
> must not read payloads. trip2g should steal those patterns as notes, not adopt a broker.

Scope note: an earlier version of this comparison leaned on trip2g's fleet agent runtime and
its code-exec sandbox. That was the wrong axis. Fleet is a userland layer that could sit on
top of either substrate, and including it let "isolation" be won by a sandbox rather than by
the coordination model. This version compares only the two substrates:

- **trip2g mesh** = MCP federation. Independent hubs; one agent query fans out across peer
  bases; each hub enforces its own per-base access on the query; no shared storage, no
  central broker. State lives at the center of each hub as versioned markdown notes with a
  git-mirror audit trail.
- **Murmur V2** = event/message bus. Agents exchange messages through a transient NATS
  broker; durable SQLite state at every edge (outbox, dedupe, stream reassembly, lease and
  fencing subsystem); E2E ciphertext on the wire, so the broker cannot read payloads.

Sources: hands-on run of [`alexfrmn/mur-mur-v2`](https://github.com/alexfrmn/mur-mur-v2)
v2.4.0 against local NATS 2.10 (evidence below, carried over from the 2026-07-02 evaluation);
trip2g side from `docs/en/user/federation.md`, `docs/en/user/mcp.md`,
`docs/dev/federation_protocol.md`, and `internal/federation/` / `internal/case/mcp/`.

---

## 1. The two substrates

**Mesh (trip2g federation).** A hub is a trip2g instance holding its own notes. A KB-note
(frontmatter field `mcp_federation_kb_url`) registers a peer. The agent talks to one MCP
endpoint; `federated_search` fans out to all configured peers in parallel (2s timeout per
peer, depth cap 3) and merges results. Private peers authenticate with HMAC-SHA256 JWTs
signed by pre-exchanged secrets; each secret (`kid`) is scoped to specific subgraphs on the
answering hub, and that hub filters results at query time. The federated tool set is
`federated_search`, `federated_similar`, `federated_note_html`, `federated_expand`: reads,
all of them. There is no federated write. State never leaves its hub except as query
responses; each hub's notes are versioned (`note_changes`) and mirrored to git (`gitapi`).

**Bus (Murmur V2).** ~12 npm packages under `@murmurv2`. An MCP server gives any MCP client
agent-to-agent messaging over core NATS. Envelope: X25519 + XChaCha20-Poly1305 AEAD payload
with a detached Ed25519 signature over a canonical form, golden-locked by test. Delivery:
at-least-once via a per-agent SQLite outbox with ACK correlation, exponential backoff, DLQ
after 3 attempts, and an unbounded dedupe table on the receiver. Peers are explicit config
entries; discovery presence frames are signed cleartext and candidates are never auto-trusted
(`promoteCandidate` is a deliberate operator step). `murmur_request` blocks the sender until
a reply arrives, a durable store-poll accelerated by a read-only metadata tap on NATS.

## 2. Evidence carried over (what actually ran)

From the 2026-07-02 hands-on run, local only, no paid infra:

| Step | Result |
|---|---|
| Core conformance suite | **49/49 pass** (envelope/ack/presence/stream schema-guard matrices, crypto golden tests) |
| Unit suite | 84/84 pass (outbox ack-timeout, JetStream advisory to DLQ, lease CAS, policy) |
| `murmur_request` round-trip, two MCP servers | Blocked sender unblocked in **3328ms** with `wokenBySignal: true, tapAttached: true` |
| Wire privacy | Outbox rows hold only `payloadCiphertext` + nonce + signature; plaintext only in each side's `local_messages` |
| **Outage survival** | Killed recipient daemon, sent anyway: outbox retried (`attempts: 2, last_error: "ack-timeout"`), recipient restarted, `attempts: 4, status: acked`, **exactly one copy received** (dedupe) |
| Discovery | Signed presence frame verified, folded in as `trusted: false`; **tampered frame rejected**; promote returned an exact peer config entry |
| Streaming | 102 KB in 13 chunks, published shuffled plus one duplicate; reassembly completed, sha256 match |

This matters for honesty: the bus was tested adversarially and did not drop or duplicate a
message end to end. Any version of the thesis that rests on "buses lose data" is refuted by
this run.

## 3. Stability

**Failure modes, mesh.** A dead peer costs the caller a 2-second timeout on that leg of the
fan-out; results from live peers still return. That is the documented behavior, not a hope:
"slow or unreachable peers are skipped; you get results from the rest." A dead hub takes its
own agent offline and makes its base unreachable to peers, but no other hub degrades beyond
losing that one source. There is no component whose failure stops the whole mesh, because
there is no whole-mesh component. Recovery is a restore: the hub's state is one SQLite base,
backed up and git-mirrored, and after restore every query works again with no reconciliation
step, because nothing else in the mesh held derived state about it.

**Failure modes, bus.** While the broker is down, nothing moves. The outbox buffers and
catches up cleanly (proven above), so no data is lost, but coordination halts globally for
the duration. That is the structural difference: the mesh degrades per peer, the bus degrades
all at once. After a partial failure, consistency across N edges is nobody's job; alice's
`local_messages` and bob's can diverge, and there is no single record to reconcile against.
Murmur's own v2.4 changelog shows the cost of earning exactly-once semantics at the edges:
the multi-session double-emit problem needed a whole `SessionLeaseStore` with single-statement
CAS claims and fencing tokens. Solid engineering, and the classic distributed-messaging tax.

**Idempotency.** This is the cleanest asymmetry. A mesh read is idempotent by construction:
run `federated_search` twice, get the same answer twice, no state changed anywhere. Replay a
query, nothing happens. The bus achieves effective exactly-once, and the test proves it, but
it achieves it: outbox plus ACK correlation plus an unbounded dedupe table plus leases. The
mesh does not have the problem. When one design has to engineer a property the other gets
from its shape, the shape is doing real work.

**Partitions.** Mesh: a partitioned peer is just an unreachable peer, 2 seconds and move on;
both sides keep serving their own agents at full function. Bus: a participant partitioned
from the broker keeps its local state and its queue but is coordination-dead until the
partition heals; in-flight blocking `murmur_request` calls hang until timeout.

**Where the mesh is honestly weaker.** Fan-out latency is bounded by the slowest configured
peer up to the timeout, and one misconfigured KB-note silently costs every fanned-out query
2 seconds until someone reads the logs. The JWT design has a 30-second expiry and no replay
cache on the hub (the protocol doc recommends `rid` replay tracking on peers; trip2g itself
does not do it, and the user docs say so). And the mesh has no delivery at all: it cannot
tell anyone anything, it can only answer when asked. Everything push-shaped is out of scope
by design, which is the honest reading of section 6.

**Call: thesis holds, corrected form.** Not "the bus loses data" (it does not), but: the
mesh's stability is a property of the model, the bus's stability is a property of the
engineering bolted around the model. One recovery point per hub versus N edge databases that
each have to be right.

## 4. Security

**Enforcement point.** The mesh enforces at the only place that matters: the hub that owns
the data, on every query, always on. A peer's `kid` is scoped to specific subgraphs at secret
creation time; the answering hub filters every result set through that scope; there is no
mode where the check is skipped. The failure mode is deliberately quiet: a bad, revoked, or
unknown token gets anonymous treatment and empty results, never a 401, so probing does not
even confirm the target exists. Revocation is unilateral and immediate: mark the `kid`
revoked and the peer's next call returns nothing, no coordination needed.

Murmur's trust model is genuinely good at the peer layer: explicit config entries, signed
invite blobs, no auto-trust of discovered candidates, tamper rejection proven by test. But
ingress enforcement of the signed `authToken` sits behind `MURMUR_ENFORCE_AUTH`, and that
flag is **default-OFF**. A security control that ships disabled protects the deployments that
least need protecting. The E2E envelope means a stray subscriber cannot read payloads, which
is real defense, but "cannot read" and "cannot inject accepted traffic" are different
properties, and the second one is the one that defaults off.

**Blast radius of a compromised participant.** Compromise a mesh peer and you get: its own
data (already yours, you are it), plus read access to whatever other hubs granted its
outbound `kid`s, plus the ability to serve poisoned answers to whoever queries it. You get no
write path into anyone else, because none exists: the federated surface is read-only. Poisoned
answers are bad, but they are pull-shaped bad; the victim asked, received, and can stop
asking. Compromise a bus participant and you hold an Ed25519 identity every configured peer
trusts; you can push arbitrary signed, encrypted messages to all of them, and the broker
cannot inspect what you are saying, because not being able to inspect is the design. The E2E
property that is a feature across organizations is exactly what removes the last choke point
during an incident inside one.

**Auditability.** Each hub carries one ordered, versioned record of its state: `note_changes`
with version IDs, plus a git mirror, one place to diff, blame, and revert. To be precise
about what that covers: it audits state, not queries. Federated queries show up in hub logs
and per-`kid` `last_used_at`, which is thinner. Murmur is not record-free either, and saying
so matters: every node keeps full conversation history in `local_messages`. What the bus
lacks is a single ordered record and any point where content-level policy could be applied;
the history is sharded across edges, encrypted in transit, and reconstructable only by
collecting every participant's database.

**Call: thesis holds.** Always-on query-time ACL versus default-off ingress auth; read-only
federation surface versus signed push to every trusting peer; one versioned trail per hub
versus edge-sharded ciphertext history. The mesh's caveats (HMAC only, no mTLS, 30s JWTs
without hub-side replay cache) are real but small against that.

## 5. Network isolation

What each substrate forces on your network, posture by posture:

**Mesh.** A hub that only answers needs exactly one inbound HTTPS endpoint, `/_system/mcp`,
authenticated, and nothing else; it never has to dial out, so a pure provider can sit behind
an allowlist firewall with zero egress. A hub that only consumes needs egress to its peers'
endpoints and no inbound at all. A hub with no federation configured is a complete, fully
functional system with no network dependency beyond its own agent: air-gapped operation is
not a degraded mode, it is just a hub without KB-notes. Traffic is short-lived
request/response with a 2-second ceiling, which is the easiest possible shape to firewall,
inspect, and rate-limit.

**Bus.** Participation means a live connection to the broker; the coordination fabric is
itself a network hop. In fairness, the shape is not bad: NATS clients dial out, so a bus
participant is egress-only by nature, and that passes most corporate firewalls. The MCP
server and agent process touch only local SQLite; strictly, only the daemon needs broker
reach. But the fast path of the flagship tool (`murmur_request`'s read-only tap) needs broker
reach from inside the request path, the connection is persistent rather than transactional,
and the broker is shared infrastructure that every participant's availability now depends
on. Air-gapped operation does not exist: a Murmur node with no broker is a mailbox with no
post office, its outbox filling until connectivity returns.

**Call: thesis holds.** Both can run egress-only; only the mesh can run inbound-only or with
no network at all. The mesh offers a menu of postures and lets each hub pick independently;
the bus offers one posture, reachable-broker, and everyone takes it.

## 6. Where the bus genuinely wins

Four workloads where the bus model is right and the mesh model is a workaround, framed
substrate against substrate:

- **Blocking interactive request/reply.** `murmur_request` measured a 3.3s round-trip
  between two live LLM sessions, sender suspended until the answer arrived. The mesh has no
  primitive for "ask and wait": federation is pull-query over existing state, so agent A
  cannot suspend on agent B producing something new. For live pair-work ("review this diff
  now, I'm waiting") the bus shape is simply correct.
- **Presence and capability discovery.** Signed presence frames with TTLs, a capability
  query (`queryCandidates(capability: "code-review")` hit in the live run), and an explicit
  promote step. The mesh has static topology (KB-notes and the federation graph page with
  its edge-status crawl) but nothing that answers "who is alive with what capabilities right
  now."
- **Large-payload streaming.** 102 KB in 13 chunks, shuffled with a duplicate injected,
  reassembled with a matching sha256. Notes are versioned documents; they are the wrong tool
  for a 100 MB artifact handoff, and pretending otherwise would bloat the version history
  for no benefit.
- **Cross-org confidential relay.** Two organizations' agents exchanging messages the
  transit infrastructure provably cannot read. The mesh cannot offer this: a trip2g hub
  reads everything it serves, that is what makes scoping and audit work. Hub-can't-read is a
  real feature across org boundaries and an anti-feature inside one instance; the bus picked
  the cross-org side of that trade and executes it well.

## 7. Verdict, and what the mesh should steal

The thesis survives at substrate level in its strong form: the mesh is more stable because
its stability is structural rather than engineered, more secure because enforcement is
always-on at the data owner with a read-only surface, and clearly better for network
isolation because it supports postures the bus cannot express. The weak form ("the bus loses
messages") is refuted by the bus's own test evidence and should not be argued.

Three things worth taking from Murmur, none of which require a broker, and none of which
exist in trip2g today:

1. **A blocking `ask_note` request/reply over notes.** Murmur's real innovation is not NATS,
   it is `murmur_request` = send + durable store-poll + event tap. trip2g has the parts:
   `updateNotes` (send), `noteChanges` SSE (the tap), versioned notes (the store). An
   `ask_note(path, payload, timeout)` tool that writes a question note, subscribes to SSE
   filtered to the answer path, blocks, and falls back to polling would close the bus's
   biggest UX win with zero new infrastructure, and the conversation would remain notes:
   versioned, scoped, auditable. Not built; this is a proposal.
2. **Heartbeat presence notes.** A `presence/<peer>.md` convention with a TTL in frontmatter
   would give "who is alive, offering what" as ordinary queryable notes. The federation
   graph page already crawls link status; liveness is the missing half. Not built.
3. **The explicit-promote trust ceremony.** Federation secrets are already exchanged
   manually, which is the right instinct. Murmur's discovery design goes further: candidates
   are observed, verified, queryable, and trusted only by deliberate operator action, with
   tampered announcements rejected fail-closed. That is the reference to copy if trip2g ever
   grows peer discovery.

Not worth adopting: the broker (a component whose reliability Murmur itself had to rebuild
out of SQLite at every edge), JetStream, or E2E encryption inside one instance (the hub
reading everything is what makes scoping, audit, and replay possible). If confidential
cross-org agent dialogue ever becomes a requirement, bridging to a Murmur-style bus at the
federation boundary beats weakening the KB model, and Murmur's conformance-suite-backed wire
protocol makes that bridge a bounded job rather than a research project.

---

*Murmur run artifacts from the 2026-07-02 evaluation: scratchpad `mur-mur-v2/` clone at
v2.4.0, NATS 2.10-alpine containers `murmur-nats` (:4222, token) and `murmur-nats-open`
(:4322), daemon logs `alice-daemon.log` / `bob-daemon.log`, drivers `mcp-drive.mjs` and
`presence-stream-demo.mjs`, SQLite evidence in `.alice/murmur.db` / `.bob/murmur.db`.*
