# fleet GraphQL API: roles, agent-interaction graph, and the debugger surface

**Status:** Open design (not yet built, 2026-07-14). Sibling to `codellm_extraction.md`, `agents.md`, `hat.md`.
**Builds on:** the fleet agent runtime (`internal/fleet`, `cmd/fleet`), the monolith's GraphQL + session auth (`internal/graph`, `internal/usertoken`, `cmd/server/auth.go`), and the existing SSE/Caddy proxy pattern.

## TL;DR

fleet grows its **own** GraphQL endpoint, separate from the monolith's, serving what only fleet understands: the parsed roles, the **agent-interaction graph** (who-triggers-whom, derived from trigger/read/write patterns), and a **stateful debugger** for stepping a role's code block-by-block. The monolith stays a dumb note store and knows nothing about roles — the boundary holds because fleet owns its own API. Browser tools reach it same-origin via a Caddy `/_fleet/*` reverse-proxy that just forwards the request (and the session cookie). **Auth is delegated per-request:** fleet takes the incoming browser cookie and asks its own trip2g `query { viewer { role } }`; `admin` → allowed, anything else → 401, monolith-unreachable → fail-closed. This gives full parity with the monolith's admin check (including the ban validator) while fleet holds no secret and verifies nothing itself. Two trip2g **templates** (graph visualizer, debugger) consume this API — the same "app = layout + GraphQL" pattern as kanban_template / theme_editor. **codellm is the one deliberate exception to "everything GraphQL":** it stays an OpenAI-REST server-to-server endpoint (see `codellm_extraction.md`), because its whole value is impersonating an LLM.

## fleet identity & mutual exclusion: hash-in-webhook-url (no migration)

One fleet serves one LLM endpoint/provider; two providers = two fleet processes. A role picks its fleet with `fleet_id: X` in frontmatter; each fleet processes only roles whose `fleet_id` is its own (a partition key over the role set). codellm needs no special case: it is just a fleet whose `--llm-base-url` points at the codellm service (the earlier `--codellm-base-url` cutover flag is dropped).

**Claiming a `fleet_id` reuses the existing webhook registration — no lease table, no new column, no migration.** fleet already registers `change_webhooks` for its roles (the reconcile loop). The trick: derive the webhook URL deterministically from the fleet_id.

- Convention: a fleet's delivery endpoint is `/_fleet/<h>/webhook`, where `h = sha256("fleet:" + fleet_id)`.
- `h` is a stable, namespaced marker (the `fleet:` prefix namespaces it against other sha256 uses and keeps the plaintext fleet_id out of the URL). It is **not a secret** — fleet_id is not sensitive; `h` is just an opaque stable id, never used for auth.
- The marker lives in the **existing `url` column** (`change_webhooks.url`), so there is nothing to migrate.

**Mutual exclusion / takeover = match by the `/<h>/` segment.** On reconcile, a fleet finds the webhooks whose URL path contains `/<h>/`, points them at its own current address, and drops any others sharing that `/<h>/`. Whoever registers last owns the URL the monolith delivers to → **last-writer-wins**. A duplicate fleet with the same `fleet_id` computes the same `h`, targets the same webhook, and "takes over the webhook_url and that's it." Dedup is by the **`/<h>/` segment**, not the full URL — this covers both topologies: one box behind Caddy (URL is the same relative path anyway) and a fleet on another machine (full URL differs by host, but `/<h>/` is identical).

**Crash recovery is free:** a restarted fleet recomputes `h` and re-claims the webhook (updates the URL to its new address). No lease, no TTL, no heartbeat, no `flock` (which is local-only and breaks across machines). `created_at` / `updated_at` on the row give free observability of when a `fleet_id` was last (re)claimed — surface it in the graph viz.

Cost: mutual exclusion is application-enforced (reconcile), not a DB unique constraint, so a concurrent double-claim has a bounded flip window where both briefly look owned. Since the monolith delivers to whatever URL is currently stored, only one fleet receives at a time; the residual risk is one in-flight overlapping run — acceptable exactly to the degree roles are idempotent (find/replace, content-hash dedup survive it; append / external POST do not). If a hard atomic guarantee is ever needed, that is the migration (a `fleet_id` column + unique index) — deferred until a non-idempotent role demands it.

### Auth split on the `/_fleet/*` prefix

Two different callers share the prefix and use **different** auth — the fleet mux routes by path:
- `/_fleet/graphql` (and other browser tools) — **browser → fleet**, gated by the delegated-admin cookie check (below).
- `/_fleet/<h>/webhook` — **monolith → fleet**, server-to-server change delivery, authenticated by the existing webhook `secret` / HMAC (`shared_webhooks.md`), NOT the cookie. The delegated-admin middleware must NOT wrap the webhook path.

