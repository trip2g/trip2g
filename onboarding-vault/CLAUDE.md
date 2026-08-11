---
subgraph: private
---

# trip2g vault

This vault's agent guide lives in `AGENTS.md`, at the vault root. Read it first — it covers the publishing model, the sync CLI, git access, and the MCP tools configured in `.mcp.json`.

Publish everything you changed:

```bash
node .obsidian/plugins/trip2g/trip2g-sync.mjs --folder .
```

Run it from the vault root — credentials are read automatically from `.obsidian/plugins/trip2g/data.json`.
