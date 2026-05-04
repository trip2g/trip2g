# Federation ACL Tests Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add two new federation peer services (peer2, peer3) and a comprehensive e2e test suite that proves private and subgraph-gated KB-notes never leak to unauthorized callers — across search, targeted routing, and fan-out.

**Architecture:** The hub already filters KB-notes via `accessibleKBNotes` before any fan-out HTTP call, and `FederationClient` explicitly refuses to downgrade a configured-private peer to anonymous when its outbound secret is revoked. The tests assert these invariants hold for three principals (anonymous, scoped-user, admin) × three peers (open, semi-private, admin-only).

**Tech Stack:** Playwright, docker-compose, existing `gql`/`mcpCallFresh` helpers from `e2e/helpers/auth.js`.

**Spec:** `docs/superpowers/specs/2026-05-04-federation-acl-test-design.md`

---

## File map

| Action | File | Responsibility |
|--------|------|----------------|
| Create | `testdata/seedvault2/peer2-note.md` | Unique note on peer2 with identifiable content string |
| Create | `testdata/seedvault3/peer3-note.md` | Unique note on peer3 with identifiable content string |
| Create | `docs/demo/federation_kb_semi.md` | Semi-private KB-note for peer2 (`subgraphs: federation-test`) |
| Create | `docs/demo/federation_kb_private.md` | Admin-only KB-note for peer3 (no `free:`, no `subgraphs:`) |
| Modify | `docker-compose.test.yml` | Add `app-peer2` (20093/20094) and `app-peer3` (20095/20096) services |
| Modify | `scripts/test-e2e.sh` | Add waitfor + sync functions for peer2 and peer3 |
| Create | `e2e/federation-acl.spec.js` | Full ACL test suite (13 tests across 4 groups) |

No Go backend changes. All ACL logic already exists.

---

### Task 1: Seed notes and KB-notes (static files)

**Files:**
- Create: `testdata/seedvault2/peer2-note.md`
- Create: `testdata/seedvault3/peer3-note.md`
- Create: `docs/demo/federation_kb_semi.md`
- Create: `docs/demo/federation_kb_private.md`

- [ ] **Step 1: Create peer2 seed note**

```bash
mkdir -p testdata/seedvault2
```

`testdata/seedvault2/peer2-note.md`:
```markdown
---
title: Peer2 Note
free: true
---

Semi-private peer2 content for federation ACL testing.
```

- [ ] **Step 2: Create peer3 seed note**

```bash
mkdir -p testdata/seedvault3
```

`testdata/seedvault3/peer3-note.md`:
```markdown
---
title: Peer3 Note
free: true
---

Admin-only peer3 content for federation ACL testing.
```

- [ ] **Step 3: Create semi-private KB-note on hub**

`docs/demo/federation_kb_semi.md`:
```markdown
---
title: "Peer2 Federation KB"
subgraphs: federation-test
mcp_federation_kb_url: http://app-peer2:20093/_system/mcp
mcp_federation_kb_id: peer2
---

Semi-private knowledge base for federation ACL testing.
Accessible only to users with the federation-test subgraph.
```

- [ ] **Step 4: Create admin-only KB-note on hub**

`docs/demo/federation_kb_private.md`:
```markdown
---
title: "Peer3 Federation KB"
mcp_federation_kb_url: http://app-peer3:20095/_system/mcp
mcp_federation_kb_id: peer3
---

Admin-only knowledge base for federation ACL testing.
No free: true, no subgraphs: — readable only by admin.
```

- [ ] **Step 5: Commit**

```bash
git add testdata/seedvault2/ testdata/seedvault3/ docs/demo/federation_kb_semi.md docs/demo/federation_kb_private.md
git commit -m "test(federation): add peer2/peer3 seed notes and hub KB-notes for ACL tests"
```

---

### Task 2: Docker services for peer2 and peer3

**Files:**
- Modify: `docker-compose.test.yml`

Add two services after the `app-peer` block (before the `volumes:` section). Pattern is identical to `app-peer` with adjusted ports, DB path, minio prefix, JWT secret, cookie name, and vault path.

- [ ] **Step 1: Add app-peer2 and app-peer3 to docker-compose.test.yml**

