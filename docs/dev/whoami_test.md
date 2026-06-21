# Whoami / Federation E2E — structure and example queries

A **structural** example of federation-e2e for trip2g: a set of linked KBs, notes
inside them, and a check of federated search + isolation. **No agents** (no Telegram
bots, no LLM) — only KBs, notes, subgraphs, and federation edges. This is the
"skeleton" you can seed via SQL/GraphQL and run as an ordinary e2e.

The prototype is the simplepanel stand (`scripts/fed_e2e/mesh.json` +
`cloud_whoami_test.py`), where the same relations run across 10 instances. Here the
same relations are described as a pure trip2g structure.

---

## 1. Primitives (what is what)

| Primitive | What it is in trip2g |
|-----------|----------------------|
| **KB** | knowledge base, a graph node. Its own `kb_id`, its own `/_system/mcp` endpoint. |
| **Note** | a markdown note inside a KB. Indexed for `search`. |
| **subgraph** | a note's visibility label (frontmatter `subgraph: <name>`). Federation returns only the subgraphs the edge allows; a note in a private subgraph never leaves. |
| **federation edge** | a directed link `from → to`. The `from` side holds the **federation_secret** (`kid`) of peer `to`; this grants `from` the right to call `to` via `federated_*`. See [federation_protocol.md](federation_protocol.md). |
| **fan-out** | `federated_search` without `kb_id` broadcasts the query to all **directly connected** KBs — **one level only** (see §7.3). Depth comes from materialized `federation-index.md`, not live recursion. |

Hub limits (from `federation_protocol.md`): recursion `MCP_FEDERATION_MAX_DEPTH=3`,
per-peer timeout `MCP_FEDERATION_FANOUT_TIMEOUT=2s`.

**Key point:** an edge = "who may search whom", a subgraph = "which notes are visible
while doing so". These two axes are independent — and that is exactly what the test
checks.

---

## 2. Topology (10 KBs, 11 edges)

```
owner
├── company-a-hub
│   ├── dept-a-hub
│   │   └── sub-a ←───────────────┐
│   └── dept-b-hub                │
│       └── sub-b ←──────┐        │
└── company-b-hub        │        │
    ├── company-site ────┼────────┘   (company-site → sub-a)
    └── sub-b ───────────┘            (company-b-hub → sub-b)

boss-a → company-a-hub      (enters company A's graph from the side)
boss-b → company-b-hub      (enters company B's graph from the side)
```

### Nodes

| kb_id | Role | Position | Type |
|-------|------|----------|------|
| `owner` | owner | root, sees both companies | — |
| `boss-a` | boss-a | top management A | — |
| `boss-b` | boss-b | top management B | — |
| `company-a-hub` | company-a-hub | company A hub | hub |
| `company-b-hub` | company-b-hub | company B hub | hub |
| `dept-a-hub` | dept-a-hub | department A hub | hub |
| `dept-b-hub` | dept-b-hub | department B hub | hub |
| `company-site` | company-site | company B site | hub |
| `sub-a` | sub-a | leaf (2 parents: dept-a-hub, company-site) | — |
| `sub-b` | sub-b | leaf (2 parents: dept-b-hub, company-b-hub) | — |

"hub" = nodes that run `federated_search` themselves and aggregate
`#federation-index`.

### Edges (`from → to`, directed)

```
owner          → company-a-hub
owner          → company-b-hub
boss-a         → company-a-hub
boss-b         → company-b-hub
company-a-hub  → dept-a-hub
company-a-hub  → dept-b-hub
company-b-hub  → company-site
company-b-hub  → sub-b
dept-a-hub     → sub-a
dept-b-hub     → sub-b
company-site   → sub-a
```

Features that make the graph interesting for the test:
- **Diamonds / double parents:** `sub-a` is reachable both via `dept-a-hub` and via
  `company-site`; `sub-b` — via `dept-b-hub` and via `company-b-hub`.
- **Side entry:** `boss-a` sees company A's graph but **not** company B.
- **Depth 3:** `owner → company-b-hub → company-site → sub-a` — exactly at the
  `MCP_FEDERATION_MAX_DEPTH` limit.
- **Leaves:** `sub-a`, `sub-b` have no outgoing edges — their federated_search is empty.

