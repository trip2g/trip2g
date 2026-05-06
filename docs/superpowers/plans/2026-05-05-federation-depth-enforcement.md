# Federation Depth Enforcement Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Enforce `FederationMaxDepth` on inbound MCP requests and propagate request depth through the federation client chain, then add bidirectional federation e2e tests.

**Architecture:** Read `X-MCP-Federation-Depth` header in `endpoint.go` — reject early if `>= FederationMaxDepth`, store depth in ctx. Thread ctx through `FederationClient` by adding it to the interface. In `main.go`'s impl, read depth from ctx and set `peer.Depth` so outbound header increments correctly. Separately add a seedvault file + e2e spec for bidirectional federation.

**Tech Stack:** Go (fasthttp, context), Playwright (e2e), moq (mock gen).

---

## File map

| File | Change |
|------|--------|
| `internal/model/federation.go` | Add `ctx` param to `FederationClient` in interface |
| `internal/case/mcp/resolve.go` | Add `federationDepthContextKey`, `contextWithFederationDepth`, exported `FederationDepthFromContext` |
| `internal/case/mcp/endpoint.go` | Read depth header, enforce max, store in ctx |
| `internal/case/mcp/federation.go` | Pass `ctx` to `env.FederationClient` in `fanout` |
| `internal/case/mcp/federation_handlers.go` | Pass `ctx` to `env.FederationClient` in two call sites |
| `cmd/server/main.go` | Update `FederationClient` signature, read depth from ctx, set `peer.Depth`, check per-KB `MaxDepth` |
| `internal/case/mcp/mocks_test.go` | Regenerate via `go generate` |
| `internal/case/mcp/endpoint_dispatch_test.go` | Add depth enforcement tests |
| `testdata/seedvault/hub-kb.md` | New: KB-note pointing back to hub for bidirectional tests |
| `e2e/federation-bidir.spec.js` | New: bidirectional federation e2e spec |
| `scripts/test-e2e.sh` | Add `federation-bidir.spec.js` to Playwright run |

---

## Task 1: Add ctx to FederationClientFactory interface

**Files:**
- Modify: `internal/model/federation.go`

- [ ] **Step 1: Update the interface**

In `internal/model/federation.go`, change line 18:

```go
// Before:
FederationClient(kbID string) (Federation, error)

// After:
FederationClient(ctx context.Context, kbID string) (Federation, error)
```

Add `"context"` to the import if not already present. The full updated file:

```go
package model

import (
	"context"
	"encoding/json"
)

type Federation interface {
	Search(ctx context.Context, params FederationSearchParams) (FederationResult, error)
	Similar(ctx context.Context, params FederationSimilarParams) (FederationResult, error)
	NoteHTML(ctx context.Context, params FederationNoteHTMLParams) (FederationResult, error)
	FederatedSearch(ctx context.Context, params FederationSearchParams) (FederationResult, error)
	FederatedSimilar(ctx context.Context, params FederationSimilarParams) (FederationResult, error)
	FederatedNoteHTML(ctx context.Context, params FederationNoteHTMLParams) (FederationResult, error)
}

type FederationClientFactory interface {
	FederationClient(ctx context.Context, kbID string) (Federation, error)
}

type FederationPeer struct {
	KBID   string
	KBURL  string
	KID    string
	Secret []byte
	Issuer string
	Depth  int
}

type FederationSearchParams struct {
	Query string   `json:"query"`
	KBID  string   `json:"kb_id,omitempty"`
	KBIDs []string `json:"kb_ids,omitempty"`
}

type FederationSimilarParams struct {
	KBID   string `json:"kb_id,omitempty"`
	PID    int64  `json:"pid,omitempty"`
	NoteID int64  `json:"note_id,omitempty"`
	Path   string `json:"path,omitempty"`
	Href   string `json:"href,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

