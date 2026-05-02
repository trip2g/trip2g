# Trip2g Federation Protocol — Adapter Implementation Guide

**Audience:** anyone building an MCP-compatible peer/adapter that should be reachable from trip2g hub via `federated_search` / `federated_similar` / `federated_note_html`.

**Status:** v1, current as of 2026-05-02. Reference implementations:
- Hub side: `internal/case/mcp/`, `internal/federation/`, `internal/model/federation.go` in trip2g.
- Peer side (telegram): `trip2g_telegram_adapter` (after the fixes described in this doc).

---

## 1. Transport

- HTTP POST `<base>/_system/mcp`.
- Request body: JSON-RPC 2.0 envelope.
- Response body: JSON-RPC 2.0 envelope.
- Content-Type: `application/json` both directions.
- Hub default fan-out timeout: 2 seconds per peer (`MCP_FEDERATION_FANOUT_TIMEOUT` env on hub).
- Recursion depth limit (hub-enforced): 3 (`MCP_FEDERATION_MAX_DEPTH`).

A peer is a black box behind one URL. There is no separate metadata endpoint; everything goes through `tools/list` and `tools/call`.

---

## 2. Authentication

### 2.1 Channels accepted by the adapter

The peer MUST inspect `Authorization: Bearer <value>` and `?token=<value>` query parameter. Hub uses the header.

### 2.2 Federation HMAC JWT (verbatim spec)

Federation tokens are HMAC-SHA256 JWTs. The peer treats any non-empty Bearer/`?token=` value as a JWT and validates it per the rules below. Anything that fails validation → anonymous Auth.

**Header:**
```json
{"alg":"HS256","typ":"JWT","kid":"<kid>"}
```

**Payload:**
```json
{
  "iss": "<hub-url>",
  "aud": "<peer-url>",
  "rid": "<request-uuid>",
  "iat": <unix-seconds>,
  "exp": <unix-seconds>
}
```

**Signature:** `HMAC-SHA256(<base64url(header)>.<base64url(payload)>, secret_bytes)`, base64url-encoded without padding.

**Validation rules on the peer:**
1. Three segments, each base64url-decodable.
2. Header `alg == "HS256"` and `typ == "JWT"`. Reject otherwise.
3. Header `kid` is non-empty; look it up in the peer's federation_secrets store. If not found or revoked → anonymous Auth (NOT error).
4. `signature == HMAC-SHA256(header.payload, secret_bytes)`. Constant-time compare. Mismatch → anonymous.
5. `iat` and `exp` are present. Allow ±10s clock skew. Outside the window → anonymous.
6. `aud` (when present and non-empty) MUST equal the peer's own base URL. Mismatch → anonymous.
7. (Optional, recommended) `rid` MUST NOT have been seen before within `exp`. Track in-memory `sync.Map[rid → exp]` with periodic eviction. Replay → anonymous.

**Failure mode = anonymous, NOT error.** Returning HTTP 401 or JSON-RPC `error` for an unrecognized kid leaks existence. Always return a successful JSON-RPC envelope with empty results.

### 2.3 Authorization context

After verification the peer constructs an internal Auth object:

```go
type Auth struct {
    KID       string            // for logging
    AccountID string            // peer-local scope owner
    Allowed   map[string]bool   // scoped resource ids (e.g. dialog ids)
    AllowAll  bool              // when scope is "*"
}
```

Map federation `kid` → adapter-local scope: which resources this caller may see. Trip2g hub does not control scope; the peer admin defines it at federation_secret creation time.

### 2.4 Issuing federation_secrets (peer admin UI)

Peer MUST expose admin endpoints to:
- create a secret: 32 random bytes, hex-encoded, returned **once** to the admin (mirrors trip2g UI).
- list secrets per scope: `kid`, scope, created_at, last_used_at, revoked_at.
- revoke (soft delete): `revoked_at = now()`.

Storage: encrypt `secretHex` at rest with the peer's master key. Verification needs raw bytes, so do NOT hash.

The two-step exchange (peer admin generates → sends `kid + secretHex` to hub admin → hub admin pastes into trip2g admin → hub stores encrypted in its own `federation_secrets`) is described in trip2g's user federation docs.

---

## 3. JSON-RPC envelope

### 3.1 Request

```json
{
  "jsonrpc": "2.0",
  "method": "<method-name>",
  "params": {...},
  "id": "<unique-string-or-int>"
}
```

Hub uses string IDs (UnixNano timestamp). Peer SHOULD echo `id` verbatim in the response.

### 3.2 Response (success)

```json
{
  "jsonrpc": "2.0",
  "id": "<echoed>",
  "result": {...}
}
```

### 3.3 Response (error)

```json
{
  "jsonrpc": "2.0",
  "id": "<echoed>",
  "error": {"code": <int>, "message": "<text>"}
}
```

**Use JSON-RPC `error` ONLY for protocol/transport problems** (parse error, malformed envelope, method-not-found, transport timeout). Codes:
- `-32700` parse error
- `-32600` invalid request
- `-32601` method not found
- `-32602` invalid params
- `-32603` internal error (transport, infra)

