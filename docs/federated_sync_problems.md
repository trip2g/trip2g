# Mirroring a federated peer's notes: what blocks it

**Status:** problem statement, not a plan. Nothing here is built. No decision taken.

## The goal

A note carries a map in frontmatter — `sync.<kb_id>: { <remote_note_path>: <local_path> }`
— and something walks it, pulling each remote note from a federated peer into the
local instance as plain local markdown, assets included, so links keep working.

The natural first shape was a fleet role: `sync.*` is config, the walk is
mechanical, and the fleet already writes notes back with a delivery-scoped token.

## What is NOT a problem (ruled out by reading the code)

**Federated calls are not privileged.** `builtinToolHandlers()`
(`internal/case/mcp/resolve.go:195-228`) registers `federated_search`,
`federated_similar`, `federated_note_html` and `federated_expand` with no
authorization wrapper. Only `graphql_introspection` and `graphql_request` are
gated on `mcpAdminToolsEnabled`. Any caller that authenticates at all — API key,
personal token, even anonymous within its own read scope — can make federated
calls.

**The fleet already holds a credential that works.** `/_system/mcp` resolves
`t2g_*` personal tokens through `req.UserToken()` (`internal/case/mcp/endpoint.go:80`,
`internal/appreq/request.go:215-243`), and the fleet is started with
`--trip2g-admin-personal-token`. So the fleet process can call
`federated_note_html` today with zero server-side change. A role would reach it
through a fleet-side tool (`remote_read(kb_id, path)` or similar) implemented on
the admin lane — the same trust boundary `remoteKB.Write`/`.Patch` already use,
where the token lives in the Go host and the sandbox only ever sees the
`agentruntime.KB` interface. Cost of that: any role holding such a tool reads
every peer with the fleet's authority, so it needs a per-role allowlist. The
`sync` map is the natural place to declare it.

**Writing notes and assets locally is already solved.** `updateNotes` returns
`updated[] { path, versionId }` (`internal/graph/schema.graphqls:1871`), and
`uploadNoteAsset` already enforces scoped-token `write_patterns`
(`internal/case/uploadnoteasset/resolve.go:46`, `TestResolve_ScopedToken`). The
delivery token covers both.

**`pass_api_key` is a misleading name.** It does not pass an API key. It makes the
delivery mint `payload.api_token`, a shortapitoken signed with
`ShortAPITokenSecret` carrying depth and read/write patterns
(`internal/case/backjob/delivercronwebhook/resolve.go:139`,
`deliverchangewebhook/resolve.go:96`). The fleet hardcodes it on for both webhook
kinds (`cmd/fleet/internal/fleet/reconcile.go:202,277`).

That token is not one of the three credential types `/_system/mcp` understands —
it is stamped only in the GraphQL middleware (`internal/graph/handler.go:163`) —
so it reaches `/_system/graphql` and nothing else. That is a fact about which
lane the role's own credential can use, not a missing permission.

## Problem 1 — markdown does not cross federation

`handleNoteHTML` returns `string(note.HTML)` (`internal/case/mcp/resolve.go:769`).
There is no format parameter; `model.MCPNoteHTMLParams`
(`internal/model/mcp_params.go:48`) has no such field.

The GraphQL side does not help either. `federated_graphql_request` restricts peers
to `note`, `search`, `similarNotes`, `viewer`
(`internal/case/mcp/graphql_tools.go:65-70`), and the `note` root field returns
`PublicNote` (`internal/graph/schema.graphqls:1522`), which carries `html` but
neither `content` nor `assetReplaces`. `notePaths` is excluded from the allowlist
on purpose — it returns unfiltered paths.

There is no raw-markdown HTTP route anywhere on the public surface.

The peer holds everything needed: `model.NoteView` (`internal/model/note.go:161-213`)
has `Content []byte` and `AssetReplaces` sitting next to the `HTML` that is already
served. `PublicNote` is declared
`@goExtraField(name: "NoteView", type: "*model.NoteView")`, so the object already
carries the struct — exposing markdown is a resolver, not data plumbing.

### Options

| Option | Cost | Notes |
|---|---|---|
| Gated `content` + `assets` on `PublicNote` | S | Field + resolver off `obj.NoteView` + gqlgen regen; gate on `CurrentFederatedScope(ctx)` so the anonymous web never sees it. No new MCP method, `federation_protocol.md` untouched, multi-hop rides the existing graphql hop routing. Requires `MCPFederatedGraphQLEnabled` on the peer (default off). |
| New `federated_note_markdown` MCP tool | M | Six Go files (`model/federation.go`, `internal/federation/client.go`, `mcp/tools.go`, `mcp/resolve.go`, `mcp/federation_handlers.go`, `mcp/endpoint.go`) plus docs; widens the third-party adapter contract. Precedent for size: `federated_expand`. Only advantage: does not need the graphql flag on the peer. |
| A `format` param on `note_html` | — | **Reject.** An older peer silently ignores an unknown JSON field and returns HTML where markdown was asked for. A new method name fails loud. |
| HTML → markdown on the consumer | — | **Reject.** Frontmatter is gone, wikilinks/callouts/embeds are already rendered, asset URLs have to be re-derived by parsing HTML. |

The ACL story is the same for the first two: `canReadMCPNote`
(`internal/case/mcp/resolve.go:401`) routes a federated caller through
`canreadnote.ResolveWithSubgraphs`, which allows `note.Free` **or** an
intersection between the note's `SubgraphNames` and the peer's grant. Federation
already serves non-public content; markdown adds exactly one new thing over HTML
— **frontmatter**, which is where operational config lives (routes, webhook
config, role config, the sync maps themselves). Bounded, but real, and it should
be a deliberate choice rather than a side effect.