## Why fleet gets its own API (not a monolith resolver, not a note)

Three rejected alternatives and why:

- **A `fleetRoles` resolver in `cmd/server`.** Rejected: the monolith must not know about fleet agents. A resolver would make it import `fleet.ParseRole` and understand role schema — a domain dependency pointing the wrong way (today fleet consumes the monolith's GraphQL, not vice versa).
- **fleet writes the graph as a note the template renders.** Works for a static graph but is a poor fit for the **interactive** debugger (stateful stepping, inspecting the pipe between blocks), and a materialized note goes stale versus an on-demand resolver.
- **The template parses role notes client-side.** Reimplements `ParseRole` (trigger defaults, exclude semantics, validation) in JS, which drifts from the Go source of truth.

fleet already parses every role (`Discovery` / `DiscoverParsed`, `internal/fleet/discovery.go`) and holds the debug primitives (`internal/fleet/debug.go`). Exposing that as GraphQL keeps the knowledge where it lives and enforces the boundary: the monolith serves notes, fleet serves fleet.

## Auth: delegated per-request, cookie pass-through

The load-bearing decision. The monolith's admin session cookie is a stateless HS256 JWT (`internal/usertoken/token.go:100` `parseClaims` verifies signature with the shared secret; `Role` is a claim) **plus** one DB validator — a ban check (`cmd/server/auth.go:26`, `UserBanByUserID`). fleet could verify the JWT locally (it has the secret), but that skips the ban check, leaving a banned-admin-within-token-TTL gap. So instead:

1. Browser tool calls same-origin `/_fleet/graphql`; the Secure session cookie rides automatically.
2. Caddy forwards the request **and the cookie** to fleet (dumb pass-through; no auth in Caddy).
3. fleet, per request, forwards **the end-user's cookie** to its own trip2g `POST /_system/graphql` with `query { viewer { role } }`.
4. `role == "admin"` → serve; else → 401; monolith unreachable/slow → **fail-closed** (reject).

Properties:
- **Full parity** with the monolith admin check — the ban validator and every other validator run in the monolith, not a fleet copy. No TTL ban gap.
- **fleet holds no secret for this** and verifies nothing itself — it asks the authority. Minimal trust in fleet.
- **Critical nuance:** fleet forwards the *end-user's* cookie, NOT its own admin-HAT lane (the one it uses for reconcile/discovery, `internal/fleet/hatauth.go`). Using fleet's own creds would answer "is fleet admin," not "is the caller admin."
- Because every request is authenticated, fleet being reachable is less critical than in a trust-the-channel model — but defense-in-depth still favors binding fleet to a private interface with Caddy as the sole ingress.
- Debug traffic is low and interactive, so the per-request round-trip is a negligible tax. A few-seconds cache of (cookie→role) is possible but unnecessary initially.

## Endpoint discovery

- **Default (fleet + trip2g on one box):** same-origin `/_fleet/graphql`, reverse-proxied by Caddy to fleet's private port — mirrors the existing SSE proxy (`:9081 → :8081`). The template uses a fixed path and does `$trip2g_graphql_request` against it; no config, no CORS.
- **Remote fleet (federated / off-box):** an override — a `fleet_endpoint:` in the template frontmatter or a single admin-config setting — with CORS + the pass-through auth above. Escape hatch, not the default.

## Schema (sketch)

```graphql
type Role {
  name: String!
  path: String!            # role note path under agents-folder
  executor: String!        # "llm" | "code"
  model: String
  triggerOn: [String!]!    # create | update | remove
  triggerInclude: [String!]!
  triggerExclude: [String!]!
  readPatterns: [String!]!
  writePatterns: [String!]!
}

type GraphNode { role: String! inboxGlob: [String!]! outboxGlob: [String!]! orphan: Boolean! }
enum EdgeKind { TRIGGER DATAFLOW }
type GraphEdge { from: String! to: String! kind: EdgeKind! exact: Boolean! cutByDepth: Boolean! }
type RoleGraph { nodes: [GraphNode!]! edges: [GraphEdge!]! cycles: [[String!]!]! }

type Query {
  roles: [Role!]!
  roleGraph: RoleGraph!
}
```

The `roleGraph` resolver reuses `fleet.ParseRole` (already in fleet) — no duplication. `ParseRole` may later move to a neutral `internal/fleetrole` package if any other binary needs it, but not `cmd/server`.

### Graph derivation (feeds the visualizer template)

- **Node** = role. `inboxGlob` = `triggerInclude ∖ triggerExclude`; `outboxGlob` = `writePatterns`.
- **TRIGGER edge** A→B when `A.writePatterns ∩ B.triggerInclude ≠ ∅` and `B.triggerExclude` doesn't cover the intersection — "A writes a note that wakes B" (the causal graph).
- **DATAFLOW edge** A→B when `A.writePatterns ∩ B.readPatterns ≠ ∅` — "A produces data B reads but isn't woken by."
- **Glob intersection** is the one hard bit. v1 approximates: expand `A.writePatterns` to representative concrete paths and test them against `B.triggerInclude`; mark the edge `exact: false` when approximated. False positives beat false negatives (better to show a spurious edge than hide a loop).
- **Cycles** = SCCs over TRIGGER edges → runaway-loop feedback made visible before running. `cutByDepth` annotates an edge/cycle that the webhook `max_depth` guard would sever.
- **Orphan** = no inbound TRIGGER edge (only manual/cron) and/or no outbound (writes wake nobody).

## Debugger uses codellm's standard OpenAI path — no bespoke debug protocol

The debugger steps a role's markdown block-by-block and inspects the pipe **between** blocks (e.g. to write/validate a payload mid-pipeline). After `codellm_extraction.md`, the block/pipe execution lives in **codellm**, and codellm keeps **one** protocol: OpenAI REST. The debugger does **not** get a separate GraphQL schema — it calls the **standard path** `/_codellm/v1/chat/completions`, the same endpoint fleet uses, so any OpenAI client (the browser template, curl, an SDK) drives it identically.

**Stepping is stateless re-execution, which is exactly right for deterministic code.** codellm holds no session. To step, the debugger client sends blocks `1..i` and codellm re-runs from the top (deterministic → re-running is correct, not a workaround). To see the pipe *between* blocks and edit it, the debugger asks for debug info via an **opt-in extension** on the request; codellm returns the standard completion **plus** a non-standard field a real LLM would never send:

```jsonc
// request: standard chat/completions + an extension flag
{ "model": "codellm", "messages": [...], "x_fleet_debug": true }
// response: standard OpenAI shape + an ignored-by-normal-clients extension
{ "choices": [...],
  "x_fleet_debug": { "blocks": [ {"index":0,"stdout":"...","pipeBuffer":"..."}, ... ] } }
```

A normal OpenAI client ignores `x_fleet_debug`; the debugger reads it. To edit an intermediate buffer and continue, the client re-sends `1..i` with the edited buffer injected — re-execution again. This keeps codellm a pure OpenAI endpoint (interchangeable with a real LLM, which simply ignores the debug flag) while giving the debugger everything it needs. The block/pipe primitives must stay addressable for this; `pipelineDebugTaps` (`runcode.go:159`) is currently **dead code** (no caller passes a non-nil tap) and must gain a real test as it moves into codellm — that tap is how per-block `pipeBuffer` is captured.

**codellm also exposes a small GraphQL for markdown *structure* (editor support), separate from execution.** The debugger UI wants a `text_area` per block; to get blocks whose boundaries match execution exactly, the parsing must be codellm's own — the same `ExtractFencedBlocks` it runs with, not a JS reimplementation (else the editor's blocks diverge from what runs, and the debugger lies). So:

```graphql
enum BlockKind { CODE PROSE }
type MdBlock { index: Int! kind: BlockKind! language: String content: String! }
type Query    { parseMarkdown(md: String!): [MdBlock!]! }        # split into ordered blocks
type Mutation { assembleMarkdown(blocks: [MdBlockInput!]!): String! }  # round-trip back to md
```

Round-trip fidelity is a server guarantee: `assembleMarkdown(parseMarkdown(md))` reproduces `md` (code blocks re-fenced with their language, prose preserved in order). Debugger flow: `parseMarkdown` → one text_area per block → user edits → `assembleMarkdown` → run via `/_codellm/v1/chat/completions` with `x_fleet_debug`. This is the **only** thing on codellm's GraphQL — structure, not execution or stepping; execution stays REST.

Same-origin reach: Caddy proxies `/_codellm/*` to codellm (cookie forwarded); codellm gates **both** `/v1/*` and `/graphql` with the **same delegated admin check** as fleet (below). So codellm is reachable three ways total: fleet's locked server-to-server `/v1` lane, the browser's cookie-gated `/_codellm/v1/*` (execution + `x_fleet_debug`), and the browser's cookie-gated `/_codellm/graphql` (parse/assemble). fleet's own GraphQL keeps only **roles + roleGraph**.