**Tool execution failures are NOT JSON-RPC errors.** They go inside `result` with `isError: true` (see §5).

---

## 4. Required methods

A peer MUST implement these four methods. Hub never queries anything else.

| Method | Purpose | Required by |
|---|---|---|
| `initialize` | Handshake, capabilities advertisement | All MCP clients |
| `tools/list` | Enumerate tools | Hub during peer discovery |
| `tools/call` | Execute a tool | All federated_* operations |
| `notifications/initialized` | No-op (client → server notice) | All MCP clients |

### 4.1 `initialize`

Request `params`: capabilities object (ignored by peer in v1).

Response `result`:
```json
{
  "protocolVersion": "2024-11-05",
  "capabilities": {"tools": {}},
  "serverInfo": {"name": "<peer-name>", "version": "<semver>"}
}
```

### 4.2 `tools/list`

Request `params`: empty.

Response `result`:
```json
{
  "tools": [
    {
      "name": "search",
      "description": "<human-readable, includes input format examples>",
      "inputSchema": {"type": "object", "properties": {...}, "required": [...]}
    },
    {"name": "note_html", ...},
    {"name": "similar", ...}
  ]
}
```

A peer MAY expose only `search` and `note_html` if `similar` is meaningless for it. Trip2g hub will gracefully degrade — `federated_similar` calls are not used by the agent unless it explicitly asks for similar.

### 4.3 `tools/call`

Request `params`:
```json
{"name": "<tool-name>", "arguments": {...}}
```

Response `result`: **MUST be a `CallToolResult`** (see §5).

### 4.4 `notifications/initialized`

Client → server only. Peer MAY respond with empty `result: {}` or treat as a notification (no response required by JSON-RPC, but trip2g hub accepts an echo).

---

## 5. CallToolResult — the canonical response shape

Every `tools/call` response MUST conform to:

```json
{
  "content": [{"type": "text", "text": "<short summary, <=500 chars>"}],
  "structuredContent": <any JSON value, optional>,
  "isError": <bool, optional, default false>
}
```

Reference Go type (`internal/model/federation.go`):
```go
type FederationResult struct {
    Content           []FederationContent `json:"content"`
    StructuredContent json.RawMessage     `json:"structuredContent,omitempty"`
    IsError           bool                `json:"isError,omitempty"`
}
type FederationContent struct {
    Type string `json:"type"` // always "text"
    Text string `json:"text"`
}
```

### 5.1 `content` (REQUIRED, even if empty)

Array of human/agent-readable text items. **Always at least one item** when results exist. For empty result sets, return one item with a summary like `"Found 0 results for query \"X\""` — this lets clients without `structuredContent` handling still display something.

`type` is always `"text"` in v1. Image/audio types reserved for future use.

### 5.2 `structuredContent` (OPTIONAL but strongly recommended)

Tool-specific JSON. Hub forwards this to the agent. Agents that understand the structure parse it; legacy clients fall back to `content[0].text`.

Conventions per tool (see §6).

### 5.3 `isError` (OPTIONAL, default false)

Set `true` when the tool failed in a way the **agent should know about** (no results due to scope, embedding service down, malformed query, rate limited). The HTTP status is still 200 and `result` is still populated; the error is **inside** the tool result.

When `isError: true`:
- `content[0].text` MUST contain a human-readable error message.
- `structuredContent` MAY contain machine-readable error fields (`{error_code, retry_after}`).
- Hub will surface this to the agent; the federated_search fan-out will not include error results in merged output.

**Do NOT use `isError` for auth failures** — those return empty results (anonymous), per §2.

---

## 6. Tool-specific conventions

### 6.1 `search`

**Arguments:**
```json
{"query": "<plain text or JSON DSL string>"}
```

The peer is free to define a DSL inside `query` (telegram adapter accepts JSON DSL with `op`, `dialog_kinds`, `after`, etc.). Hub treats `query` opaquely.

**`structuredContent`:**
```json
{
  "query": "<echo>",
  "results": [
    {
      "note_id": <uint64>,         // peer-stable id, opaque to hub
      "note_path": "<peer-route>",
      "title": "<display title>",
      "score": <float>,            // 0..1, present for semantic search
      "url": "<https URL>",
      "href": "<deep link, optional>",
      "kind": "<peer-defined>",
      "matches": [
        {"match_id": "<id>", "snippet": "<...>", "context_words": <int>}
      ],
      "proof": {<peer-defined provenance>}
    }
  ]
}
```

**`content[0].text`:** `"Found N results for \"<query>\""` plus optionally a one-line digest of the top match.

### 6.2 `note_html`

**Arguments:** at least one of `pid`, `note_id`, `path`, `href`, plus optional `match_id`, `context_words`.

**`structuredContent`:**
```json
{
  "html": "<sanitized full HTML, optional>",
  "payload": {<peer-defined raw fields>},
  "queued": <bool, optional>
}
```

