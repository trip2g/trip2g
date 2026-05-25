---
title: GraphQL API
free: true
lang_redirect: "[[ru/user/graphql]]"
---

You want to automate admin operations, recover lost content, or drive the admin panel from an agent — without opening a browser. The GraphQL endpoint at `/_system/graphql` is how you do it.

It serves both a GraphiQL browser UI (useful for exploring the schema interactively) and the raw API (used by the Obsidian sync plugin and admin panel). Regular users don't need this — it's for agents and developers.

### Authentication

Three methods work:

| Method | How |
|--------|-----|
| Personal access token | `Authorization: Bearer t2g_…` header, or `?token=t2g_…` query param |
| API key | `X-API-Key: <key>` header |
| Browser session | Automatic — open `/_system/graphql` while logged in as admin |

Personal access tokens are created in **User → Tokens**. API keys are created in **Admin → API Keys** (the same key the Obsidian sync plugin uses).

### GraphiQL

Open `/_system/graphql` in a browser while logged in as admin. GraphiQL picks up your session automatically — no token setup needed. Use it to explore the schema, run queries interactively, and prototype operations before encoding them into agent code.

### Admin access via API key

An API key gives admin-level content access by default. To also call admin mutations — creating webhooks, applying frontmatter patches, reading note versions — enable **MCP admin tools** on the key in **Admin → API Keys**.

With that flag on, the key can execute any query or mutation. Treat it like a root password.

This flag is also what enables the `graphql_introspection` and `graphql_request` tools in MCP. See [[en/user/agent_admin]] for the full setup and [[en/user/mcp]] for the MCP tool reference.

Direct admin GraphQL access without MCP — running the admin panel headlessly via the API — is planned and in progress.

### Example: recover an overwritten note

The version history queries are a concrete example of what admin GraphQL access enables. See [[en/user/version_requests]] for the full walkthrough.

### Security

- Revoke compromised keys in **Admin → API Keys**, tokens in **User → Tokens**
- MCP admin tools are off by default. Enabling them is a deliberate choice
- API key auth bypasses per-user subgraph restrictions — it sees all notes