type FederationNoteHTMLParams struct {
	KBID         string `json:"kb_id,omitempty"`
	PID          int64  `json:"pid,omitempty"`
	NoteID       int64  `json:"note_id,omitempty"`
	Path         string `json:"path,omitempty"`
	Href         string `json:"href,omitempty"`
	MatchID      string `json:"match_id,omitempty"`
	ContextWords int    `json:"context_words,omitempty"`
}

type FederationResult struct {
	Content           []FederationContent `json:"content"`
	StructuredContent json.RawMessage     `json:"structuredContent,omitempty"`
	IsError           bool                `json:"isError,omitempty"`
}

type FederationContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}
```

- [ ] **Step 2: Verify it doesn't compile yet** (callers not updated)

```bash
go build ./internal/model/...
```

Expected: compiles fine (interface only).

```bash
go build ./...
```

Expected: compile errors at the 3 call sites in mcp package and in main.go — that's correct, we'll fix them in subsequent tasks.

---

## Task 2: Add depth context helpers in resolve.go

**Files:**
- Modify: `internal/case/mcp/resolve.go`

The existing pattern is `federationAuthContextKey{}` at line 388. Follow it exactly.

- [ ] **Step 1: Add depth context key and helpers after `federationAuthFromContext`**

Append after line ~405 (after `federationAuthFromContext`):

```go
type federationDepthContextKey struct{}

func contextWithFederationDepth(ctx context.Context, depth int) context.Context {
	return context.WithValue(ctx, federationDepthContextKey{}, depth)
}

// FederationDepthFromContext returns the federation hop depth stored in ctx, or 0 if not set.
// Exported so cmd/server/main.go can read it from the request context.
func FederationDepthFromContext(ctx context.Context) int {
	depth, _ := ctx.Value(federationDepthContextKey{}).(int)
	return depth
}
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./internal/case/mcp/...
```

Expected: still compile errors (interface mismatch from Task 1 not yet resolved in this package).

---

## Task 3: Enforce depth in endpoint.go and store in ctx

**Files:**
- Modify: `internal/case/mcp/endpoint.go`

Add `"strconv"` to imports. Read `X-MCP-Federation-Depth` header after the JSON parse, before auth. Reject if `>= FederationMaxDepth`. Store in ctx so downstream can read it.

- [ ] **Step 1: Write failing unit tests**

Add to `internal/case/mcp/endpoint_dispatch_test.go`:

```go
func buildDispatchEnvWithDepth(t *testing.T, maxDepth int) *EnvMock {
	t.Helper()
	env := buildDispatchEnv(t, false)
	env.FederationMaxDepthFunc = func() int { return maxDepth }
	return env
}

func buildMCPFasthttpCtxWithDepth(body []byte, depthHeader string) *fasthttp.RequestCtx {
	ctx := buildMCPFasthttpCtx(body, "")
	if depthHeader != "" {
		ctx.Request.Header.Set("X-MCP-Federation-Depth", depthHeader)
	}
	return ctx
}