**`content[0].text`:** `"Loaded note <id-or-path>"` or a brief plain-text excerpt (≤500 chars).

### 6.3 `similar`

**Arguments:** at least one of `pid`, `note_id`, `path`, `href`. Optional `limit`.

**`structuredContent`:** same shape as `search.results`.

**`content[0].text`:** `"Found N similar notes to <id>"`.

### 6.4 `federated_*` (recursive federation)

Peers MAY chain to their own peers. If a peer chooses NOT to (telegram adapter does not), it MUST still respond with a valid `CallToolResult`:

```json
{"content": [{"type":"text","text":"This peer does not federate further."}],
 "structuredContent": {"results": []}}
```

**Do NOT return JSON-RPC `error: -32601 method not found`** — hub treats peers as opaque; the method exists in the schema, the peer just chose not to walk further. Empty CallToolResult is correct.

---

## 7. Hub-side behavior (informational)

Hub fans out `federated_search` without `kb_id` to all peers visible to the calling user (KB-notes filtered by subgraph access). Each peer call:

1. Hub looks up `federation_secrets` row by `kb_url`.
2. Decrypts secret, signs JWT for kid.
3. POSTs `tools/call search` with the user's query.
4. Reads response into `model.FederationResult`.
5. Aggregates `result.structuredContent.results` from all peers, sorts by score, takes top 20.
6. Returns merged result to the agent in hub-shaped CallToolResult.

If a peer returns `isError: true`, its results are skipped but other peers proceed.

If a peer returns malformed JSON or non-CallToolResult shape, hub silently drops the peer's contribution from the merged output. **This is the bug previously seen with telegram adapter** — fix is in §3 of the adapter TZ.

---

## 8. Replay protection details

If the peer chooses to implement replay protection (recommended):

- Cache: `sync.Map[rid string]int64` where value is `exp`.
- On request: lookup `rid`. If present → reject (anonymous). Else insert.
- Eviction: every minute, walk map, drop entries where `exp < now()`.
- Memory bound: O(rps × 30s window). At 100 rps this is ~3000 entries. Fine for sync.Map.

---

## 9. Observability

Peers SHOULD emit (logs/metrics):
- per-request: kid, hub-iss, latency, status (auth_ok / auth_anon / tool_error / ok).
- per-secret: `last_used_at` updated on successful auth (best-effort, fire-and-forget).

Trip2g hub logs incoming federation auth under prefix `mcp:federation`. Peers SHOULD do the same so two-instance debugging is grep-friendly.

---

## 10. Versioning

This is v1. Future versions will bump `protocolVersion` in `initialize`. Compatibility rules:

- `protocolVersion` strict equality required for new features.
- Backward additions (new optional fields in `structuredContent`) — non-breaking.
- New required fields → bump version.
- Deprecated fields kept for one major version with a warning log.

---

## 11. Conformance checklist for adapter authors

- [ ] HTTP POST `/_system/mcp` accepts JSON-RPC 2.0.
- [ ] `Authorization: Bearer` AND `?token=` both checked.
- [ ] JWT shape detection (3 dot-separated base64 segments) routes to federation auth.
- [ ] HS256 verification with `kid` lookup, `iat`/`exp` validation, optional `aud`/`rid` checks.
- [ ] Auth failures return anonymous (NOT JSON-RPC error or HTTP 4xx).
- [ ] Admin UI for federation_secrets CRUD with one-time plaintext display.
- [ ] `initialize`, `tools/list`, `tools/call`, `notifications/initialized` implemented.
- [ ] **Every `tools/call` response is a `CallToolResult`** with `content` + optional `structuredContent`.
- [ ] Tool failures use `isError: true` inside result, NOT JSON-RPC error.
- [ ] `federated_*` methods return empty CallToolResult, not method-not-found.
- [ ] Replay protection (recommended).
- [ ] Logs prefixed `mcp:federation` for grep parity with hub.

---

## 12. Reference implementations

- **Hub-side validator** (peer auth): `internal/case/mcp/federation_helpers.go` (`verifyInbound`, `parseFederationJWT`).
- **Hub-side signer** (outbound): `internal/case/mcp/federation_helpers.go` (`signOutboundAt`).
- **Hub-side client** (calls peers): `internal/federation/client.go` (`callTool`).
- **Hub-side aggregator**: `internal/case/mcp/federation_handlers.go` (`handleFederatedSearch`).
- **Adapter Authorize**: `trip2g_telegram_adapter/internal/mcp/mcp.go` (`Authorize`, `authorizeFederation`).
- **Adapter store**: `trip2g_telegram_adapter/internal/store/store.go` (`FederationSecret`, `FederationSecretByKID`).

---

## 13. Out of scope (planned for v2)

- mTLS / OAuth.
- Per-request per-user `on_behalf_of` claim (currently scope is per-`kid`, not per-end-user).
- Streaming responses (SSE) for long-running searches.
- Image/audio content types in `content[]`.
- Negotiated `protocolVersion` minimum during `initialize`.
