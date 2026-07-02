---
title: "Obsidian MCP server: four ways to connect your vault to AI agents"
description: "Compare the ways to give Claude and other agents access to an Obsidian vault over MCP: local REST-API servers, the filesystem server, and a hosted vault that works with your laptop closed."
free: true
lang_redirect: "[[ru/user/obsidian-mcp-server]]"
---

You keep your knowledge in an Obsidian vault, and you want Claude (or Cursor, or any MCP client) to read it, search it, and cite it. There are several MCP servers for this, and they split into two families: local servers that read the vault on your machine, and a hosted vault the agent reaches over the network. The right choice depends on one question: does the agent only work when your laptop is open?

This page surveys the local options fairly, then shows the hosted route trip2g provides, including where it is not the best fit.

*Updated: July 2026.*

**Quick verdict:**

- **One user, one machine, full privacy**: a local MCP server (`mcp-obsidian` or the filesystem server). Nothing leaves your laptop.
- **Vault available to agents 24/7, from any machine, with access control**: a synced trip2g instance. The vault stays in Obsidian; a server copy answers MCP queries.

## The three local ways

### 1. `mcp-obsidian` (Local REST API family)

The most popular approach (the `mcp-obsidian` repo counts 4k+ GitHub stars as of July 2026): MarkusPfundstein's `mcp-obsidian` and similar servers (such as cyanheads' `obsidian-mcp-server`) talk to the **Local REST API** community plugin inside Obsidian. The agent gets real Obsidian semantics: list files, read and patch notes, run search.

Requirements follow from the design: Obsidian must be installed and **running** with the plugin enabled, you configure an API key, and the MCP server runs as a local process on the same machine. Setup takes a few minutes and everything stays local.

### 2. Filesystem MCP server

A vault is a folder of markdown files, so the generic filesystem MCP server (`@modelcontextprotocol/server-filesystem`) pointed at the vault also works. Obsidian does not even need to be running. The tradeoff: the agent sees files, not a knowledge base. No semantic search, no wikilink resolution, no metadata awareness; on a large vault the agent falls back to reading whole files, which is slow and token-hungry.

### 3. Obsidian plugins with built-in MCP

Some community plugins embed an MCP (or similar) endpoint directly in Obsidian. Convenient, same constraint as option 1: the vault is reachable exactly while Obsidian is open on that machine.

All three local ways share the same profile: **fully private, quick to set up, and tied to your running desktop.** For a single user on one machine, that profile is often exactly right, and you should start there.

## The fourth way: a hosted vault

trip2g inverts the topology. Instead of an agent reaching into Obsidian, the vault syncs to a server you host (or a free cloud instance), and the server exposes the MCP endpoint at `/_system/mcp`. Obsidian stays your editor; [[en/user/two-way-sync|two-way sync]] keeps the copies identical in both directions.

What changes compared to the local family:

| | Local MCP servers | trip2g hosted vault |
|---|---|---|
| Works with laptop closed | no | **yes** |
| Obsidian must be running | yes (REST API family) | no |
| Fully offline / nothing leaves the machine | **yes** | no (vault syncs to your server) |
| Setup time | **minutes** | ~10 minutes plus a server |
| Search | plugin search or file grep | hybrid full-text + semantic, section-precise |
| Multiple users / agents | no | yes, per-note access control |
| Access from other machines | no | yes, token-scoped |
| Version history | via your own git | built-in `note_versions` + git mirror |
| Same vault as a website | no | **yes** |
| Cost | free | free (MIT, self-hosted) or free cloud instance |

Two rows matter most. If "fully offline" is your requirement, the local family wins and no hosted option should talk you out of it. If "agent works while my laptop sleeps" is your requirement, only the hosted row delivers it.

### What the agent gets

The MCP endpoint exposes `search`, `expand`, `note_html`, `similar`, plus `federated_*` tools for reaching peer bases ([[en/user/federation|federation]]). Search returns a `toc_path` pointing at the exact section of a note, so the agent reads that section instead of the whole file, about 15× cheaper at the median, measured and reproducible in [[en/user/token-economy-bench|the token benchmark]].

Access is not all-or-nothing: notes without `free: true` require a token, and subscription subgraphs let you expose one folder of the vault to an agent (or a person) without exposing the rest. That is the piece local servers structurally cannot do, and it is what makes a shared team vault workable.

### Set it up

1. Create an instance: self-host with Docker ([[en/user/selfhosted|guide]]), boot a throwaway local one with [[en/user/memcli|memcli]], or use a free cloud instance.
2. Install the trip2g sync plugin in Obsidian, connect it, press Sync. Details in [[en/user/getting-started|getting started]].
3. Add the MCP endpoint to your client:

```json
{
  "mcpServers": {
    "obsidian-vault": {
      "url": "https://your-site.example.com/_system/mcp"
    }
  }
}
```

For private vaults, create a personal token under **User → Tokens** and send it as `Authorization: Bearer t2g_…`. All auth options: [[en/user/mcp|MCP reference]].

### Verify

```bash
curl https://your-site.example.com/_system/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"tools/list","id":1}'
```

The response lists the tools. Then, in your MCP client, ask the agent to `search` for a phrase you know is in the vault and to quote the note it found. If it quotes your text, the loop is closed. You can test against a live instance right now: `https://trip2g.com/_system/mcp` serves this documentation site's own vault.

## FAQ

**Can the agent write to the vault, not just read?**
Yes, two ways: with two-way sync, anything the agent writes into the synced folder lands back in Obsidian; and the GraphQL admin tools over MCP can update notes directly (see [[en/user/agent_admin|agent admin]]).

**Do I have to publish my notes to use this?**
No. A synced vault is private by default; the MCP endpoint requires a token for everything that is not explicitly `free: true`. Publishing is a separate, per-note decision.

**Is my vault stored on someone else's server?**
Only on the server you choose. Self-hosting keeps it on your hardware; the cloud option stores it on that instance. If that is unacceptable, use a local MCP server; that is the honest boundary of this approach.

**What happens if the sync conflicts?**
The plugin tracks changes on both sides and resolves simple conflicts automatically; see [[en/user/two-way-sync|two-way sync]] for the rules.

**Which option is cheapest in tokens?**
The hosted route, and not by a little: section-precise retrieval reads ~200 tokens where a file-reading agent reads thousands. Numbers in [[en/user/token-economy-bench|the benchmark]].

**Can I combine local and hosted?**
Yes, they are not exclusive. Many setups keep a local filesystem server for quick edits and the hosted endpoint for search, teammates, and always-on agents.

## Related

- [[en/user/mcp-memory-server|MCP memory server overview]]: the same server as agent memory
- [[en/user/agent-memory|Long-term memory for AI agents]]: full local setup with memcli
- [[en/user/mcp|MCP reference]]: tools, tokens, entry points, access control
- [[en/user/llm-wiki|LLM Wiki]]: structuring a vault agents can navigate
- [[en/user/two-way-sync|Two-way sync]]: how the vault and server stay identical