func TestMCPEndpointDepthEnforcement(t *testing.T) {
	t.Run("no depth header passes through", func(t *testing.T) {
		env := buildDispatchEnvWithDepth(t, 3)
		httxCtx := buildMCPFasthttpCtxWithDepth(mcpInitBody, "")
		req := wiredRequest(t, httxCtx, env)
		_, err := (&mcp.Endpoint{}).Handle(req)
		require.NoError(t, err)
		var resp mcp.Response
		require.NoError(t, json.Unmarshal(httxCtx.Response.Body(), &resp))
		require.Nil(t, resp.Error)
	})

	t.Run("depth below max passes through", func(t *testing.T) {
		env := buildDispatchEnvWithDepth(t, 3)
		httxCtx := buildMCPFasthttpCtxWithDepth(mcpInitBody, "2")
		req := wiredRequest(t, httxCtx, env)
		_, err := (&mcp.Endpoint{}).Handle(req)
		require.NoError(t, err)
		var resp mcp.Response
		require.NoError(t, json.Unmarshal(httxCtx.Response.Body(), &resp))
		require.Nil(t, resp.Error)
	})

	t.Run("depth equal to max is rejected", func(t *testing.T) {
		env := buildDispatchEnvWithDepth(t, 3)
		httxCtx := buildMCPFasthttpCtxWithDepth(mcpInitBody, "3")
		req := wiredRequest(t, httxCtx, env)
		_, err := (&mcp.Endpoint{}).Handle(req)
		require.NoError(t, err)
		var resp mcp.Response
		require.NoError(t, json.Unmarshal(httxCtx.Response.Body(), &resp))
		require.NotNil(t, resp.Error)
		require.Contains(t, resp.Error.Message, "max depth")
	})

	t.Run("depth above max is rejected", func(t *testing.T) {
		env := buildDispatchEnvWithDepth(t, 3)
		httxCtx := buildMCPFasthttpCtxWithDepth(mcpInitBody, "10")
		req := wiredRequest(t, httxCtx, env)
		_, err := (&mcp.Endpoint{}).Handle(req)
		require.NoError(t, err)
		var resp mcp.Response
		require.NoError(t, json.Unmarshal(httxCtx.Response.Body(), &resp))
		require.NotNil(t, resp.Error)
		require.Contains(t, resp.Error.Message, "max depth")
	})
}
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
go test ./internal/case/mcp/... -run TestMCPEndpointDepthEnforcement -v 2>&1 | head -30
```

Expected: compile error or FAIL (no depth enforcement yet).

- [ ] **Step 3: Add depth enforcement to endpoint.go**

Add `"strconv"` to imports. Insert depth check block after the JSON parse and JSONRPC version check, before auth (line ~29 area). The full updated `Handle` method:

```go
func (*Endpoint) Handle(req *appreq.Request) (interface{}, error) {
	env := req.Env.(Env)

	var rpcReq Request
	err := json.Unmarshal(req.Req.PostBody(), &rpcReq)
	if err != nil {
		resp := errorResponse(nil, ErrCodeParseError, "Parse error: "+err.Error())
		return writeJSONResponse(req, resp)
	}

	if rpcReq.JSONRPC != "2.0" {
		resp := errorResponse(rpcReq.ID, ErrCodeInvalidRequest, "Invalid JSON-RPC version")
		return writeJSONResponse(req, resp)
	}

	// Enforce federation hop depth limit.
	resolveCtx := context.Context(req.Req)
	depthHeader := req.Req.Request.Header.Peek("X-MCP-Federation-Depth")
	if len(depthHeader) > 0 {
		incomingDepth, _ := strconv.Atoi(string(depthHeader))
		if incomingDepth >= env.FederationMaxDepth() {
			resp := errorResponse(rpcReq.ID, ErrCodeInternal, "federation max depth exceeded")
			return writeJSONResponse(req, resp)
		}
		resolveCtx = contextWithFederationDepth(resolveCtx, incomingDepth)
	}

	userToken, utErr := req.UserToken()
	if utErr != nil {
		resp := errorResponse(rpcReq.ID, ErrCodeInternal, "Auth failed: "+utErr.Error())
		return writeJSONResponse(req, resp)
	}

	if userToken == nil {
		authHeader := strings.TrimSpace(string(req.Req.Request.Header.Peek("Authorization")))
		token, isBearerToken := strings.CutPrefix(authHeader, "Bearer ")
		if authHeader != "" && (!isBearerToken || strings.TrimSpace(token) == "") {
			return writeJSONResponse(req, errorResponse(rpcReq.ID, ErrCodeInternal, "Federation auth failed: malformed bearer token"))
		}
		if isBearerToken && strings.TrimSpace(token) != "" {
			kid, allowedSubgraphs, verifyErr := verifyInbound(req.Req, env, strings.TrimSpace(token))
			if verifyErr != nil {
				return writeJSONResponse(req, errorResponse(rpcReq.ID, ErrCodeInternal, "Federation auth failed: "+verifyErr.Error()))
			}
			resolveCtx = contextWithFederationAuth(resolveCtx, kid, allowedSubgraphs)
		}
	}

	rpcReq.MethodOverride = string(req.Req.Request.URI().QueryArgs().Peek("method"))
	resp := Resolve(resolveCtx, env, rpcReq)
	return writeJSONResponse(req, resp)
}
```

Note: `resolveCtx` declaration moved up (was `resolveCtx := context.Context(req.Req)` at line 31 — now declared earlier before the depth block).

- [ ] **Step 4: Run tests**

```bash
go test ./internal/case/mcp/... -run TestMCPEndpointDepthEnforcement -v 2>&1 | head -40
```

Expected: PASS (4 subtests).

- [ ] **Step 5: Commit**

```bash
git add internal/case/mcp/endpoint.go internal/case/mcp/resolve.go internal/case/mcp/endpoint_dispatch_test.go
git commit -m "feat(federation): enforce FederationMaxDepth on inbound MCP requests"
```

---

## Task 4: Thread ctx through FederationClient call sites

**Files:**
- Modify: `internal/case/mcp/federation.go` (fanout)
- Modify: `internal/case/mcp/federation_handlers.go` (2 call sites)

- [ ] **Step 1: Update fanout in federation.go**

Change line 38 in `fanout`:

```go
// Before:
client, err := env.FederationClient(kb.ID)

