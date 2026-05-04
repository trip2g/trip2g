# Federation ACL Test Design

**Date:** 2026-05-04  
**Status:** Approved for implementation

## Problem

The federation protocol targets thousands of nodes. Any access-control leak in the hub layer means:

1. **Topology enumeration** — anonymous agents can discover private nodes by probing `federated_search(kb_id=X)` and observing whether the response differs between "exists but private" and "doesn't exist at all".
2. **Content leak via fan-out** — if `accessibleKBNotes` filtering happens after the HTTP call instead of before, private content arrives at the hub before being discarded.
3. **Cascade leak** — node A knows about private node B (via KB-note); anonymous agent on A must not infer B's existence.

These are regression guards, not feature tests. A refactor of `accessibleKBNotes`, `findFederationKB`, or `fanout` can silently break these invariants. The tests make breakage visible immediately.

---

## Current state

Two federation nodes:
- **Hub** (port 20081) — `docs/demo/` vault, open to all
- **Peer** (port 20091) — `testdata/seedvault/` vault, `federation_kb.md` is `free: true`

Existing tests cover: tools/list, local search finds KB-note, `federated_search`, unknown kb_id, secret revocation. **No tests for access-controlled KB-notes. No authenticated MCP calls in federation tests.**

---

## New infrastructure

### Two new peer services in docker-compose.test.yml

| Service | Ports | DB | Seed vault |
|---------|-------|----|------------|
| `app-peer2` | 20093 / 20094 | `test-peer2.sqlite3` | `testdata/seedvault2/` |
| `app-peer3` | 20095 / 20096 | `test-peer3.sqlite3` | `testdata/seedvault3/` |

Each peer has its own JWT secret, cookie name, and minio prefix to prevent session bleed. Both share the same embedding and minio services (same pattern as `app-peer`).

### Seed notes (one per peer, free: true, unique strings for assertion)

**testdata/seedvault2/peer2-note.md**
```yaml
---
title: Peer2 Note
free: true
---
Semi-private peer2 content for federation ACL testing.
```

**testdata/seedvault3/peer3-note.md**
```yaml
---
title: Peer3 Note
free: true
---
Admin-only peer3 content for federation ACL testing.
```

### Three KB-notes on hub (docs/demo/)

| File | Frontmatter | Tier | kb_id | Points to |
|------|-------------|------|-------|-----------|
| `federation_kb.md` _(existing)_ | `free: true` | Open | `peer` | peer (20091) |
| `federation_kb_semi.md` _(new)_ | `subgraphs: federation-test` | Semi-private | `peer2` | peer2 (20093) |
| `federation_kb_private.md` _(new)_ | _(none — no free, no subgraphs)_ | Admin-only | `peer3` | peer3 (20095) |

`federation_kb_private.md` has no `free:` and no `subgraphs:` — readable only by admin. This is the strictest tier.

---

## Three principals

| Principal | Auth mechanism | Expected KB visibility |
|-----------|---------------|------------------------|
| **Anonymous** | No auth header | Open KB only (peer) |
| **Scoped user** | Personal token of non-admin user with `federation-test` subgraph | Open + Semi-private (peer, peer2) |
| **Admin** | Personal token of admin user | All three (peer, peer2, peer3) |

### Why non-admin user, not a scoped admin token

Personal tokens inherit the full access of the creating user — there is no per-token subgraph scoping (confirmed in `personal-tokens.spec.js`). The only way to get a token with restricted subgraph access is to create a non-admin user and assign subgraphs via `createUserSubgraphAccess`. This is the same pattern used in tests 8/9 of `personal-tokens.spec.js`.

### Scoped user setup (in beforeAll)

```
1. Create subgraph "federation-test" on hub → get subgraphId
2. Create non-admin user on hub (random email)
3. createUserSubgraphAccess(userId, [subgraphId])
4. graphqlSignIn(user) → isolatedCtx
5. createPersonalToken(user) → scopedToken
```

---

## Federation secrets setup (in beforeAll)

Two new KID pairs — one per new peer:

| KID | Inbound (on peer) | Outbound (on hub) | Points to |
|-----|-------------------|-------------------|-----------|
| `KID_PEER2` | peer2 admin | hub admin | `http://app-peer2:20093/_system/mcp` |
| `KID_PEER3` | peer3 admin | hub admin | `http://app-peer3:20095/_system/mcp` |

Same pattern as existing `KID` in `federation.spec.js`. Secrets are deterministic hex constants for reproducibility.

---

## Test spec: e2e/federation-acl.spec.js

### Setup

```
beforeAll:
  - sign into hub admin (hubRequest)
  - sign into peer2 admin (peer2Request)
  - sign into peer3 admin (peer3Request)
  - create federation-test subgraph on hub → subgraphId
  - create non-admin user on hub → assign federation-test subgraph → scopedToken
  - create admin personal token on hub → adminToken
  - create inbound secrets on peer2 and peer3
  - create outbound secrets on hub for peer2 and peer3
  - store outbound IDs for cleanup

afterAll:
  - revoke all created tokens and secrets
  - dispose all request contexts
```

All MCP calls use `mcpCallFresh` (fresh cookie-free context) so session cookies do not interfere with Bearer auth.

---

### Group 1: search visibility (what KB-notes appear in local search results)

These tests call `search` on the hub and assert which KB-notes are (or are not) present in the result list by note path.

| Test | Principal | Asserts |
|------|-----------|---------|
| 1.1 | Anonymous | sees `federation_kb` (open), does NOT see `federation_kb_semi` or `federation_kb_private` |
| 1.2 | Scoped user | sees `federation_kb` and `federation_kb_semi`, does NOT see `federation_kb_private` |
| 1.3 | Admin | sees all three KB-notes |

**Assertion detail for 1.1 (privacy guarantee):**
```js
expect(text).not.toContain('federation_kb_semi');
expect(text).not.toContain('federation_kb_private');
```
The note path must not appear at all — not even as a "you don't have access" message.

---

### Group 2: targeted routing (federated_search with explicit kb_id)

These tests assert what happens when a specific `kb_id` is requested by each principal.

| Test | Principal | kb_id | Asserts |
|------|-----------|-------|---------|
| 2.1 | Anonymous | `peer2` | response matches `/not.*(found\|configured)/i` |
| 2.2 | Anonymous | `peer3` | response matches `/not.*(found\|configured)/i` |
| 2.3 | Scoped user | `peer2` | returns results containing peer2-note content |
| 2.4 | Scoped user | `peer3` | response matches `/not.*(found\|configured)/i` |
| 2.5 | Admin | `peer2` | returns results containing peer2-note content |
| 2.6 | Admin | `peer3` | returns results containing peer3-note content |

**Critical assertion for 2.1 and 2.2 (existence must not leak):**
```js
// Must be "not configured", NOT "access denied" or "401"
expect(text.toLowerCase()).toMatch(/not.*(found|configured)/);
expect(text.toLowerCase()).not.toMatch(/access.denied|unauthorized|forbidden/);
```
A "permission denied" response reveals the node exists. "Not configured" is the correct response for any kb_id that the caller cannot access — indistinguishable from a kb_id that simply does not exist.

---

### Group 3: fan-out (federated_search without kb_id)

Fan-out must only call peers whose KB-notes are accessible to the caller. This is enforced by `accessibleKBNotes` running before `fanout()` — the restricted peers never receive an HTTP call.

| Test | Principal | Asserts |
|------|-----------|---------|
| 3.1 | Anonymous | result does NOT contain peer2-note or peer3-note unique strings |
| 3.2 | Scoped user | result contains peer2-note content, does NOT contain peer3-note content |
| 3.3 | Admin | result contains peer2-note and peer3-note content |

**Assertion for 3.1 (no content leak via fan-out):**
```js
expect(text).not.toContain('Semi-private peer2 content');
expect(text).not.toContain('Admin-only peer3 content');
```

---

### Group 4: revocation + ACL decoupling