## Problem 2 — asset bytes do not cross federation

Two halves, both missing:

**Enumeration.** The list lives on `NoteView.assetReplaces { id url hash absolutePath }`,
which is not on `PublicNote`. Solved for free if Problem 1 is solved by carrying
an asset list in the same payload.

**Bytes.** `GET /_system/assets/{sha256}/{fileName}` (`internal/case/serveasset`)
serves anonymously only when an owning note is publicly readable; a valid
`X-API-Key` grants access outright; otherwise a session plus `CanReadNote` is
required. There is no federation branch. So a note shared with a peer by subgraph
grant has readable text and unreachable images — inconsistent with what federation
already allows for HTML.

### Options

| Option | Cost | Notes |
|---|---|---|
| Anonymous content-addressed GET | none | Works only for publicly readable notes. |
| Sealed peer `X-API-Key` on the fetching side | none | `serveasset` honours a valid API key outright. Coarse (full read on the peer) but operationally honest while one operator runs both instances. |
| Federation branch on `serveasset` | M | Accept the federation credential and gate via `ResolveWithSubgraphs`, same check `canReadMCPNote` already does. Keeps streaming, content-addressing and immutable caching. See the caveat below. |
| Short-lived signed asset URLs minted by the peer | M | Peer returns per-asset URLs with temporary access inside the note payload. Keeps `serveasset` free of federation identity; adds a signing/expiry scheme and a way to verify it. |
| Base64 bytes inside the MCP result | — | **Reject.** `internal/federation/client.go:23` caps responses at 1 MB (`defaultMaxResponseBody`) with a 2s per-request timeout; base64 adds ~33%. `serveasset` deliberately streams to avoid buffering. |
| Hotlink remote URLs without copying | — | Rejected against the stated goal: the mirror stops being self-contained, and private assets 403 for local readers. |

**Caveat on the federation branch.** The federation JWT has no `aud` claim
(`internal/case/mcp/federation_helpers.go:78-84`); its claims are `iss/iat/exp/rid/bh`,
and `bh` binds the token to the request-body digest. `bodyDigest` returns empty
for an empty body (`:351-357`) and `verifyInbound:156` tolerates an empty `bh` by
design. A body-less GET therefore gets a bearer with a 30-second replay window
and no binding. On top of that, `verifyInbound` does a
`ListFederationSecretSubgraphsByKID` lookup — a database hit per asset request, on
a route built around immutable caching that deliberately skips `api_key_log`
because it is hot. Workable, but it is new auth surface, not free reuse.

## Problem 3 — the `sync.<kb_id>: {...}` frontmatter shape does not survive transport

Only relevant if the walker is a fleet role; a server-side job reads `RawMeta`
directly and is unaffected.

`noteViewResolver.Meta` (`internal/graph/schema.resolvers.go:3032-3040`) emits
`NoteViewMeta { key, raw }` with `Raw: fmt.Sprintf("%v", value)` over top-level
`RawMeta` keys. There is no dot-flattening anywhere. A nested YAML map arrives as
Go's map print — `map[remote/a.md:local/a.md]` — unquoted and unordered, and not
recoverable once a key or value contains a space, comma or colon. Discovery then
copies it into a flat `map[string]string`
(`cmd/fleet/internal/fleet/discovery.go:69-71`) before `ParseRole` ever sees it.

This already bit once: `cmd/fleet/internal/fleet/role.go:276-288` carries a lossy
fallback parser with the comment *"trip2g's note meta.raw renders a YAML list as
space-joined inside brackets"*. Nested maps are the same disease, one step worse.

Ways out:

- Author the map as a single flow/JSON string under one flat key
  (`sync_kb: '{"remote/a.md":"local/a.md"}'`). Zero code.
- JSON-encode non-scalar values in the resolver, leaving scalars byte-for-byte.
  Nothing in the frontend breaks: the only consumer of `meta.raw` is
  `assets/ui/admin/noteview/graph/graph.view.ts:303`, which groups graph nodes by
  a scalar field. A list-valued grouping label would change from `[a b]` to
  `["a","b"]`.
- JSON-encode everything including scalars. Cleaner typing, but that one admin
  line then needs a parse-with-fallback, since a scalar would arrive quoted.

Either encoding change also implies `Role.Frontmatter` becoming
`map[string]interface{}` and touching `discovery.go`, `render.go` (Jet vars) and
the input bag.

## Open question raised but not resolved

A third shape for Problem 1 + 2 together: a single new federation method — `note`
/ `federated_note` — returning markdown **and** asset links carrying temporary
access, so one call answers both halves and `serveasset` never learns about
federation. Not priced here. It sits between the two Problem-1 options (it is a
new MCP method, so it widens the adapter contract) and the signed-URL row of
Problem 2.

## Host: role or server-side job

Not decided. Both consulted models independently argued for a server-side mirror
job over a fleet role: the walk is deterministic, there is no LLM judgment
anywhere in it, and a role pays tokens per run for a mechanical copy. The job
would also be the consumer loop that `docs/dev/2026-07-02_federation_subscription.md`
already describes, and it makes Problem 3 disappear.

The counter-argument is that the fleet needs no new credential and no new server
query to reach federation (see above), so a role is not blocked on anything a job
would not also need.

A hybrid was floated: a server-side job produces raw local copies, and a fleet
role transforms local→local afterwards, knowing nothing about federation.

## What any first version would deliberately not support

Multi-hop asset fetch (markdown hops through graphql routing; assets are
single-hop or public-only), hide/delete propagation, conflict handling on locally
edited mirrors (one-way overwrite), assets served by non-trip2g adapters, and any
content transformation on ingest.
