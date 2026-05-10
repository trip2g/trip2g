# Agent context

Publishing platform: Obsidian vault → website. Sync via Obsidian plugin or CLI.

## MCP servers

`.mcp.json` / `codex.json` are pre-configured with your credentials.

| Name | Endpoint |
|------|----------|
| `my-trip2g-instance` | `{{publicUrl}}/_system/mcp` — tools for your site |
| `trip2g-docs-public-hub` | `https://trip2g.com/_system/mcp` — platform docs & query examples |

**Antigravity:** copy `antigravity-mcp-config.json` → `~/.gemini/antigravity/mcp_config.json`

## Sync

Two options with the same API token:

- **Obsidian plugin** — built-in, configured automatically
- **CLI** — `node trip2g-sync.mjs` ([download](https://github.com/trip2g/obsidian-sync/releases/download/0.3.7/trip2g-sync.mjs))

## Enable admin GraphQL

To manage the site via agent, enable **"Execute admin GraphQL"** for your API key:

[Open API key settings]({{publicUrl}}/admin#!nav=integrations/integrations_nav=apikeys/id=key1) → Enable Admin GraphQL

Or manually: `{{publicUrl}}/admin` → Integrations → API Keys → your key → Enable Admin GraphQL

After enabling, an introspection tool becomes available in `my-trip2g-instance`.
Use `trip2g-docs-public-hub` for query examples or run introspection directly.