---

## 3. What's in each KB

Each KB gets three notes (two public, one private):

**`whoami.md`** — public subgraph (visible to federation):
```markdown
# whoami

Role: <role>
KB: <kb_id>
Neighbours: <list of to-edges>
```

**`federation-index.md`** — hub nodes only, public subgraph. Content = the result of
this hub's `federated_search("#whoami")` (aggregate of children's whoami):
```markdown
# federation-index

Hub role: <role>
<whoami entries gathered via federation, concatenated>
```

**`private.md`** — private subgraph, **never** leaves, not even over a valid edge:
```markdown
---
subgraph: private-internal
---
private-only-<role>
```

An edge grants access to the public subgraph, but `private-internal` is never handed
out by edges — that's what the isolation check rests on (§5).

---

## 4. What each KB sees via federated_search

Target reachability over edges within 3 hops. NB: a single fan-out reaches only
**direct** peers (§7.3); the multi-hop reach below is realized via materialized
`federation-index.md` (built bottom-up), or via targeted `kb_id="a/b/c"`.

| From | Reachable KBs (≤3 hops) | Unreachable |
|------|-------------------------|-------------|
| `owner` | company-a-hub, company-b-hub, dept-a-hub, dept-b-hub, company-site, sub-a, sub-b | boss-a, boss-b |
| `boss-a` | company-a-hub, dept-a-hub, dept-b-hub, sub-a, sub-b | company-b-hub, company-site, boss-b, owner |
| `boss-b` | company-b-hub, company-site, sub-b, sub-a | all of branch A, owner |
| `company-a-hub` | dept-a-hub, dept-b-hub, sub-a, sub-b | everything outside the department |
| `company-b-hub` | company-site, sub-b, sub-a | branch A |
| `dept-a-hub` | sub-a | — |
| `dept-b-hub` | sub-b | — |
| `company-site` | sub-a | — |
| `sub-a` / `sub-b` | ∅ (leaves) | everything |

---

## 5. Example search queries

All calls are MCP `tools/call` to `POST <base>/_system/mcp` (JSON-RPC 2.0). Local
`search` runs under the KB's own auth; for `federated_*` the hub sends the peer a
federation JWT (`Authorization: Bearer <kid-JWT>`, see §2 of the protocol).

### 5.1 Local search of own whoami (any KB)

```json
{ "jsonrpc": "2.0", "id": 1, "method": "tools/call",
  "params": { "name": "search", "arguments": { "query": "whoami Role:" } } }
```
Expect: a snippet from this KB's `whoami.md` (breadcrumb `whoami > …`).

### 5.2 Hub gathers children's whoami (fan-out)

`company-a-hub` calls federated_search without `kb_id` — the query spreads over its
edges (`dept-a-hub`, `dept-b-hub`) and on to the leaves:

```json
{ "jsonrpc": "2.0", "id": 2, "method": "tools/call",
  "params": { "name": "federated_search", "arguments": { "query": "#whoami" } } }
```
Expect: whoami from `dept-a-hub`, `dept-b-hub`, `sub-a`, `sub-b`. This is exactly what
goes into `federation-index.md`.

### 5.3 owner reaches a leaf at the 3rd hop

```json
{ "jsonrpc": "2.0", "id": 3, "method": "tools/call",
  "params": { "name": "federated_search", "arguments": { "query": "Role: sub-a" } } }
```
Expect: `sub-a`'s whoami. NB (§7.3): a single fan-out is 1-level, so `owner` reaches
`sub-a` via the **materialized indexes** of the hubs on the path — uncapped by
`MCP_FEDERATION_MAX_DEPTH`, not by live recursion. The live **targeted** form
`kb_id="company-b-hub/company-site/sub-a"` *does* recurse but is capped: `MAX_DEPTH=N`
allows **N−1** live hops, so that exact 3-hop call at `MAX_DEPTH=3` is rejected (the
e2e asserts both the materialized reach and the cap).

### 5.4 Targeted query to a single peer (`kb_id`)

```json
{ "jsonrpc": "2.0", "id": 4, "method": "tools/call",
  "params": { "name": "federated_search",
              "arguments": { "query": "#federation-index", "kb_id": "dept-a-hub" } } }
```
Expect: only `dept-a-hub`'s results, no fan-out. (`kb_ids: [...]` — to select several
peers.)

