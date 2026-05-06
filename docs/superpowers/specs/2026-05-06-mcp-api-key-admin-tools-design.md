# MCP: API Key Auth + Admin GraphQL Tools

**Date:** 2026-05-06  
**Status:** Approved

## Problem

MCP currently supports two auth methods: personal tokens (`t2g_*`) and federation JWT. API keys (used by the Obsidian sync plugin) are not accepted. This means an agent that receives a pre-configured vault zip with an API key cannot use MCP without a separate personal token. Additionally, there is no way for an agent to call admin GraphQL operations through MCP.

## Goal

1. Accept API keys in the MCP endpoint.
2. Add opt-in `graphql_introspection` and `graphql_request` MCP tools, gated behind a per-key flag.
3. Document the agent-admin workflow in `docs/{en,ru}/user/agent_admin.md`.

## Design

### Auth flow (mcp/endpoint.go)

New detection order:

```
1. Personal token — Bearer t2g_* or ?token=t2g_*  →  usertoken.Data in ctx
2. X-API-Key header                                →  db.ApiKey in ctx (new ctx value)
3. Bearer without t2g_ prefix                      →  federation JWT (unchanged)
```

For API key resolution:
- Copy hashing + lookup logic from `checkapikey` (sha256 → hex, fallback to plaintext for old keys).
- Log via `InsertAPIKeyLog` with action `"mcp"`.
- Add `contextWithAPIKey(ctx, *db.ApiKey)` / `apiKeyFromContext(ctx)` — same pattern as `contextWithFederationAuth`.
- Content access level: admin (all notes and subgraphs visible).

`mcp.Env` gains: `ApiKeyByValue`, `InsertAPIKeyLog`, `UpsertAPIKeyLogAction`, `UpsertAPIKeyLogIP`.

### DB migration

```sql
ALTER TABLE api_keys ADD COLUMN enable_mcp_admin_tools boolean
```

Default is NULL (false). No NOT NULL constraint — consistent with SQLite nullable boolean pattern in this codebase.

### GraphQL tools

Both tools appear in `tools/list` only when `apiKeyFromContext(ctx).EnableMcpAdminTools == true`. Not available for personal token sessions (out of scope for this iteration).

**`graphql_introspection(pattern string)`**

Executes a full `__schema` introspection, then filters:
- Keep types and root operations (query/mutation fields) whose name matches `pattern` (treated as regexp, fallback to substring match).
- For each matched item, include all transitively referenced types (input types, output types, field types).
- Returns filtered schema as JSON — grep-like: matched nodes + their context.

**`graphql_request(query string, variables? object)`**

Pass-through GraphQL execution:
- Runs `query` with optional `variables` through the internal GraphQL handler.
- Executes as admin (full access, no whitelist).
- Returns raw JSON response.

`mcp.Env` gains: `GraphQLIntrospect(ctx context.Context) ([]byte, error)` and `GraphQLRequest(ctx context.Context, query string, variables map[string]any) ([]byte, error)`.

### Admin UI

Add `enableMcpAdminTools` field to the `ApiKey` GraphQL type. Add a mutation to toggle it (reuse or extend existing API key update mutation). The checkbox appears in the API keys admin page.

### Documentation

**`docs/{en,ru}/user/mcp.md`** — new section "API key authentication" alongside the existing "Personal access tokens" section. Covers:
- `X-API-Key: <key>` header usage
- How to enable admin tools (checkbox in API keys settings)
- Two new tools in the Methods table: `graphql_introspection(pattern)` and `graphql_request(query, variables?)` with descriptions of what they do and a link to `agent_admin.md` for full context

**`docs/en/user/agent_admin.md`** + **`docs/ru/user/agent_admin.md`** — new files describing the agent-admin concept:
- An agent receives a pre-configured Obsidian vault zip with the sync plugin already set up with an API key.
- The same key authenticates to MCP — no extra setup needed.
- With `enable_mcp_admin_tools` enabled, the agent can call `graphql_introspection` to find relevant operations and `graphql_request` to execute them.
- Example scenario: applying frontmatter patches, managing notes, configuring webhooks — without the agent needing to know the admin UI structure upfront.
- Future: GraphQL schema will be enriched with comments and examples so `search` can find usage examples and the agent can act on them.

## Out of scope

- GraphQL schema comments and usage examples (separate task).
- `graphql_request` access for personal token users.
- Rate limiting or per-operation whitelisting for `graphql_request`.
