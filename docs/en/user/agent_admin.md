---
title: Agent Admin Access
free: true
lang_redirect: "[[ru/user/agent_admin]]"
---

trip2g can be fully driven by an AI agent using a single API key. The same key the Obsidian sync plugin uses to push notes can also authenticate MCP and call admin GraphQL operations — no browser login needed.

### How it works

```
Obsidian vault zip          API key (already inside)
      │                              │
      ▼                              ▼
Agent opens vault     ──────▶   MCP endpoint
                                     │
                               Content access
                               (search, notes)
                                     │
                         enable_mcp_admin_tools?
                                     │
                                     ▼
                           graphql_introspection
                           graphql_request
                                     │
                              Admin mutations
                           (webhooks, patches, etc.)
```

The agent only needs the vault zip. The API key is already inside the sync plugin config; no extra setup steps for the user.

### Setup

1. Create an API key in **Admin → API Keys**
2. Distribute the vault zip with the key already configured in the sync plugin
3. Enable **MCP admin tools** on that key if the agent needs to call mutations

### Available admin tools

When `enable_mcp_admin_tools` is on, two extra MCP tools appear:

| Tool | What it does |
|------|-------------|
| `graphql_introspection(pattern)` | Find GraphQL operations matching a keyword (regexp). Returns matched types + every type they reference — like `grep -A -B` on the schema. |
| `graphql_request(query, variables?)` | Execute any query or mutation as admin. Full access. |

Agents typically use them in pairs: first introspect to discover the right operation and its inputs, then request to execute it.

### Example: apply a frontmatter patch

```
Agent: graphql_introspection({ pattern: "frontmatter" })
→ Returns FrontmatterPatch types and createFrontmatterPatch mutation
  with all input fields and types they reference.

Agent: graphql_request({
  query: "mutation Create($input: CreateFrontmatterPatchInput!) {
    adminMutation { createFrontmatterPatch(input: $input) { ... on CreateFrontmatterPatchPayload { patch { id } } } }
  }",
  variables: { input: { ... } }
})
→ Patch applied.
```

### Example: configure a webhook

```
Agent: graphql_introspection({ pattern: "webhook" })
→ Returns ChangeWebhook, createWebhook mutation with input shape.

Agent: graphql_request({
  query: "mutation { adminMutation { createWebhook(input: {...}) { ... } } }"
})
→ Webhook created.
```

### Security

- API key auth gives admin-level content access (all notes and subgraphs)
- `graphql_request` with admin tools enabled can run any mutation — treat the key like a root password
- Revoke compromised keys in **Admin → API Keys**
- Admin tools are off by default. Turning them on is a deliberate choice.