Open `docker-compose.test.yml`. Find the line `volumes:` at the end of the file (after app-peer's healthcheck). Insert before it:

```yaml

  app-peer2:
    build:
      context: .
      dockerfile: Dockerfile
    container_name: trip2g-test-app-peer2
    depends_on:
      minio:
        condition: service_healthy
      embedding:
        condition: service_healthy
    ports:
      - "20093:20093"
      - "20094:20094"
    environment:
      - LISTEN_ADDR=0.0.0.0:20093
      - INTERNAL_LISTEN_ADDR=:20094
      - DB_FILE=/data/peer2.sqlite3
      - DEV=true
      - LOG_LEVEL=info
      - OWNER_EMAIL=hello@example.com
      - SHUTDOWN_GRACE_PERIOD=1ms
      - SHUTDOWN_TIMEOUT=1ms
      - 'FEATURES={"vector_search": {"enabled": true, "model": "bge-m3", "base_url": "http://embedding:8000/v1"}}'
      - MINIO_ENDPOINT=minio:29000
      - MINIO_ACCESS_KEY_ID=testuser
      - MINIO_SECRET_KEY=testpassword
      - MINIO_BUCKET=test-bucket
      - MINIO_PREFIX=peer2/
      - MINIO_USE_SSL=false
      - PUBLIC_URL=http://app-peer2:20093
      - JWT_SECRET=test-peer2-secret-key-do-not-use
      - USER_TOKEN_COOKIE_NAME=trip2g_e2e_peer2
      - GIT_API_REPO_PATH=/app/testdata/seedvault2
      - GIT_API_BASE_PATH=/git
      - RESEND_API_KEY=test-key
      - MAIL_FROM=test@example.com
      - MCP_FEDERATION_MAX_DEPTH=3
    volumes:
      - ./tmp/data:/data
      - ./testdata/seedvault2:/app/testdata/seedvault2:ro
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:20094/healthz"]
      interval: 5s
      timeout: 5s
      retries: 10
      start_period: 10s

  app-peer3:
    build:
      context: .
      dockerfile: Dockerfile
    container_name: trip2g-test-app-peer3
    depends_on:
      minio:
        condition: service_healthy
      embedding:
        condition: service_healthy
    ports:
      - "20095:20095"
      - "20096:20096"
    environment:
      - LISTEN_ADDR=0.0.0.0:20095
      - INTERNAL_LISTEN_ADDR=:20096
      - DB_FILE=/data/peer3.sqlite3
      - DEV=true
      - LOG_LEVEL=info
      - OWNER_EMAIL=hello@example.com
      - SHUTDOWN_GRACE_PERIOD=1ms
      - SHUTDOWN_TIMEOUT=1ms
      - 'FEATURES={"vector_search": {"enabled": true, "model": "bge-m3", "base_url": "http://embedding:8000/v1"}}'
      - MINIO_ENDPOINT=minio:29000
      - MINIO_ACCESS_KEY_ID=testuser
      - MINIO_SECRET_KEY=testpassword
      - MINIO_BUCKET=test-bucket
      - MINIO_PREFIX=peer3/
      - MINIO_USE_SSL=false
      - PUBLIC_URL=http://app-peer3:20095
      - JWT_SECRET=test-peer3-secret-key-do-not-use
      - USER_TOKEN_COOKIE_NAME=trip2g_e2e_peer3
      - GIT_API_REPO_PATH=/app/testdata/seedvault3
      - GIT_API_BASE_PATH=/git
      - RESEND_API_KEY=test-key
      - MAIL_FROM=test@example.com
      - MCP_FEDERATION_MAX_DEPTH=3
    volumes:
      - ./tmp/data:/data
      - ./testdata/seedvault3:/app/testdata/seedvault3:ro
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:20096/healthz"]
      interval: 5s
      timeout: 5s
      retries: 10
      start_period: 10s
```

- [ ] **Step 2: Commit**

```bash
git add docker-compose.test.yml
git commit -m "test(federation): add app-peer2 and app-peer3 docker services"
```

---

### Task 3: Wire peer2 and peer3 into the e2e shell script

**Files:**
- Modify: `scripts/test-e2e.sh`

Two changes: (a) wait for peer2/peer3 to start, (b) push seedvault2/seedvault3 to them.

- [ ] **Step 1: Add sync helper functions after `sync_seedvault_to_peer` function (around line 129)**

Find the line `# NOTE: presigned MinIO URLs` (after the closing `}` of `sync_seedvault_to_peer`) and insert before it:

```bash
# Helper: sign in to peer2 and push seedvault2 content
sync_seedvault2_to_peer2() {
  local PEER_URL="http://localhost:20093"
  local PEER_GRAPHQL="$PEER_URL/graphql"

  echo "🔄 Setting up peer2 instance (seedvault2 push)..."

  curl -sf -X POST "$PEER_GRAPHQL" \
    -H 'Content-Type: application/json' \
    -d '{"query":"mutation { requestEmailSignInCode(input: { email: \"hello@example.com\" }) { ... on RequestEmailSignInCodePayload { success } } }"}' > /dev/null

  local PEER_TOKEN
  PEER_TOKEN=$(curl -sf -X POST "$PEER_GRAPHQL" \
    -H 'Content-Type: application/json' \
    -d '{"query":"mutation { signInByEmail(input: { email: \"hello@example.com\", code: \"111111\" }) { ... on SignInPayload { token } } }"}' \
    | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

  if [ -z "$PEER_TOKEN" ]; then
    echo -e "${RED}✗ Failed to sign in to peer2${NC}"
    return 1
  fi

  local PEER_API_KEY
  PEER_API_KEY=$(curl -sf -X POST "$PEER_GRAPHQL" \
    -H 'Content-Type: application/json' \
    -H "Cookie: trip2g_e2e_peer2=$PEER_TOKEN" \
    -d '{"query":"mutation AdminCreateApiKey($input: CreateApiKeyInput!) { admin { createApiKey(input: $input) { ... on ErrorPayload { message } ... on CreateApiKeyPayload { value } } } }","variables":{"input":{"description":"demo"}}}' \
    | grep -o '"value":"[^"]*"' | cut -d'"' -f4)

  if [ -z "$PEER_API_KEY" ]; then
    echo -e "${RED}✗ Failed to create peer2 API key${NC}"
    return 1
  fi

  npx tsx obsidian-sync/src/sync/cli/cmd.ts --folder testdata/seedvault2 --api-key "$PEER_API_KEY" --api-url "$PEER_GRAPHQL"
  echo -e "${GREEN}✓ Seedvault2 pushed to peer2${NC}"
}

# Helper: sign in to peer3 and push seedvault3 content
sync_seedvault3_to_peer3() {
  local PEER_URL="http://localhost:20095"
  local PEER_GRAPHQL="$PEER_URL/graphql"

  echo "🔄 Setting up peer3 instance (seedvault3 push)..."

  curl -sf -X POST "$PEER_GRAPHQL" \
    -H 'Content-Type: application/json' \
    -d '{"query":"mutation { requestEmailSignInCode(input: { email: \"hello@example.com\" }) { ... on RequestEmailSignInCodePayload { success } } }"}' > /dev/null

  local PEER_TOKEN
  PEER_TOKEN=$(curl -sf -X POST "$PEER_GRAPHQL" \
    -H 'Content-Type: application/json' \
    -d '{"query":"mutation { signInByEmail(input: { email: \"hello@example.com\", code: \"111111\" }) { ... on SignInPayload { token } } }"}' \
    | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

  if [ -z "$PEER_TOKEN" ]; then
    echo -e "${RED}✗ Failed to sign in to peer3${NC}"
    return 1
  fi

  local PEER_API_KEY
  PEER_API_KEY=$(curl -sf -X POST "$PEER_GRAPHQL" \
    -H 'Content-Type: application/json' \
    -H "Cookie: trip2g_e2e_peer3=$PEER_TOKEN" \
    -d '{"query":"mutation AdminCreateApiKey($input: CreateApiKeyInput!) { admin { createApiKey(input: $input) { ... on ErrorPayload { message } ... on CreateApiKeyPayload { value } } } }","variables":{"input":{"description":"demo"}}}' \
    | grep -o '"value":"[^"]*"' | cut -d'"' -f4)

  if [ -z "$PEER_API_KEY" ]; then
    echo -e "${RED}✗ Failed to create peer3 API key${NC}"
    return 1
  fi

  npx tsx obsidian-sync/src/sync/cli/cmd.ts --folder testdata/seedvault3 --api-key "$PEER_API_KEY" --api-url "$PEER_GRAPHQL"
  echo -e "${GREEN}✓ Seedvault3 pushed to peer3${NC}"
}
```

- [ ] **Step 2: Add waitfor calls for peer2 and peer3**

Find the block:
```bash
# Wait for peer instance (federation e2e)
./scripts/waitfor localhost:20091 || {
  echo -e "${RED}✗ Peer service failed to start${NC}"
  exit 1
}
```

Add after it:
```bash
# Wait for peer2 and peer3 instances (federation ACL e2e)
./scripts/waitfor localhost:20093 || {
  echo -e "${RED}✗ Peer2 service failed to start${NC}"
  exit 1
}

./scripts/waitfor localhost:20095 || {
  echo -e "${RED}✗ Peer3 service failed to start${NC}"
  exit 1
}
```

- [ ] **Step 3: Add sync calls for peer2 and peer3**

Find the block:
```bash
# Push seedvault to peer instance for federation tests
sync_seedvault_to_peer || {
  echo -e "${RED}✗ Peer seedvault sync failed${NC}"
  exit 1
}
echo ""
```

Add after it:
```bash
sync_seedvault2_to_peer2 || {
  echo -e "${RED}✗ Peer2 seedvault sync failed${NC}"
  exit 1
}
echo ""

sync_seedvault3_to_peer3 || {
  echo -e "${RED}✗ Peer3 seedvault sync failed${NC}"
  exit 1
}
echo ""
```

- [ ] **Step 4: Commit**

```bash
git add scripts/test-e2e.sh
git commit -m "test(federation): wire peer2 and peer3 sync into e2e script"
```

---

### Task 4: Write the federation-acl spec

**Files:**
- Create: `e2e/federation-acl.spec.js`

This is one serial describe block with 4 groups of tests. The `beforeAll` sets up all state; tests are read-only against that state except Group 4 which revokes peer2's outbound secret.

- [ ] **Step 1: Write e2e/federation-acl.spec.js**

```js
// @ts-check
import { test, expect } from '@playwright/test';
import crypto from 'crypto';
import { graphqlSignIn, createPersonalToken, revokePersonalToken } from './helpers/auth.js';

/**
 * Federation ACL E2E Tests
 *
 * Three-tier KB-note access control:
 *   peer  (port 20091) — open, free: true KB-note
 *   peer2 (port 20093) — semi-private, subgraphs: federation-test KB-note
 *   peer3 (port 20095) — admin-only, no free/subgraphs KB-note
 *
 * Three principals:
 *   anonymous  — no auth header
 *   scoped     — non-admin user with federation-test subgraph personal token
 *   admin      — admin personal token
 */

const HUB_URL = process.env.APP_URL || 'http://localhost:20081';
const PEER2_URL = 'http://localhost:20093';
const PEER3_URL = 'http://localhost:20095';
const HUB_MCP = `${HUB_URL}/_system/mcp`;

// Deterministic secrets — unique per run via KID suffix.
const SECRET_HEX_P2 = 'b1c1d1e1f1020304050607080900a1b1c1d1e1f1020304050607080900a1b100';
const SECRET_HEX_P3 = 'c2d2e2f2030405060708090a0b0c0d0e0f1011121314151617181900c2d2e2f2';
const KID_P2 = `e2e-acl-p2-${crypto.randomBytes(4).toString('hex')}`;
const KID_P3 = `e2e-acl-p3-${crypto.randomBytes(4).toString('hex')}`;

/** GraphQL helper with cookie auth. */
async function gql(request, baseURL, cookie, query, variables = {}) {
  const response = await request.post(`${baseURL}/graphql`, {
    headers: { 'Content-Type': 'application/json', Cookie: cookie },
    data: { query, variables },
  });
  if (!response.ok()) {
    const body = await response.text();
    throw new Error(`GraphQL ${baseURL} failed: ${body}`);
  }
  const body = await response.json();
  if (body.errors) throw new Error(`GraphQL errors: ${JSON.stringify(body.errors)}`);
  return body.data;
}

/**
 * MCP call in a fresh cookie-free context.
 * Cookie wins over Bearer, so fresh context is required for personal token tests.
 */
async function mcpFresh(playwright, method, params = {}, token = null) {
  const ctx = await playwright.request.newContext({});
  try {
    const headers = { 'Content-Type': 'application/json' };
    if (token) headers['Authorization'] = `Bearer ${token}`;
    const response = await ctx.post(HUB_MCP, {
      headers,
      data: { jsonrpc: '2.0', id: 1, method, params },
    });
    expect(response.ok(), `MCP ${method} status ${response.status()}`).toBeTruthy();
    return response.json();
  } finally {
    await ctx.dispose();
  }
}

test.describe.serial('Federation ACL', () => {
  let hubRequest;
  let peer2Request;
  let peer3Request;
  let hubCookie;
  let peer2Cookie;
  let peer3Cookie;

  let adminToken;
  let adminTokenId;
  let scopedToken;
  let scopedTokenId;
  let scopedCookie;
  let scopedCtx;

  let outboundPeer2Id;
  let outboundPeer3Id;

  test.beforeAll(async ({ playwright }) => {
    // ── Sign into all three admins ────────────────────────────────────────────
    hubRequest = await playwright.request.newContext({ baseURL: HUB_URL });
    const hubJwt = await graphqlSignIn(hubRequest);
    hubCookie = `trip2g_e2e=${hubJwt}`;

    peer2Request = await playwright.request.newContext({ baseURL: PEER2_URL });
    const peer2Jwt = await graphqlSignIn(peer2Request, 'hello@example.com', '111111', { useCache: false });
    peer2Cookie = `trip2g_e2e_peer2=${peer2Jwt}`;

    peer3Request = await playwright.request.newContext({ baseURL: PEER3_URL });
    const peer3Jwt = await graphqlSignIn(peer3Request, 'hello@example.com', '111111', { useCache: false });
    peer3Cookie = `trip2g_e2e_peer3=${peer3Jwt}`;

    // ── Create inbound secrets on peer2 and peer3 ─────────────────────────────
    await gql(peer2Request, PEER2_URL, peer2Cookie, `
      mutation($input: CreateInboundFederationSecretInput!) {
        admin { createInboundFederationSecret(input: $input) {
          ... on CreateInboundFederationSecretPayload { id kid }
          ... on ErrorPayload { message }
        } }
      }
    `, { input: { kid: KID_P2, secretHex: SECRET_HEX_P2 } });

    await gql(peer3Request, PEER3_URL, peer3Cookie, `
      mutation($input: CreateInboundFederationSecretInput!) {
        admin { createInboundFederationSecret(input: $input) {
          ... on CreateInboundFederationSecretPayload { id kid }
          ... on ErrorPayload { message }
        } }
      }
    `, { input: { kid: KID_P3, secretHex: SECRET_HEX_P3 } });

    // ── Create outbound secrets on hub ────────────────────────────────────────
    const p2Out = await gql(hubRequest, HUB_URL, hubCookie, `
      mutation($input: CreateOutboundFederationSecretInput!) {
        admin { createOutboundFederationSecret(input: $input) {
          ... on CreateOutboundFederationSecretPayload { id kid }
          ... on ErrorPayload { message }
        } }
      }
    `, { input: { kid: KID_P2, secretHex: SECRET_HEX_P2, kbURL: 'http://app-peer2:20093/_system/mcp' } });
    outboundPeer2Id = p2Out.admin.createOutboundFederationSecret.id;
    expect(outboundPeer2Id, 'outbound peer2 secret must be created').toBeTruthy();

    const p3Out = await gql(hubRequest, HUB_URL, hubCookie, `
      mutation($input: CreateOutboundFederationSecretInput!) {
        admin { createOutboundFederationSecret(input: $input) {
          ... on CreateOutboundFederationSecretPayload { id kid }
          ... on ErrorPayload { message }
        } }
      }
    `, { input: { kid: KID_P3, secretHex: SECRET_HEX_P3, kbURL: 'http://app-peer3:20095/_system/mcp' } });
    outboundPeer3Id = p3Out.admin.createOutboundFederationSecret.id;
    expect(outboundPeer3Id, 'outbound peer3 secret must be created').toBeTruthy();

    // ── Find federation-test subgraph ID (auto-created from federation_kb_semi.md frontmatter) ──
    const sgData = await gql(hubRequest, HUB_URL, hubCookie, `
      query { admin { allSubgraphs { nodes { id name } } } }
    `);
    const sg = sgData.admin.allSubgraphs.nodes.find(n => n.name === 'federation-test');
    expect(sg, 'federation-test subgraph must exist — check that hub vault sync ran').toBeTruthy();
    const federationTestSubgraphId = sg.id;

    // ── Create non-admin user with federation-test subgraph access ────────────
    const scopedEmail = `e2e-fed-acl-${crypto.randomBytes(4).toString('hex')}@example.com`;
    const createUserData = await gql(hubRequest, HUB_URL, hubCookie, `
      mutation($input: CreateUserInput!) {
        admin { createUser(input: $input) {
          ... on CreateUserPayload { user { id } }
          ... on ErrorPayload { message }
        } }
      }
    `, { input: { email: scopedEmail } });
    const scopedUserId = createUserData.admin.createUser.user?.id;
    expect(scopedUserId, 'scoped user must be created').toBeTruthy();

    await gql(hubRequest, HUB_URL, hubCookie, `
      mutation($input: CreateUserSubgraphAccessInput!) {
        admin { createUserSubgraphAccess(input: $input) {
          ... on CreateUserSubgraphAccessPayload { accesses { id } }
          ... on ErrorPayload { message }
        } }
      }
    `, { input: { userId: scopedUserId, subgraphIds: [federationTestSubgraphId] } });

    // Allow subgraph access to propagate before creating personal token.
    await new Promise(r => setTimeout(r, 1000));

    // Sign in scoped user (isolated context — must not share admin session).
    scopedCtx = await playwright.request.newContext({ baseURL: HUB_URL });
    const scopedJwt = await graphqlSignIn(scopedCtx, scopedEmail, '111111');
    scopedCookie = `trip2g_e2e=${scopedJwt}`;
    const scopedResult = await createPersonalToken(scopedCtx, HUB_URL, scopedCookie, {
      name: 'e2e-fed-acl-scoped',
    });
    scopedToken = scopedResult.plaintextToken;
    scopedTokenId = scopedResult.id;

    // Admin personal token for authenticated MCP calls.
    const adminResult = await createPersonalToken(hubRequest, HUB_URL, hubCookie, {
      name: 'e2e-fed-acl-admin',
    });
    adminToken = adminResult.plaintextToken;
    adminTokenId = adminResult.id;
  });

  test.afterAll(async () => {
    if (adminTokenId) {
      await revokePersonalToken(hubRequest, HUB_URL, hubCookie, adminTokenId).catch(() => {});
    }
    if (scopedTokenId && scopedCtx) {
      await revokePersonalToken(scopedCtx, HUB_URL, scopedCookie, scopedTokenId).catch(() => {});
    }
    // outboundPeer2Id may already be revoked by test 4.1 — ignore error.
    if (outboundPeer2Id) {
      await gql(hubRequest, HUB_URL, hubCookie, `
        mutation($id: Int64!) { admin { revokeFederationSecret(id: $id) {
          ... on RevokeFederationSecretPayload { revokedId }
          ... on ErrorPayload { message }
        } } }
      `, { id: outboundPeer2Id }).catch(() => {});
    }
    if (outboundPeer3Id) {
      await gql(hubRequest, HUB_URL, hubCookie, `
        mutation($id: Int64!) { admin { revokeFederationSecret(id: $id) {
          ... on RevokeFederationSecretPayload { revokedId }
          ... on ErrorPayload { message }
        } } }
      `, { id: outboundPeer3Id }).catch(() => {});
    }
    await scopedCtx?.dispose();
    await hubRequest?.dispose();
    await peer2Request?.dispose();
    await peer3Request?.dispose();
  });

  // ─── Group 1: search visibility ────────────────────────────────────────────
  // Asserts which KB-notes appear in hub local search results per principal.
  // Assertions use note paths (federation_kb_semi, federation_kb_private) which
  // appear in the formatted search text alongside federation metadata.

  test('1.1 anonymous search: open KB visible, semi-private and admin-only absent', async ({ playwright }) => {
    const result = await mcpFresh(playwright, 'tools/call', {
      name: 'search',
      arguments: { query: 'federation knowledge base' },
    });
    expect(result.error).toBeUndefined();
    const text = result.result?.content?.[0]?.text ?? '';
    expect(text).toContain('federation_kb');
    expect(text).not.toContain('federation_kb_semi');
    expect(text).not.toContain('federation_kb_private');
  });

  test('1.2 scoped user search: open + semi-private visible, admin-only absent', async ({ playwright }) => {
    const result = await mcpFresh(playwright, 'tools/call', {
      name: 'search',
      arguments: { query: 'federation knowledge base' },
    }, scopedToken);
    expect(result.error).toBeUndefined();
    const text = result.result?.content?.[0]?.text ?? '';
    expect(text).toContain('federation_kb');
    expect(text).toContain('federation_kb_semi');
    expect(text).not.toContain('federation_kb_private');
  });

  test('1.3 admin search: all three KB-notes visible', async ({ playwright }) => {
    const result = await mcpFresh(playwright, 'tools/call', {
      name: 'search',
      arguments: { query: 'federation knowledge base' },
    }, adminToken);
    expect(result.error).toBeUndefined();
    const text = result.result?.content?.[0]?.text ?? '';
    expect(text).toContain('federation_kb');
    expect(text).toContain('federation_kb_semi');
    expect(text).toContain('federation_kb_private');
  });

  // ─── Group 2: targeted routing ─────────────────────────────────────────────
  // federated_search with explicit kb_id. Inaccessible KB-notes must return
  // "not configured" — indistinguishable from a kb_id that simply does not exist.

  test('2.1 anonymous → peer2: not configured (existence must not leak)', async ({ playwright }) => {
    const result = await mcpFresh(playwright, 'tools/call', {
      name: 'federated_search',
      arguments: { kb_id: 'peer2', query: 'anything' },
    });
    expect(result.error).toBeUndefined();
    const text = result.result?.content?.[0]?.text ?? '';
    expect(text.toLowerCase()).toMatch(/not.*(found|configured)/);
    // Must not reveal that peer2 is a configured-but-restricted peer.
    expect(text.toLowerCase()).not.toMatch(/access.?denied|unauthorized|forbidden/);
  });

  test('2.2 anonymous → peer3: not configured', async ({ playwright }) => {
    const result = await mcpFresh(playwright, 'tools/call', {
      name: 'federated_search',
      arguments: { kb_id: 'peer3', query: 'anything' },
    });
    expect(result.error).toBeUndefined();
    const text = result.result?.content?.[0]?.text ?? '';
    expect(text.toLowerCase()).toMatch(/not.*(found|configured)/);
    expect(text.toLowerCase()).not.toMatch(/access.?denied|unauthorized|forbidden/);
  });

  test('2.3 scoped user → peer2: returns peer2-note content', async ({ playwright }) => {
    const result = await mcpFresh(playwright, 'tools/call', {
      name: 'federated_search',
      arguments: { kb_id: 'peer2', query: 'federation ACL testing' },
    }, scopedToken);
    expect(result.error).toBeUndefined();
    const text = result.result?.content?.[0]?.text ?? '';
    expect(text).toContain('Semi-private peer2 content');
  });

  test('2.4 scoped user → peer3: not configured (admin-only KB inaccessible)', async ({ playwright }) => {
    const result = await mcpFresh(playwright, 'tools/call', {
      name: 'federated_search',
      arguments: { kb_id: 'peer3', query: 'anything' },
    }, scopedToken);
    expect(result.error).toBeUndefined();
    const text = result.result?.content?.[0]?.text ?? '';
    expect(text.toLowerCase()).toMatch(/not.*(found|configured)/);
  });

  test('2.5 admin → peer2: returns peer2-note content', async ({ playwright }) => {
    const result = await mcpFresh(playwright, 'tools/call', {
      name: 'federated_search',
      arguments: { kb_id: 'peer2', query: 'federation ACL testing' },
    }, adminToken);
    expect(result.error).toBeUndefined();
    const text = result.result?.content?.[0]?.text ?? '';
    expect(text).toContain('Semi-private peer2 content');
  });

  test('2.6 admin → peer3: returns peer3-note content', async ({ playwright }) => {
    const result = await mcpFresh(playwright, 'tools/call', {
      name: 'federated_search',
      arguments: { kb_id: 'peer3', query: 'federation ACL testing' },
    }, adminToken);
    expect(result.error).toBeUndefined();
    const text = result.result?.content?.[0]?.text ?? '';
    expect(text).toContain('Admin-only peer3 content');
  });

  // ─── Group 3: fan-out ──────────────────────────────────────────────────────
  // federated_search without kb_id fans out to ALL accessible KB-notes.
  // accessibleKBNotes filters BEFORE fanout(), so restricted peers receive no
  // HTTP call at all — their content must not appear in results.

  test('3.1 anonymous fan-out: no peer2 or peer3 content', async ({ playwright }) => {
    const result = await mcpFresh(playwright, 'tools/call', {
      name: 'federated_search',
      arguments: { query: 'federation ACL testing' },
    });
    expect(result.error).toBeUndefined();
    const text = result.result?.content?.[0]?.text ?? '';
    expect(text).not.toContain('Semi-private peer2 content');
    expect(text).not.toContain('Admin-only peer3 content');
  });

  test('3.2 scoped user fan-out: peer2 content present, peer3 absent', async ({ playwright }) => {
    const result = await mcpFresh(playwright, 'tools/call', {
      name: 'federated_search',
      arguments: { query: 'federation ACL testing' },
    }, scopedToken);
    expect(result.error).toBeUndefined();
    const text = result.result?.content?.[0]?.text ?? '';
    expect(text).toContain('Semi-private peer2 content');
    expect(text).not.toContain('Admin-only peer3 content');
  });

  test('3.3 admin fan-out: content from peer2 and peer3 both present', async ({ playwright }) => {
    const result = await mcpFresh(playwright, 'tools/call', {
      name: 'federated_search',
      arguments: { query: 'federation ACL testing' },
    }, adminToken);
    expect(result.error).toBeUndefined();
    const text = result.result?.content?.[0]?.text ?? '';
    expect(text).toContain('Semi-private peer2 content');
    expect(text).toContain('Admin-only peer3 content');
  });

  // ─── Group 4: revocation + ACL decoupling ─────────────────────────────────
  // Revoking an outbound secret and hiding a KB-note are independent operations.
  // FederationClient (main.go) refuses to silently downgrade a configured-private
  // peer to anonymous when its outbound secret is revoked — it returns an explicit
  // error ("no active federation secret ... may be revoked").
  // But accessibleKBNotes still returns the KB-note (ACL is unaffected by secret
  // revocation), so search still shows it to authorized callers.

  test('4.1 revoke peer2 outbound secret → federated_search returns revocation error', async ({ playwright }) => {
    const revokeData = await gql(hubRequest, HUB_URL, hubCookie, `
      mutation($id: Int64!) { admin { revokeFederationSecret(id: $id) {
        ... on RevokeFederationSecretPayload { revokedId }
        ... on ErrorPayload { message }
      } } }
    `, { id: outboundPeer2Id });
    expect(revokeData.admin.revokeFederationSecret.revokedId).toBe(outboundPeer2Id);

    const result = await mcpFresh(playwright, 'tools/call', {
      name: 'federated_search',
      arguments: { kb_id: 'peer2', query: 'federation ACL testing' },
    }, adminToken);
    const text = result.result?.content?.[0]?.text ?? result.error?.message ?? '';
    // Hub must not silently downgrade to anonymous — must report revocation.
    expect(text.toLowerCase()).toMatch(/revoked|no active|secret/);
    // "not configured" would mean KB-note is inaccessible — but admin can still see it.
    expect(text.toLowerCase()).not.toContain('federation is not configured for kb_id');
  });

  test('4.2 after revocation: admin search still shows federation_kb_semi (ACL unchanged)', async ({ playwright }) => {
    const result = await mcpFresh(playwright, 'tools/call', {
      name: 'search',
      arguments: { query: 'federation knowledge base' },
    }, adminToken);
    expect(result.error).toBeUndefined();
    const text = result.result?.content?.[0]?.text ?? '';
    // KB-note is still readable — secret revocation does not affect ACL.
    expect(text).toContain('federation_kb_semi');
  });

  test('4.3 after revocation: scoped user search still shows federation_kb_semi', async ({ playwright }) => {
    const result = await mcpFresh(playwright, 'tools/call', {
      name: 'search',
      arguments: { query: 'federation knowledge base' },
    }, scopedToken);
    expect(result.error).toBeUndefined();
    const text = result.result?.content?.[0]?.text ?? '';
    expect(text).toContain('federation_kb_semi');
  });
});
```

- [ ] **Step 2: Run the spec in isolation to verify it parses**

```bash
npx playwright test e2e/federation-acl.spec.js --list
```

Expected output: list of 13 tests, no parse errors.

- [ ] **Step 3: Commit**

```bash
git add e2e/federation-acl.spec.js
git commit -m "test(federation): add ACL spec — 3-tier access control for anonymous/scoped/admin"
```

---

### Task 5: Verify

- [ ] **Step 1: Start the full test stack with peer2 and peer3**

```bash
docker compose -f docker-compose.test.yml up --build -d
```

Expected: 5 containers healthy (minio, embedding, app, app-peer, app-peer2, app-peer3).

```bash
docker compose -f docker-compose.test.yml ps
```

Expected: all 6 services `healthy` or `running`.

- [ ] **Step 2: Run vault and peer syncs manually**

```bash
# From test-e2e.sh flow — sync hub vault first (creates federation-test subgraph)
npx playwright test e2e/setup.spec.js
API_KEY=$(cat .test-api-key)
./scripts/test-sync-cli.sh --api-key "$API_KEY" --endpoint http://localhost:20081/graphql
```

Then run the peer sync functions (copy-paste from `scripts/test-e2e.sh` or run the full script).

- [ ] **Step 3: Run federation-acl tests**

```bash
APP_URL=http://localhost:20081 npx playwright test e2e/federation-acl.spec.js --reporter=line
```

Expected: 13/13 passed.

- [ ] **Step 4: Run full existing federation suite to check no regressions**

```bash
APP_URL=http://localhost:20081 npx playwright test e2e/federation.spec.js e2e/personal-tokens.spec.js --reporter=line
```

Expected: all existing tests still pass.

- [ ] **Step 5: Final commit if any fixes needed**

```bash
git add -p
git commit -m "test(federation): fix <whatever needed>"
```