### 5.5 Isolation check (the key case)

`dept-a-hub` has an edge to `sub-a` but must **not** see `sub-a`'s private note:

```json
{ "jsonrpc": "2.0", "id": 5, "method": "tools/call",
  "params": { "name": "federated_search",
              "arguments": { "query": "private-only-sub-a" } } }
```
Expect: **empty** (no `Found`). A note in `subgraph: private-internal` is never handed
out by edges. If it leaks — that's an ISOLATION BREACH.

The "parent → child's private note" pairs the e2e checks:

| Searcher | Token | Expected |
|----------|-------|----------|
| owner | `private-only-company-a-hub` | empty |
| boss-a | `private-only-company-a-hub` | empty |
| company-a-hub | `private-only-dept-a-hub` | empty |
| company-b-hub | `private-only-company-site` | empty |
| dept-a-hub | `private-only-sub-a` | empty |
| dept-b-hub | `private-only-sub-b` | empty |
| company-site | `private-only-sub-a` | empty |

### 5.6 Edge negative (no edge → no result)

`boss-a` has no path into company B:

```json
{ "jsonrpc": "2.0", "id": 6, "method": "tools/call",
  "params": { "name": "federated_search", "arguments": { "query": "Role: company-site" } } }
```
Expect: empty — `company-site` is unreachable from `boss-a` (see the §4 table).

---

## 6. How to assemble without agents

In the prototype, agents were needed only to write notes and call MCP. For a pure
trip2g e2e that's unnecessary:

1. **Create 10 KBs** with the `kb_id`s from §2 (each with its own `/_system/mcp`).
2. **Seed 3 notes** into each (§3): `whoami.md`, `private.md`
   (`subgraph: private-internal`), and for hubs also `federation-index.md`. Seed via a
   SQL dump (like [e2e_seed.md](e2e_seed.md)) or GraphQL `updateNotes` — no agent
   involved.
3. **Create 11 federation_secrets** — one per edge in §2 (on the `from` side, the
   secret of peer `to`), scope = the public subgraph, **without** `private-internal`.
4. **Run the §5 checks** with an ordinary HTTP client: local `search` for whoami,
   `federated_search` for the aggregate/depth, and the `private-only-*` tokens for
   isolation.

Pass criterion: each KB finds its own `#whoami`; hubs assemble `#federation-index`; no
`private-only-*` is visible via `federated_search`, not even over a valid edge.

---

## 7. How it runs as an e2e (stand architecture)

This section turns §1–6 from "structure" into an executable case alongside the current
e2e (`docker-compose.test.yml` + Playwright). The facts below are verified against the
code.

### 7.1 One node = one process (not "10 KBs in one instance")

Federation in trip2g is **strictly inter-process, over HTTP**. A KB-note
(`mcp_federation_kb_url` in frontmatter) is a **pointer to a peer's URL**, not a
locally-served KB:

- `fanout` calls `env.FederationClient(ctx, kb.ID)` for each KB-note
  (`internal/case/mcp/federation.go:38`);
- the client always does an HTTP POST to `kb.URL` via fasthttp
  (`internal/federation/client.go`, `cmd/server/main.go` → `FederationClient`);
- there is no in-process path; the JWT `Issuer` = the instance's `PublicURL()`.

So each graph node from §2 = a **separate docker-compose service** with its own
`PUBLIC_URL` and its own `/_system/mcp`. The topology (10 nodes) = **10 app processes**.
The current stand already works this way: `app` + 3 peers = 4 processes.

"One process serves 10 KBs" does **not** apply to federation: an instance has one
`PUBLIC_URL` and one `/_system/mcp`; its KB-notes only list *where* to fan out (over
HTTP), not *what* it hosts locally.

### 7.2 Memory and why we turn vector off

Measured on the current 4-instance stand (`docker stats`, **vector search ON**):

| container | RSS | why |
|-----------|-----|-----|
| `app` (hub) | **~44 MiB** | full `testdata/vault` loaded |
| `app-peer` / `peer2` / `peer3` | **~19 MiB** | tiny seed vault |

