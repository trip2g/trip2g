# Federation Agent Setup — raw technical reference

**Purpose:** raw technical reference for an agent that configures knowledge sharing between trip2g instances **on its own** (private-by-default, a shared `shared` pool, hubs, oversight). A separate pipeline turns this into a user-facing guide for agents; here are all the facts, exact signatures, and edge cases, unpolished.

**Status:** v0, 2026-06-17. Verified against trip2g code:
- note access: `internal/case/canreadnote/resolve_with_subgraphs.go`
- federation: `internal/case/mcp/federation_handlers.go`, `internal/case/mcp/federation_helpers.go`, `internal/federation/`
- frontmatter patches: `internal/frontmatterpatch/evaluate.go`
- MCP admin tools: `internal/case/mcp/resolve.go`

**User docs (trip2g.com/docs) referenced:**
- [federation](https://trip2g.com/docs/en/user/federation) — federation model end-to-end
- [advanced](https://trip2g.com/docs/en/user/advanced) — subgraphs and access control
- [frontmatter-patches](https://trip2g.com/docs/en/user/frontmatter_patches) — frontmatter patches
- [hub create](https://trip2g.com/docs/en/hub/_create) — adding a base to a hub (KB-note)
- [mcp](https://trip2g.com/docs/en/user/mcp) — local MCP server and tools
- [selfhosted](https://trip2g.com/docs/en/user/selfhosted) — environment variables

---

## 1. Control plane: how the agent issues admin operations

Every operation below is **admin GraphQL** against a specific instance. There are two ways to invoke it; the rest of the document is path-agnostic.

### 1.1 Via simplepanel (recommended within a pool)

The panel is the trusted control plane: it knows the pool master secret and authenticates as admin into any pool instance over a cookie (HAT → session cookie). The agent works by `instance_id` and never handles per-instance secrets.

Panel MCP tools (authorized by the panel API key):

| Tool | Args | What it does |
|---|---|---|
| `list_slots` | — | pool instances (slots): `{instance_id, name, domain, ...}`. `instance_id` = slot id |
| `instance_federation` | `instance_id` | instance topology (see §7): self, subgraphs, secrets, KB-notes |
| `instance_graphql_request` | `instance_id`, `query`, `variables?` | run admin GraphQL on the instance; the panel proxies it as cookie admin |

Proxy mechanics: the panel signs a HAT JWT (`{e: adminEmail, ae:true}`) with the instance key, POSTs `/_system/hat` → session cookie → uses that cookie against `/_system/graphql`. Authorization on the instance is the usual `checkAdmin` over the session cookie (admin bypass). The implementation reuses the client in `internal/federation/client.go` (`GetTopology`, `GraphQL`).

### 1.2 Directly via trip2g MCP (`graphql_request`)

If someone operates an instance separately (outside the panel pool), trip2g itself exposes admin tools over MCP. This requires an instance API key with `enable_mcp_admin_tools=true` (mutation `setApiKeyMcpAdminTools`). Then `/_system/mcp` offers (see `internal/case/mcp/resolve.go`):

- `graphql_introspection(pattern)` — find operations/types by regexp before calling;
- `graphql_request(query, variables?)` — run a query/mutation as admin.

Throughout the document, "call admin GraphQL X" means either `instance_graphql_request(instance_id, X)` (panel) or `graphql_request(X)` (native).

> **Deployment-dependent.** Path 1.1 assumes a deployed simplepanel with access to the pool master secret. Path 1.2 assumes MCP admin tools are enabled on the key. With neither, configuration is done by hand in the admin UI.

---

## 2. Privacy model

The unit is an **agent with its own knowledge base** (a trip2g instance). Tiers:

- **`private`** — the default. A note with no subgraph should land in `private` (via the patch in §5.1) and be visible only to the agent/admin itself.
- **`shared`** — the common pool for colleagues. A note with `subgraph: shared` is deliberately shared. This is also the default scope of federation links.
- **oversight** — selected readers (management/security) read **everything, including `private`**, through links whose scope includes `private`.

Default topology (low ceremony):
- agents are connected through **hub(s)** over `shared` (instead of an N² direct mesh);
- one or a few **oversight** readers with scope = all subgraphs.

There can be several hubs (e.g. per department); hubs may federate to each other (subject to the depth limit, §8).

---

## 3. Access semantics — MUST read

When a federated peer (with a `kid`) reads a note, trip2g checks (`canreadnote/resolve_with_subgraphs.go`, `allowed` = the subgraphs in that `kid`'s scope):

```
1. note free:true            → ALWAYS readable (public)
2. len(allowed) == 0         → NOTHING (except free). Empty scope = NO access
3. note with no subgraphs    → readable by ANY peer with a non-empty scope   ← IMPORTANT
4. note subgraph ∈ allowed   → readable
5. otherwise                 → not readable
```

(There is also a nuance in the code: if any of the note's subgraphs is flagged `requireSignin`, a signed-in reader is allowed — rarely relevant to federation setup, listed here for completeness.)

**Two consequences that drive the whole setup:**

- **(rule 3) A note with no subgraph is visible to any peer with any non-empty scope.** So "private by default" holds ONLY when the default-private patch (§5.1) is installed. Without it, granting someone `shared` also exposes every untagged note. **The default-private patch is load-bearing, not optional.**
- **(rule 2) Empty scope = no access, NOT "full access".** "Read everything" is expressed by enumerating subgraphs in the scope (including `private`); there is no wildcard.

`free:true` is a separate axis: it makes a note public regardless of subgraphs. For a private internal hub, don't use `free` on knowledge notes (only, optionally, on the hub's KB-notes themselves — see §5.5).

---

## 4. Exact GraphQL signatures (reference)

All admin (cookie bypass via §1). Source: `internal/graph/schema.graphqls`.

**Federation secrets:**
```graphql
# inbound: "someone may read me". kid is REQUIRED; secretHex optional (empty → server generates).
createInboundFederationSecret(input: { kid: String!, description: String, secretHex: String })
  -> { id, kid, secretHex }     # secretHex shown ONCE

# outbound: "I may read a peer".
createOutboundFederationSecret(input: { kid: String!, secretHex: String!, kbURL: String!, description: String })
  -> { id }

# inbound secret scope: which subgraphs the peer sees under this kid (allow-list).
addFederationSecretSubgraph(input: { kid: String!, subgraphID: Int64! }) -> { success }
removeFederationSecretSubgraph(input: { kid: String!, subgraphID: Int64! }) -> { success }

revokeFederationSecret(id: Int64!) -> { revokedId }   # select __typename if you don't need revokedId
```

**Subgraphs (read):**
```graphql
allSubgraphs -> AdminSubgraphsConnection { nodes { id, name, color, hidden, requireSignin } }
```
A subgraph is **auto-created** when a note carrying `subgraph: <name>` reaches the instance (there is no separate create mutation). See §5.2.

**Notes (write) — creates/updates notes, including arbitrary frontmatter:**
```graphql
updateNotes(input: { changes: [ { upsert: { path: String!, content: String! } } ] }) -> { ... }
# admin via cookie passes (checkapikey admin-bypass). X-Api-Key not needed under cookie admin.
```

**Frontmatter patches:**
```graphql
allFrontmatterPatches -> { nodes { id, includePatterns, excludePatterns, jsonnet, priority, description, enabled } }
createFrontmatterPatch(input: {
  includePatterns: [String!]!, excludePatterns: [String!], jsonnet: String!,
  priority: Int!, description: String!, enabled: Boolean!
}) -> { frontmatterPatch { id } }
updateFrontmatterPatch(input: {...}) ; deleteFrontmatterPatch(input: { id })
```

**Federation (read your own secrets):**
```graphql
federationSecrets -> [ { id, kid, kbUrl, description, createdAt, revokedAt, subgraphCount } ]
```
(`kbUrl != null` → outbound; `kbUrl == null` → inbound.)

**Aggregated (if the panel endpoint exists):** `GET /_system/federation/admin` returns self + subgraphs + inbound/outbound with scope + KB-notes in one call (tool `instance_federation`).

---

## 5. Operation recipes

Each recipe = what to call + verification. All calls go through §1.

### 5.1 Default-private (load-bearing; do it FIRST on every instance)

Installs a patch: a note with no subgraph gets `subgraph: private`. Files on disk are not modified (the patch is virtual, applied at load).

```graphql
createFrontmatterPatch(input: {
  includePatterns: ["**/*.md"],
  excludePatterns: [],
  priority: -100,                 # applied first
  enabled: true,
  description: "agent:default-private",
  jsonnet: "if std.objectHas(meta, \"subgraph\") || std.objectHas(meta, \"subgraphs\") then {} else { subgraph: \"private\" }"
})
```

Mechanics (`frontmatterpatch/evaluate.go`): `meta` is passed to jsonnet as an object (ExtVar JSON); the result is **shallow-merged over** the current meta. Returning `{}` changes nothing → notes with an explicit subgraph are untouched; notes without one get `private`.

**Idempotency:** before creating, check `allFrontmatterPatches` for `description == "agent:default-private"`. If present, don't duplicate.

**Verification:** `allFrontmatterPatches` contains the patch; any untagged note now reports `subgraph: private`.

> **Caveat (deployment-dependent).** If the instance also serves a public site, public notes must have `free: true` (it overrides privacy) or add their paths to `excludePatterns`. Irrelevant for a purely internal instance.

### 5.2 Create the `shared` subgraph (+ sharing policy in its note)

A subgraph materializes by writing a note with `subgraph: shared`. The note body doubles as the human-readable policy ("what goes here").

```graphql
updateNotes(input: { changes: [ { upsert: {
  path: "subgraphs/shared.md",
  content: "---\nsubgraph: shared\n---\nPolicy: put here knowledge that colleagues' agents should find. Everything else stays private by default."
} } ] })
```

**Verification:** `allSubgraphs` contains `shared`.

The `private` subgraph already exists after the first private note; a policy note for it is optional (`subgraphs/private.md` if desired).

### 5.3 Share a note

The author/agent adds `subgraph: shared` to the note (via `updateNotes` upsert with updated frontmatter). Anything untagged stays `private` (§5.1).

### 5.4 Private federation edge A→B (scope `shared`)

Two-step key exchange (as in [federation](https://trip2g.com/docs/en/user/federation), but automated):

1. **On B** (target) — inbound secret; generate a unique `kid` (e.g. `<A-name>-<unixtime>`):
   ```graphql
   createInboundFederationSecret(input: { kid: "A2026", description: "from A" }) -> { kid, secretHex }
   ```
2. **On B** — scope the secret to `shared` (needs `subgraphID` from `allSubgraphs`):
   ```graphql
   addFederationSecretSubgraph(input: { kid: "A2026", subgraphID: <shared.id on B> })
   ```
3. **On A** (source) — outbound secret to B:
   ```graphql
   createOutboundFederationSecret(input: { kid: "A2026", secretHex: "<from step 1>", kbURL: "https://B/_system/mcp" })
   ```
4. **On A** — the KB-note route (otherwise agent A won't know the path; see [hub create](https://trip2g.com/docs/en/hub/_create)):
   ```graphql
   updateNotes(input: { changes: [ { upsert: {
     path: "hub/B.md",
     content: "---\nmcp_federation_kb_url: https://B/_system/mcp\nmcp_federation_kb_id: B\nsubgraph: shared\n---\nWhen to query base B."
   } } ] })
   ```

**KB-note visibility (important).** A KB-note triggers `federated_search` only if the searcher on A can **see** it. So the KB-note's own subgraph must match the searching agent's access: on a shared hub, `subgraph: shared` (or `free: true` for a public hub). Do NOT confuse this with the secret's scope (which is about what A reads from B): the secret scope and the KB-note's subgraph are independent.

**Verification:** on A, `federationSecrets` contains an outbound to `https://B/_system/mcp`; on B, an inbound with the same `kid` and a non-empty scope; `federated_search(kb_id="B")` from A returns results.

### 5.5 Hub (instead of an N² direct mesh)

A hub H = an instance whose vault holds KB-notes for all members (folder `hub/`). For member M:
- M gives H an inbound secret with scope `shared` (H reads M's `shared`) — §5.4 steps 1–2 on M;
- H has an outbound secret + a KB-note `hub/M.md` on M — §5.4 steps 3–4 on H.

Then an agent of any member whose query reaches H (member → H) fans out across all members' `shared`. Symmetrically, the member also federates to H (KB-note `hub/H.md` + secrets) so it can search through the hub.

Multiple hubs: H1↔H2 are linked the same way (§5.4). Mind the **depth limit** (§8) — a hub→hub→member chain counts.

### 5.6 Oversight (read everything, including `private`)

For a selected reader O: when issuing it an inbound secret on each instance, add **all** subgraphs to the scope, including `private`:
```graphql
addFederationSecretSubgraph(input: { kid: "OVERSIGHT", subgraphID: <private.id> })
addFederationSecretSubgraph(input: { kid: "OVERSIGHT", subgraphID: <shared.id> })
# ... + the instance's remaining subgraphs
```
Mark such a secret's `description` as `oversight` so visualization (the panel graph) shows it as deliberate privileged access rather than an accidental leak.

### 5.7 Revoke / narrow

```graphql
revokeFederationSecret(id: <id>)                          # full revoke
removeFederationSecretSubgraph(input: { kid, subgraphID }) # remove one subgraph from scope
```
After a revoke the routing KB-note dangles — delete/rewrite it via `updateNotes`, otherwise the agent keeps getting 401s.

### 5.8 Read/verify state

- `instance_federation(instance_id)` (panel) or `federationSecrets` + `allSubgraphs` + `allFrontmatterPatches` (native).
- Health signs: inbound with a (non-empty!) scope, outbound with the same `kid`, a KB-note to the same `kbURL`, the default-private patch in place.

---

## 6. Default "team" setup (low ceremony)

The sequence the agent runs across the pool (via `list_slots`):

1. On EVERY instance: §5.1 default-private (otherwise the model is leaky).
2. On EVERY instance: §5.2 the `shared` subgraph with a policy.
3. Pick hub(s) H. For each member M: §5.5 (M↔H over `shared`).
4. (Optional) Oversight O: §5.6 on all instances (scope = all subgraphs).
5. Verify: §5.8 everywhere; on the hub `federated_search` returns members' `shared`.

Result: everything private by default; `shared` visible to the team via the hub; oversight sees everything; nobody touched keys by hand.

---

## 7. `instance_federation` / `GET /_system/federation/admin` contract

Aggregated read (admin-only), one JSON:
```json
{
  "self":     { "name", "kb_id", "mcp_url", "subgraphs": [ {id,name} ] },
  "outbound": [ { "id","kid","kb_url","revoked_at","subgraphs":[{id,name}] } ],
  "inbound":  [ { "id","kid","revoked_at","subgraphs":[{id,name}] } ],
  "kb_notes": [ { "kb_id","kb_url","description","path" } ]
}
```
Used by visualization (the simplepanel graph) and by the agent for verification (§5.8).

---

## 8. What depends on the deployment

- **Fan-out depth** — `MCP_FEDERATION_MAX_DEPTH` (default 3). Limits hub→hub→member chains and prevents loops. Plan multi-level hubs against the limit. See [selfhosted](https://trip2g.com/docs/en/user/selfhosted).
- **Per-peer timeout** — `FEDERATED_FANOUT_TIMEOUT` (default 5s). Slow/unreachable peers are reported as errors without blocking the rest.
- **Fan-out parallelism** — `FEDERATED_FANOUT_CONCURRENCY` (default 5). Max peers queried at once.
- **Blind fan-out cap** — `FEDERATED_FANOUT_LIMIT` (default 7). A `federated_search` without `kb_id`/`kb_ids` queries at most this many peers; the rest are listed in the response's `skipped` field for direct follow-up.
- **Default kb_id** = host of the instance's public URL (override with `mcp_federation_kb_id` in the KB-note). Public URL is instance config.
- **Transport** — HMAC-SHA256 only, JWT exp ~30s, no mTLS/OAuth, no replay cache. TLS is not enforced by the hub — use HTTPS peer URLs in production.
- **Control plane** — path §1.1 requires a deployed simplepanel with the pool master secret; §1.2 requires a key with `enable_mcp_admin_tools`. Without either, configuration is manual.
- **`free:true` and public sites** — see the caveat in §5.1.

---

## 9. Future: a thin CLI

After this playbook is battle-tested, a thin CLI (python/js, minimal dependencies) makes sense — a wrapper over `instance_graphql_request`/`graphql_request` with ready subcommands (`default-private`, `make-shared`, `link`, `hub-join`, `oversight`, `status`). It cuts tokens and agent error rates: short commands instead of long GraphQL strings. Build it **after** "it works, now final fixes" on the core playbook.

---

## 10. Link map (trip2g.com/docs)

| Topic | Doc |
|---|---|
| Federation model | /docs/en/user/federation |
| Subgraphs and access | /docs/en/user/advanced |
| Frontmatter patches | /docs/en/user/frontmatter_patches |
| KB-note / hub | /docs/en/hub/_create |
| MCP server and tools | /docs/en/user/mcp |
| Env (depth/timeout) | /docs/en/user/selfhosted |