This group tests that revoking an outbound secret and hiding a KB-note are independent operations. Revoking a secret does not affect who can *see* the KB-note in search — it only affects whether federation calls *succeed*.

Setup: start with admin access to peer2 (from Group 2 setup).

| Test | Action | Asserts |
|------|--------|---------|
| 4.1 | Revoke outbound secret for peer2 | `federated_search(kb_id="peer2")` by admin no longer returns peer2-note content |
| 4.2 | After revocation, admin searches hub | `federation_kb_semi` still appears in search results (ACL unchanged) |
| 4.3 | After revocation, scoped user searches hub | `federation_kb_semi` still appears in search results for scoped user |

**Why this matters:** if a developer accidentally couples secret revocation to KB-note visibility (e.g. filtering `accessibleKBNotes` by whether a valid outbound secret exists), tests 4.2 and 4.3 will fail. The two concepts must stay independent.

**Exact behavior after revocation (confirmed from `cmd/server/main.go:FederationClient`):**

`FederationClient` checks `HasFederationSecretForKBURL` — if a secret was ever configured for that URL but is now revoked, it returns an explicit error rather than silently downgrading to anonymous:

```
"no active federation secret for kb_id %q; the configured secret may be revoked"
```

This is intentional: the hub refuses to silently downgrade a configured-private peer to anonymous access. The error is surfaced as an internal error response.

**Test 4.1 assertion:**
```js
// response is an error or isError=true
const text = result.result?.content?.[0]?.text ?? result.error?.message ?? '';
expect(text.toLowerCase()).toMatch(/revoked|no active|secret/);
// NOT "not configured" — that would mean KB-note is inaccessible, but admin can still see it
expect(text.toLowerCase()).not.toMatch(/not.*(found|configured)/);
```

This also tests the important corollary: the "not configured" response and the "revoked secret" error are distinguishable to authorized callers (admin/scoped-user), but both look identical to anonymous callers (who see "not configured" in both cases because the KB-note itself is inaccessible to them).

---

## Security invariants (summary)

These are the properties the test suite guarantees hold after every commit:

1. **No topology leak** — anonymous callers cannot distinguish "private KB-note exists" from "kb_id not configured".
2. **No content leak via search** — restricted KB-notes do not appear in `search` results for unauthorized callers.
3. **No content leak via fan-out** — fan-out skips inaccessible peers before making any HTTP call; their content never reaches the hub response.
4. **Subgraph ACL respected** — a user with `federation-test` subgraph can route through semi-private KB, but not admin-only KB.
5. **Revocation independence** — revoking an outbound secret does not affect KB-note visibility (ACL and connectivity are separate).

---

## Files to create/modify

### New files
- `testdata/seedvault2/peer2-note.md`
- `testdata/seedvault3/peer3-note.md`
- `docs/demo/federation_kb_semi.md`
- `docs/demo/federation_kb_private.md`
- `e2e/federation-acl.spec.js`

### Modified files
- `docker-compose.test.yml` — add `app-peer2` and `app-peer3` services

### Existing files (unchanged)
- `e2e/federation.spec.js` — existing tests remain, no modifications
- `e2e/personal-tokens.spec.js` — existing tests remain, no modifications
- All Go source files — no backend changes needed; the ACL logic already exists

---

## Open questions (resolved)

**Q: Do we need subgraph-scoped personal tokens?**  
A: No. Personal tokens inherit user access. Scoped-user tier = non-admin user + `createUserSubgraphAccess`. Pattern already established in `personal-tokens.spec.js` tests 8/9.

**Q: Does fan-out skip restricted peers before or after HTTP call?**  
A: Before. `accessibleKBNotes` filters to accessible KB-notes; `fanout()` iterates only that filtered list. No HTTP call is made to restricted peers. This is the current behavior and the tests in Group 3 serve as regression guards for it.

**Q: What does revocation do to a private peer call?**  
A: Hub finds no valid outbound secret → calls peer anonymously → peer returns its public layer (peer notes are `free: true` in the test setup). This is not a security problem because the private node's content is protected by the KB-note ACL on the hub, not by the outbound secret alone.