The dominant cost is the **bleve index + in-memory notes** (grows with vault size),
**not** embeddings — the model lives in the external `embedding` service (a separate
~5.39 GiB image). So 10 whoami nodes, each with 2–3 tiny notes, should sit at
~18–20 MiB each → **~200 MiB total** + minio. Memory is not the constraint.

Heap profile of the 44 MiB hub (`/debug/pprof/heap` on the internal port, `go tool
pprof -inuse_space`) — **live heap is only ~15 MiB**; the rest of the 44 MiB is the Go
runtime baseline + reclaimable page cache (mmap'd sqlite/bleve/binary). The live heap
breaks down as:

- bleve FTS index (`upsidedown` batch + `gtreap` + `AnalysisWorker`) ≈ 14% — the index
  we keep; scales with note count;
- parsed GraphQL schema (`gqlparser.peekPos`) ≈ 10%, one-time;
- **embeddings (`model.BytesToFloat32Slice`) ≈ 7% (~1 MiB) — the only vector-side cost,
  and it vanishes with vector off**;
- templates (jet/CloudyKit), fasthttp, zerolog, prometheus, mdloader — small fixed
  framework costs.

No leak, no hog. Turning vector off removes only that ~1 MiB embedding slice — per-
instance RSS barely moves. The real reason to disable it is to drop the ~5.39 GiB
image, the ~2.3 GiB model download, and the ~300s startup wait — not memory.

### 7.3 Depth: fan-out is one level; depth via materialized `federation-index.md`

Verified against the code: a fan-out `federated_search` (no `kb_id`) calls each
**direct** peer's local `search` tool and stops — it does **not** recurse
(`internal/case/mcp/federation_handlers.go:34` → `client.Search`; the receiving
`handleSearch` is local-only, `resolve.go:379`). So §1's "recurses deeper" is the
*intent*, not the current behavior.

Multi-hop reachability (the §4 table) is realized two ways:

1. **Materialized `federation-index.md`** — the model this e2e uses. Each hub runs
   its own 1-level fan-out and stores the aggregate as a local note (§3); a parent's
   1-level fan-out then surfaces that child hub's index, so grandchildren ride up
   through the index, not through live recursion. Built **bottom-up** (leaves → root):
   a parent's index must be rebuilt after its children's. This is also the better fit
   for a publishing platform — the index is a curated, git-tracked, reviewable artifact
   (the hub owner decides what their federated face exposes), queries stay cheap, and a
   down peer doesn't break a query.
2. **Targeted `kb_id="company-b-hub/company-site/sub-a"`** — recurses, stripping one
   segment per hop (`federation_handlers.go:54`, depth-capped by
   `MCP_FEDERATION_MAX_DEPTH`). Use it to drill into a specific deep node on demand.

So the e2e: seed notes + wire edges → **materialize indexes bottom-up** → assert §4/§5.

---

## 8. Config: disable vector search (env-only, no code)

Verified against the code — it disables cleanly:

- `FEATURES='{"vector_search":{"enabled":false}}'` (or no `FEATURES` at all):
  `internal/features/features.go` doesn't panic; `cmd/server/main.go:428` doesn't
  create the OpenAI client (`openaiClient` stays nil).
- Search falls back to bleve FTS: `internal/case/sitesearch/resolve.go` and
  `internal/case/mcp/resolve.go` go to FTS first; the vector path is gated behind
  `Enabled && OpenAI() != nil`. All §5 queries are textual → FTS covers them.
- The §5.5 isolation doesn't depend on vectors: it rests on federation permissions
  (`internal/case/canreadnote/resolve_with_subgraphs.go`), not on ranking.

**Mandatory:** remove `depends_on: embedding (condition: service_healthy)` from every
node. Otherwise startup still waits for the embedding service's health (~300s, ~2.3 GiB
model) even though the feature is off. The `embedding` service itself isn't needed for
this set — exclude it from the whoami stack.

---

## 9. The stand in docker-compose: a profile, not "always up"

10 nodes on top of the current 4 means 14 app processes on every `docker compose up`.
To avoid bloating the default run:

- put the whoami nodes behind `profiles: [whoami]` (or a separate overlay file) and
  bring them up only for this suite: `docker compose --profile whoami up`;
