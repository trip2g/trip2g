---
title: Federation E2E Testing
publish: true
---

# Federation E2E Testing

The test stack includes a second trip2g instance (`app-peer`) on port 20091
for MCP federation integration tests.

## Architecture

- **Hub** (port 20081) — main app, loaded with `docs/demo/` content
- **Peer** (port 20091) — second app, loaded with `testdata/seedvault/` content
- Both share the same MinIO and embedding services

## Peer seed notes

| File | Subgraph | Purpose |
|------|----------|---------|
| `testdata/seedvault/team-status.md` | (none) | Public note, searchable via federation |
| `testdata/seedvault/internal-notes.md` | `premium` | Gated note for scope-filtering tests |

## Hub KB-note

`docs/demo/federation_kb.md` declares the peer as a federated knowledge base:

```yaml
mcp_federation_kb_url: http://app-peer:20091/_system/mcp
mcp_federation_kb_id: peer
```

## Quick test (curl)

With the test stack running (`./scripts/test-e2e.sh --manual`):

```bash
# List MCP tools on hub
curl -s http://localhost:20081/_system/mcp \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' | jq '.result.tools[].name'

# Search locally on hub
curl -s http://localhost:20081/_system/mcp \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"search","arguments":{"query":"peer"}}}' | jq

# Federated search (requires federation secrets to be bootstrapped)
curl -s http://localhost:20081/_system/mcp \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"federated_search","arguments":{"kb_id":"peer","query":"team status"}}}' | jq
```

## References

- [MCP Federation design](../dev/mcp_federation.md)
- [Federation spec](../../e2e/federation.spec.js)
- [docker-compose.test.yml](../../docker-compose.test.yml) — `app-peer` service