(Historical note: earlier drafts put a stateful *debug* GraphQL API on fleet, then on codellm — both dropped. Execution/stepping is the standard OpenAI path + `x_fleet_debug`; codellm's GraphQL is scoped to markdown structure only.)

```graphql
type Mutation {
  startDebug(rolePath: String!, targetPath: String): DebugSession!
  stepBlock(session: ID!, editedStdin: String): DebugStep!   # run block i, optionally with an edited inter-block buffer as stdin
}
type Subscription { debugState(session: ID!): DebugStep! }
type DebugStep { index: Int! stdout: String! pipeBuffer: String! done: Boolean! }
```

The debugger drives codellm's per-block execution and surfaces the captured inter-block buffer for inspection and edit before feeding it as the next block's stdin. This is why the extraction must keep the block/pipe model addressable (a seam), not fuse the pipeline into one opaque run.

## The two tools live in the admin $mol UI (`assets/ui`), not standalone templates

These are **admin operational tooling**, so they are built as $mol components under `assets/ui/` (the admin frontend: `.view.tree` structure + `.view.ts` behavior, GraphQL via `$trip2g_graphql_request`, typed CSS, localization), **not** as self-contained vault layouts like kanban_template / theme_editor. Rationale:

- The admin $mol app is already served same-origin by the monolith with the admin **session cookie present**, so a component calling `/_fleet/graphql` or `/_codellm/v1/chat/completions` is same-origin with the cookie riding automatically — no frontend auth work. The delegated admin check still runs server-side on fleet/codellm.
- Reactive components, `$mol_wire_sync` for async→sync, existing GraphQL plumbing and admin nav — the debugger and graph want all of it.
- These are not shareable content extensions (the template pattern's audience); they are internal tools, and the admin app is their home.

Both are **admin app views wired into the admin like the existing `editor` component** — no layout wrappers, no `layout:`-notes, no `docs/_layouts` files. They are reached through admin navigation/routing, same as the file editor and the form-submits catalog. The admin $mol app is already same-origin with the session cookie, so their GraphQL/HTTP calls just work.

| Tool | $mol source | Reached via |
|------|-------------|-------------|
| Graph visualizer | `assets/ui/fleet/viz/` (`.view.tree` + `.view.ts`) | admin nav (editor-style) |
| Debugger | `assets/ui/codellm/debugger/` | admin nav **and directly from the file editor** |

The debugger also opens **inside the existing editor**: when editing a role note (markdown with code blocks), a "Debug" action steps the blocks in place — the editor already holds the text, so it only needs codellm's block boundaries (`parseMarkdown`) to align its per-block run with real execution, plus the `x_fleet_debug` output. Authoring and debugging the role happen in the same surface. The standalone `assets/ui/codellm/debugger` view and the editor panel share the component.

- **`assets/ui/fleet/viz`** — renders `roleGraph` from fleet-GraphQL (nodes + TRIGGER/DATAFLOW edges, cycles highlighted, orphans flagged, `cutByDepth` styling). Answers "if I touch `blog/**`, which wave of agents wakes" (BFS over TRIGGER edges).
- **`assets/ui/codellm/debugger`** — drives codellm's standard `/_codellm/v1/chat/completions` with `x_fleet_debug`; shows each block's stdout and the editable inter-block pipe buffer; steps by re-execution.

Built by the existing admin `npm run build`; committed in-repo. First-party admin tooling, not a distributed extension — a shareable template version could still come later if wanted, since the fleet/codellm APIs are the same either way.

Dogfood loop: **building these with fleet agents simultaneously tests the fleet agent runtime end-to-end** — the tooling and its test are the same act.

## codellm is REST, on purpose (not a contradiction)

"GraphQL as the default inter-service contract" has exactly one deliberate exception: **fleet ↔ codellm is OpenAI REST** (`/v1/chat/completions`), because codellm impersonates an LLM so the same role can point at a real model — making it GraphQL would kill interchangeability and reintroduce a code-specific path in fleet. That channel is server-to-server (not browser-facing), so it needs no Caddy `/_fleet/*` proxy and no cookie auth — it needs its own locked channel (mTLS / shared token / loopback) per `codellm_extraction.md`. Division of labor: **codellm executes (REST, deterministic); fleet-GraphQL orchestrates and inspects (roles, graph, debug).**

## Caddy additions

One new route, mirroring the SSE proxy:

```
handle /_fleet/* {
    reverse_proxy <fleet-private-addr>   # dumb pass-through; forwards the session cookie; NO auth here
}
```

Auth is inside fleet (delegated `viewer{role}`), not Caddy. codellm gets NO Caddy route (server-to-server only).

## Open questions

- Does `fleet.ParseRole` move to a neutral `internal/fleetrole` package now, or stay in `internal/fleet` until a second consumer appears? (Lean: stay until needed.)
- fleet as a long-running GraphQL server: it currently runs as a worker + loopback `/debug`; standing up gqlgen (or a hand-rolled handler) inside `cmd/fleet` is real scope — decide gqlgen vs minimal.
- The `viewer{role}` delegated check on every request: acceptable for debug/graph traffic; revisit if fleet's GraphQL ever serves high-frequency reads.