// After:
client, err := env.FederationClient(ctx, kb.ID)
```

- [ ] **Step 2: Update federation_handlers.go — callFederatedSingleKB**

Change line 44:

```go
// Before:
client, err := env.FederationClient(kb.ID)

// After:
client, err := env.FederationClient(ctx, kb.ID)
```

- [ ] **Step 3: Update federation_handlers.go — handleFederatedSearch**

Change line 82:

```go
// Before:
client, err := env.FederationClient(kb.ID)

// After:
client, err := env.FederationClient(ctx, kb.ID)
```

- [ ] **Step 4: Regenerate the mock**

```bash
go generate ./internal/case/mcp/...
```

Expected: `mocks_test.go` is rewritten. The `FederationClientFunc` field now has signature `func(ctx context.Context, kbID string) (model.Federation, error)`.

- [ ] **Step 5: Update existing tests that set FederationClientFunc**

Search for tests that set `FederationClientFunc`:

```bash
grep -n "FederationClientFunc" /home/alexes/projects2/trip2g/internal/case/mcp/federation_handlers_test.go
```

For each occurrence, add `ctx context.Context` as the first parameter to the func literal. Example:

```go
// Before:
env.FederationClientFunc = func(kbID string) (appmodel.Federation, error) {

// After:
env.FederationClientFunc = func(_ context.Context, kbID string) (appmodel.Federation, error) {
```

Repeat for any other test files that set `FederationClientFunc`.

- [ ] **Step 6: Verify package compiles except main.go**

```bash
go build ./internal/...
```

Expected: compiles. main.go still broken.

- [ ] **Step 7: Run existing mcp tests**

```bash
go test ./internal/case/mcp/... -v 2>&1 | tail -20
```

Expected: all pass.

---

## Task 5: Update main.go FederationClient — propagate depth and enforce per-KB limit

**Files:**
- Modify: `cmd/server/main.go`

- [ ] **Step 1: Update FederationClient signature and body**

Find `func (a *app) FederationClient(kbID string)` at line ~1542. Replace with:

```go
func (a *app) FederationClient(ctx context.Context, kbID string) (model.Federation, error) {
	nvs := a.LatestNoteViews()
	if nvs == nil {
		return nil, fmt.Errorf("federation kb %q not found", kbID)
	}

	depth := mcp.FederationDepthFromContext(ctx)

	for _, kb := range nvs.MCPFederationNotes {
		if kb == nil || kb.ID != kbID {
			continue
		}

		if kb.MaxDepth > 0 && depth >= kb.MaxDepth {
			return nil, fmt.Errorf("federation kb %q max depth %d exceeded", kbID, kb.MaxDepth)
		}

		peer := model.FederationPeer{
			KBID:   kb.ID,
			KBURL:  kb.URL,
			Issuer: a.PublicURL(),
			Depth:  depth,
		}

		reqCtx := a.ctx
		if reqCtx == nil {
			reqCtx = context.Background()
		}
		secretRow, ok, err := a.FederationSecretByKBURL(reqCtx, kb.URL)
		if err != nil {
			return nil, fmt.Errorf("get federation secret by kb url: %w", err)
		}
		if ok {
			secret, decErr := a.DecryptData(secretRow.SecretCrypt)
			if decErr != nil {
				return nil, decErr
			}
			peer.KID = secretRow.Kid
			peer.Secret = secret
		} else {
			configured, confErr := a.HasFederationSecretForKBURL(reqCtx, kb.URL)
			if confErr != nil {
				return nil, fmt.Errorf("check federation secret history by kb url: %w", confErr)
			}
			if configured {
				return nil, fmt.Errorf("no active federation secret for kb_id %q; the configured secret may be revoked", kbID)
			}
		}

		return federation.NewClient(peer, a.webhookHTTPClient), nil
	}

	return nil, fmt.Errorf("federation kb %q not found", kbID)
}
```

Note: `mcp` package needs to be imported. Check if it's already imported; if not, add `"trip2g/internal/case/mcp"` to the import block.

- [ ] **Step 2: Verify full build**

```bash
go build ./...
```

Expected: clean build, no errors.

- [ ] **Step 3: Run all unit tests**

```bash
go test ./internal/... ./cmd/... 2>&1 | tail -20
```

Expected: all pass.

- [ ] **Step 4: Commit**

```bash
git add internal/model/federation.go internal/case/mcp/resolve.go internal/case/mcp/federation.go internal/case/mcp/federation_handlers.go internal/case/mcp/mocks_test.go internal/case/mcp/federation_handlers_test.go cmd/server/main.go
git commit -m "feat(federation): propagate request depth through FederationClient chain"
```

---

## Task 6: Add seedvault hub-kb.md for bidirectional federation

**Files:**
- Create: `testdata/seedvault/hub-kb.md`

- [ ] **Step 1: Create the file**

```markdown
---
title: Hub KB
mcp_federation_kb_url: http://app:20081/_system/mcp
mcp_federation_kb_id: hub
publish: true
free: true
---
Hub knowledge base link for bidirectional federation testing.
```

This uses the docker-internal hostname `app` (the hub container name from `docker-compose.test.yml`). The note is `free: true` so it's accessible to anonymous MCP callers.

- [ ] **Step 2: Commit**

```bash
git add testdata/seedvault/hub-kb.md
git commit -m "test(federation): add hub KB-note to peer seedvault for bidirectional e2e"
```

---

## Task 7: Write federation-bidir.spec.js

**Files:**
- Create: `e2e/federation-bidir.spec.js`

The test bootstraps bidirectional secrets in `beforeAll`:
- Direction A (hub→peer): fresh KID, inbound on peer, outbound on hub (same pattern as `federation.spec.js`)
- Direction B (peer→hub): fresh KID, inbound on hub, outbound on peer

The hub-kb.md note is already pushed to peer via `sync_seedvault_to_peer` in `test-e2e.sh`.

- [ ] **Step 1: Create the spec**

```js
// @ts-check
import { test, expect } from '@playwright/test';
import crypto from 'crypto';
import { graphqlSignIn } from './helpers/auth.js';

/**
 * Bidirectional Federation E2E Tests
 *
 * Verifies peer ↔ hub bidirectional federation:
 * - Peer has hub-kb.md pointing to hub's MCP endpoint
 * - Hub→peer (A) and peer→hub (B) secrets exchanged
 *
 * Key scenario: peer asks hub to search peer ("link back to yourself"):
 *   federated_search(kb_id="hub/peer") → peer→hub→peer.Search → peer's own content
 */

const HUB_URL = process.env.APP_URL || 'http://localhost:20081';
const PEER_URL = 'http://localhost:20091';
const HUB_MCP = `${HUB_URL}/_system/mcp`;
const PEER_MCP = `${PEER_URL}/_system/mcp`;

const SECRET_A = 'f0e1d2c3b4a5968778695a4b3c2d1e0ff0e1d2c3b4a5968778695a4b3c2d1e0f'; // hub→peer
const SECRET_B = '1a2b3c4d5e6f708192a3b4c5d6e7f8091a2b3c4d5e6f708192a3b4c5d6e7f809'; // peer→hub
const KID_A = `e2e-bidir-a-${crypto.randomBytes(4).toString('hex')}`;
const KID_B = `e2e-bidir-b-${crypto.randomBytes(4).toString('hex')}`;

async function gql(request, baseURL, cookie, query, variables = {}) {
  const response = await request.post(`${baseURL}/graphql`, {
    headers: { 'Content-Type': 'application/json', Cookie: cookie },
    data: { query, variables },
  });
  if (!response.ok()) throw new Error(`GraphQL ${baseURL} failed: ${await response.text()}`);
  const body = await response.json();
  if (body.errors) throw new Error(`GraphQL errors: ${JSON.stringify(body.errors)}`);
  return body.data;
}

async function mcpCall(request, url, name, args = {}) {
  const response = await request.post(url, {
    headers: { 'Content-Type': 'application/json' },
    data: { jsonrpc: '2.0', id: 1, method: 'tools/call', params: { name, arguments: args } },
  });
  expect(response.ok(), `MCP ${url} status ${response.status()}`).toBeTruthy();
  return response.json();
}

test.describe.serial('Bidirectional Federation', () => {
  let hubRequest;
  let peerRequest;
  let hubCookie;
  let peerCookie;
  let outboundAId; // hub→peer outbound secret id
  let outboundBId; // peer→hub outbound secret id

  test.beforeAll(async ({ playwright }) => {
    hubRequest = await playwright.request.newContext({ baseURL: HUB_URL });
    const hubJwt = await graphqlSignIn(hubRequest);
    hubCookie = `trip2g_e2e=${hubJwt}`;

    peerRequest = await playwright.request.newContext({ baseURL: PEER_URL });
    const peerJwt = await graphqlSignIn(peerRequest, 'hello@example.com', '111111', { useCache: false });
    peerCookie = `trip2g_e2e_peer=${peerJwt}`;

    // Direction A: hub→peer (hub calls peer)
    await gql(peerRequest, PEER_URL, peerCookie, `
      mutation($input: CreateInboundFederationSecretInput!) {
        admin { createInboundFederationSecret(input: $input) {
          ... on CreateInboundFederationSecretPayload { id kid }
          ... on ErrorPayload { message }
        } }
      }
    `, { input: { kid: KID_A, secretHex: SECRET_A } });

    const outA = await gql(hubRequest, HUB_URL, hubCookie, `
      mutation($input: CreateOutboundFederationSecretInput!) {
        admin { createOutboundFederationSecret(input: $input) {
          ... on CreateOutboundFederationSecretPayload { id kid }
          ... on ErrorPayload { message }
        } }
      }
    `, { input: { kid: KID_A, secretHex: SECRET_A, kbURL: 'http://app-peer:20091/_system/mcp' } });
    outboundAId = outA.admin.createOutboundFederationSecret.id;
    expect(outboundAId).toBeTruthy();

    // Direction B: peer→hub (peer calls hub)
    await gql(hubRequest, HUB_URL, hubCookie, `
      mutation($input: CreateInboundFederationSecretInput!) {
        admin { createInboundFederationSecret(input: $input) {
          ... on CreateInboundFederationSecretPayload { id kid }
          ... on ErrorPayload { message }
        } }
      }
    `, { input: { kid: KID_B, secretHex: SECRET_B } });

    const outB = await gql(peerRequest, PEER_URL, peerCookie, `
      mutation($input: CreateOutboundFederationSecretInput!) {
        admin { createOutboundFederationSecret(input: $input) {
          ... on CreateOutboundFederationSecretPayload { id kid }
          ... on ErrorPayload { message }
        } }
      }
    `, { input: { kid: KID_B, secretHex: SECRET_B, kbURL: 'http://app:20081/_system/mcp' } });
    outboundBId = outB.admin.createOutboundFederationSecret.id;
    expect(outboundBId).toBeTruthy();
  });

  test.afterAll(async () => {
    const revokeQ = `mutation($id: Int64!) { admin { revokeFederationSecret(id: $id) {
      ... on RevokeFederationSecretPayload { revokedId }
      ... on ErrorPayload { message }
    } } }`;
    if (outboundAId) await gql(hubRequest, HUB_URL, hubCookie, revokeQ, { id: outboundAId }).catch(() => {});
    if (outboundBId) await gql(peerRequest, PEER_URL, peerCookie, revokeQ, { id: outboundBId }).catch(() => {});
    await hubRequest?.dispose();
    await peerRequest?.dispose();
  });

  test('peer fan-out includes hub content', async () => {
    const result = await mcpCall(peerRequest, PEER_MCP, 'federated_search', {
      query: 'team status federation',
    });
    expect(result.error).toBeUndefined();
    const text = result.result?.content?.[0]?.text ?? '';
    // Fan-out labels hub results with [hub]
    expect(text).toContain('[hub]');
    expect(text.toLowerCase()).toContain('team');
  });

  test('peer → hub direct: returns hub content', async () => {
    const result = await mcpCall(peerRequest, PEER_MCP, 'federated_search', {
      query: 'team status',
      kb_id: 'hub',
    });
    expect(result.error).toBeUndefined();
    const text = result.result?.content?.[0]?.text ?? '';
    expect(text.toLowerCase()).toContain('team status');
    // URL should come from hub's public URL (localhost:20081 in e2e)
    expect(text).toContain('20081');
  });

  test('peer → hub/peer: peer gets its own content via hub round-trip', async () => {
    // Direct peer search for reference
    const directResult = await mcpCall(peerRequest, PEER_MCP, 'search', {
      query: 'team status',
    });
    const directText = directResult.result?.content?.[0]?.text ?? '';
    expect(directText.toLowerCase()).toContain('team status');

    // Round-trip: peer asks hub to search peer
    const roundTrip = await mcpCall(peerRequest, PEER_MCP, 'federated_search', {
      query: 'team status',
      kb_id: 'hub/peer',
    });
    expect(roundTrip.error).toBeUndefined();
    const text = roundTrip.result?.content?.[0]?.text ?? '';

    // Should get peer's own content back
    expect(text.toLowerCase()).toContain('team status');
    // URL comes from peer (app-peer hostname)
    expect(text).toContain('app-peer');
  });

  test('hub → peer/hub: hub gets its own content via peer round-trip', async () => {
    const result = await mcpCall(hubRequest, `${HUB_URL}/_system/mcp`, 'federated_search', {
      query: 'team status',
      kb_id: 'peer/hub',
    });
    expect(result.error).toBeUndefined();
    const text = result.result?.content?.[0]?.text ?? '';
    expect(text.toLowerCase()).toContain('team status');
    // URL comes from hub
    expect(text).toContain('20081');
  });

  test('depth header at FederationMaxDepth is rejected by hub', async () => {
    // Default FederationMaxDepth is 3. Send depth=3 → hub must reject.
    const response = await hubRequest.post(`${HUB_URL}/_system/mcp`, {
      headers: {
        'Content-Type': 'application/json',
        'X-MCP-Federation-Depth': '3',
      },
      data: { jsonrpc: '2.0', id: 1, method: 'tools/call', params: { name: 'search', arguments: { query: 'test' } } },
    });
    expect(response.ok()).toBeTruthy();
    const body = await response.json();
    const errorMsg = body.error?.message ?? body.result?.content?.[0]?.text ?? '';
    expect(errorMsg.toLowerCase()).toContain('max depth');
  });

  test('revoke peer→hub outbound: peer fan-out no longer returns hub content', async () => {
    // Revoke peer's outbound to hub
    const revokeQ = `mutation($id: Int64!) { admin { revokeFederationSecret(id: $id) {
      ... on RevokeFederationSecretPayload { revokedId }
      ... on ErrorPayload { message }
    } } }`;
    await gql(peerRequest, PEER_URL, peerCookie, revokeQ, { id: outboundBId });
    outboundBId = null; // already revoked, skip afterAll

    // Fan-out should no longer include hub
    const result = await mcpCall(peerRequest, PEER_MCP, 'federated_search', {
      query: 'team status federation',
    });
    const text = result.result?.content?.[0]?.text ?? '';
    // Hub was a configured KB — error must surface, not silent downgrade
    expect(text.toLowerCase()).toMatch(/revoked|no active|secret/);
  });
});
```

- [ ] **Step 2: Commit**

```bash
git add e2e/federation-bidir.spec.js
git commit -m "test(federation): add bidirectional federation e2e spec"
```

---

## Task 8: Wire federation-bidir.spec.js into test-e2e.sh

**Files:**
- Modify: `scripts/test-e2e.sh`

The new spec must run after `sync_seedvault_to_peer` (which pushes hub-kb.md to peer) and before webhooks. It fits naturally after the main Playwright tests block.

- [ ] **Step 1: Find where to add it**

In `test-e2e.sh`, locate the section after the main Playwright run (after `npx playwright test --grep-invert ...`) and before the webhook tests. Around line 195–200 area.

- [ ] **Step 2: Add the spec run**

Add after the main Playwright block and before the Layout CSS section:

```bash
# Run bidirectional federation E2E tests
echo ""
echo "🔗 Running bidirectional federation E2E tests..."
npx playwright test e2e/federation-bidir.spec.js || {
  echo -e "${RED}✗ Bidirectional federation E2E tests failed${NC}"
  exit 1
}
echo -e "${GREEN}✓ Bidirectional federation E2E tests passed${NC}"
```

- [ ] **Step 3: Commit**

```bash
git add scripts/test-e2e.sh
git commit -m "test(federation): run federation-bidir spec in e2e test suite"
```

---

## Self-Review

**Spec coverage:**
- ✅ Inbound depth enforcement: Task 3
- ✅ Depth propagated to outgoing client (peer.Depth): Task 5
- ✅ Per-KB `mcp_federation_kb_max_depth` enforced: Task 5
- ✅ Fan-out includes hub content: Task 7 test 1
- ✅ Peer→hub direct: Task 7 test 2
- ✅ Peer→hub/peer round-trip ("link back to yourself"): Task 7 test 3
- ✅ Hub→peer/hub round-trip: Task 7 test 4
- ✅ Depth header rejected by server: Task 7 test 5
- ✅ Revoke peer→hub breaks fan-out: Task 7 test 6
- ✅ hub-kb.md in seedvault: Task 6
- ✅ test-e2e.sh wired: Task 8

**Placeholder scan:** All steps have concrete code or commands. No "TBD".

**Type consistency:**
- `FederationDepthFromContext` is exported from `resolve.go` and imported in `main.go` as `mcp.FederationDepthFromContext` — consistent.
- `FederationClient(ctx context.Context, kbID string)` signature is the same in interface, impl, and all 3 call sites.
- `peer.Depth = depth` correctly feeds `federation.NewClient` which uses it as `c.peer.Depth` in the outbound header.
