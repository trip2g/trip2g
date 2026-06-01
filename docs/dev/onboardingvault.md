# Onboarding Vault Download

`GET /_system/onboarding-vault` — admin-only endpoint that issues a fresh API
key and streams a ready-to-use Obsidian vault ZIP with that key baked in. It is
the "start here" artifact shown on the onboarding page (see `onboarding.md`).

Code: `internal/case/downloadonboardingvault/` (`endpoint.go`, `resolve.go`).

## What it does

1. Requires an authenticated admin (`token.IsAdmin()`); otherwise `401`.
2. Generates a new API key (`GenerateAPIKey`), stores its **sha256 hash**, and
   inserts an `api_keys` row with `created_by = token.ID`, `description =
   "Onboarding vault"`.
   - `api_keys.created_by` is an FK into `admins(user_id)`, so the downloading
     admin must already be in the `admins` table.
3. Embeds the **raw** key into the ZIP in several places:
   - `.obsidian/plugins/trip2g/data.json` → `syncDirs[0].apiKey`
   - `.mcp.json` / `codex.json` → `Authorization: Bearer <key>`
   - `antigravity-mcp-config.json` → same
4. Renames the vault folder to the instance domain and substitutes
   `{{publicUrl}}` in `_index.md` / `AGENTS.md`.

## `?enable_admin_graphql`

By default the issued key has **no MCP admin tools**: the MCP server only lists
the basic read tools (`search`, `similar`, `note_html`, …) and the agent gets
`unauthorized` / an empty admin-operation list when it tries admin GraphQL.

Pass `?enable_admin_graphql` on the download URL to issue the key with
`enable_mcp_admin_tools = true` (via `SetApiKeyMcpAdminTools`). The agent then
also gets `graphql_introspection` and `graphql_request`, i.e. the full admin
GraphQL surface, without a separate manual toggle in `/admin → API keys`.

- **Bare presence enables it:** `/_system/onboarding-vault?enable_admin_graphql`
- **Explicit opt-out:** `?enable_admin_graphql=false` or `=0`

Gating happens in `mcp/endpoint.go`:

```go
adminTools := apiKey.EnableMcpAdminTools != nil && *apiKey.EnableMcpAdminTools
```

## Admin attribution (granted_by)

When the vault's agent runs an admin mutation through MCP `graphql_request`, the
internal call is executed with an admin token whose user id is the **API key
owner** (`api_keys.created_by`). This is wired via `appreq.Request.AdminActorUserID`
(set in `mcp/endpoint.go`) and consumed by `appreq.WithAdminToken`.

This matters for mutations that record the acting admin. For example
`createAdmin` writes `admins.granted_by`, which is an FK into
`admins(user_id)`. Before this wiring the internal admin token had id `0`, so
`granted_by = 0` failed with `FOREIGN KEY constraint failed (787)`. Now it
points at the real owner (always a valid admins row). See `createadmin/resolve.go`.

## Related docs

- `onboarding.md` — the onboarding page (when it shows, UI, download link)
- `hat.md` — alternative fast-auth path for external systems