- minio is reused — its own `MINIO_PREFIX` per node (as the peers do now).

**`PUBLIC_URL` = the internal docker name, not `localhost`.** In a mesh everyone calls
everyone, so `PUBLIC_URL=http://whoami-owner:PORT` (like `app-peer` today), not
`http://localhost:…`. A host-published port is needed **only** by the runner (seeding
§11 + the §5 checks); federation travels the docker network by service name. The
current `app` uses `localhost` because it's always only the *caller*; for a full mesh
that won't do.

Single-node template (modeled on `app-peer` in `docker-compose.test.yml`):

```yaml
  whoami-dept-a-hub:
    build: { context: ., dockerfile: Dockerfile }
    profiles: [whoami]
    depends_on:
      minio: { condition: service_healthy }   # no embedding!
    ports: ["30051:30051", "30052:30052"]      # host port for the runner only
    environment:
      - LISTEN_ADDR=0.0.0.0:30051
      - INTERNAL_LISTEN_ADDR=:30052
      - DB_FILE=/data/whoami-dept-a-hub.sqlite3
      - DEV=true
      - OWNER_EMAIL=hello@example.com
      - PUBLIC_URL=http://whoami-dept-a-hub:30051   # internal service name
      - FEATURES={"vector_search":{"enabled":false}}
      - MINIO_ENDPOINT=minio:29000
      - MINIO_PREFIX=whoami-dept-a-hub/
      - JWT_SECRET=whoami-dept-a-hub-secret
      - USER_TOKEN_COOKIE_NAME=trip2g_whoami_dept_a_hub
      - GIT_API_REPO_PATH=/data/git-whoami-dept-a-hub
      - MCP_FEDERATION_MAX_DEPTH=3
      - MCP_FEDERATED_GRAPHQL=true
    volumes: ["./tmp/data:/data"]
    healthcheck:
      test: ["CMD","wget","-q","--spider","http://localhost:30052/healthz"]
      interval: 5s
      retries: 10
```

---

## 10. Declarative topology (single source for compose+seed+assert)

11 edges across 10 nodes are easy to desync by hand. Keep **one data structure** that
generates the compose services, the KB-notes/secrets, and the assertions. Nodes (ports
are illustrative):

| node | service | PUBLIC_URL (internal) | host port | hub? |
|------|---------|-----------------------|-----------|------|
| owner | whoami-owner | http://whoami-owner:30011 | 30011 | — |
| boss-a | whoami-boss-a | http://whoami-boss-a:30021 | 30021 | — |
| boss-b | whoami-boss-b | http://whoami-boss-b:30031 | 30031 | — |
| company-a-hub | whoami-company-a-hub | http://whoami-company-a-hub:30041 | 30041 | hub |
| dept-a-hub | whoami-dept-a-hub | http://whoami-dept-a-hub:30051 | 30051 | hub |
| dept-b-hub | whoami-dept-b-hub | http://whoami-dept-b-hub:30061 | 30061 | hub |
| company-b-hub | whoami-company-b-hub | http://whoami-company-b-hub:30071 | 30071 | hub |
| company-site | whoami-company-site | http://whoami-company-site:30081 | 30081 | hub |
| sub-a | whoami-sub-a | http://whoami-sub-a:30091 | 30091 | — |
| sub-b | whoami-sub-b | http://whoami-sub-b:30101 | 30101 | — |

Edges — an array of `{from, to}` from §2 (11 of them). The same array drives §11.

---

## 11. Seeding notes and edges (as in the current stand)

**Notes** — like [e2e_seed.md](e2e_seed.md): a base SQL dump + pushing per-node vaults
via the `obsidian-sync` CLI (key via `requestEmailSignInCode → signInByEmail →
createApiKey`). Into each node: `whoami.md` (public subgraph), `private.md`
(`subgraph: private-internal`), and for hubs also `federation-index.md`. Don't copy 10
blocks by hand — parametrize `pushSeedvault({port, cookie, folder})` (the pain is
already noted in [../../scripts/e2e-rewrite-proposal.md](../../scripts/e2e-rewrite-proposal.md)).

**Edges** — modeled on `e2e/federation-acl.spec.js` (`createInbound/Outbound`), for
each edge `from → to`:

- on `to`: `createInboundFederationSecret`, scope = the public subgraph, **without**
  `private-internal`;
- on `from`: a KB-note with `mcp_federation_kb_url = <to PUBLIC_URL>/_system/mcp` and
  `mcp_federation_kb_id = <to>`, plus `createOutboundFederationSecret` (the same `kid`
  + 64-hex secret).

It's precisely the "inbound scope without `private-internal`" that makes the §5.5 check
live: the edge exists, but the private subgraph isn't in the scope → the private note
doesn't go out.

---

## 12. What's actually implemented (the test checks prod, not a dream)

The §5.5 subgraph isolation is working code, not a stub (verified):

- `internal/case/mcp/federation_helpers.go` — `ListFederationSecretSubgraphsByKID`
  (secret scope → allowed subgraphs);
- `internal/case/mcp/resolve.go` — allowed subgraphs are put into the context →
  `canReadMCPNote`;
- `internal/case/canreadnote/resolve_with_subgraphs.go` — a note is hidden if its
  subgraph isn't in the allow-list; there's a unit test `resolve_with_subgraphs_test.go`.

So the e2e validates **production behavior**, not aspirational code. That's the main
reason to build the stand.

---

## 13. Risks / what to verify before coding

1. **RSS — measured** (§7.2): ~19 MiB per small-vault node with vector ON, ~44 MiB for
   the full-vault hub. Re-check once the whoami vaults are seeded, but it's comfortably
   under budget.
2. **Startup serializes** across healthchecks (10× `start_period`); estimate the total
   `up` time and relax the healthcheck if needed. Each node also opens 3 SQLite
   connections and starts queues regardless of Telegram.
3. **Disk:** 10× SQLite + 10× git repos (`GIT_API_REPO_PATH` per node) — use tmpfs for
   `/data` on CI to avoid hitting I/O.
4. **Don't forget** to remove `depends_on: embedding` from **every** node — otherwise a
   silent 300s wait for a model we don't use.
5. **11 edges = 11 inbound + 11 outbound + 11 KB-notes** — keep them in one data
   structure (§10), otherwise desync is inevitable.

---

## 14. Implementation

Built and green — **40 Playwright assertions** covering all of §5. Files:

- `e2e/whoami/topology.json` — the single source (§10): 10 nodes + 11 edges.
- `scripts/e2e/whoami-mesh.mjs` — json-driven linker. Subcommands: `gen` (writes the
  compose), `up`, `seed`, `wire`, `materialize`, `down`, `all`. Plain Node + `fetch`, no
  new deps (same style as `scripts/bench-pushnotes.mjs`).
- `e2e/whoami-federation.spec.js` — the §5 assertions; self-skips if the mesh is down, so
  a normal `npx playwright test` run ignores it.
- `docker-compose.whoami.yml` — **generated** (gitignored) from the topology; an isolated
  compose project `trip2g_whoami_env` with its own minio, vector search off, no embedding.

Run:

```
npm run test:e2e:whoami     # up + seed + wire + materialize + spec (--workers=1)
npm run whoami:down         # tear down (-v)
```

What the spec asserts: §5.1 local whoami (×10) · §5.2 fan-out reach = transitive
descendants (×5 origins) · §5.3 targeted 2-hop **and** the depth-cap rejection · §5.4
targeted single · §5.5 isolation on all 11 edges **+ a local-visibility control** (the
private note exists locally, so an empty federated result is real filtering) · §5.6
negative.

Findings confirmed while building:

- Per-node RSS **~18 MiB** (vector off); whole mesh **~300 MiB** incl. minio.
- Fan-out is **1-level** (§7.3); depth comes from materialized indexes, which are
  **uncapped**. Indexes are stored as a compact one-line marker list so a parent's
  1-level search *snippet* carries the full reach.
- The live **targeted** path is capped **off-by-one**: `MCP_FEDERATION_MAX_DEPTH=N`
  allows **N−1** live hops (rejects when incoming depth ≥ N). The e2e asserts this.
- Playwright runs this file with **`--workers=1`**: parallel tests racing the same node's
  dev sign-in code contend (the project's `e2e/helpers/auth.js` caches the admin JWT for
  the same reason).
