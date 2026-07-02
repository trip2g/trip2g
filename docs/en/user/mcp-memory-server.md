---
title: "MCP memory server: persistent, self-hosted memory for AI agents"
description: "What an MCP memory server is, how the main options compare (official server, Mem0, trip2g), and how to run a self-hosted memory your agent and your team can actually read."
free: true
lang_redirect: "[[ru/user/mcp-memory-server]]"
---

Your agent forgets everything when the session ends. An MCP memory server fixes that: a service your agent connects to over the Model Context Protocol, writes facts into, and searches in later sessions. This page explains how the options differ and shows the trip2g route, where memory is plain markdown notes you can open in Obsidian, browse as a website, and version in git.

*Updated: July 2026.*

**Quick verdict:**

- **Fastest local setup**: the official `@modelcontextprotocol/server-memory`. One `npx` command, a knowledge graph in a local file, zero accounts.
- **Automatic memory with no discipline required**: Mem0 / OpenMemory. It extracts facts from conversations by itself.
- **Memory you can read, edit, and publish**: trip2g. Notes are markdown files; the same base serves your agent over MCP and your readers as a website.

## What an MCP memory server is

MCP (Model Context Protocol) is the standard interface between AI clients (Claude Code, Claude Desktop, Cursor, Gemini CLI) and external tools. A memory server is an MCP server whose tools store and retrieve knowledge: the agent writes durable facts during a session and searches them in the next one, instead of rediscovering the project from scratch.

The differences between servers come down to three questions:

1. **What form does the memory take?** A JSON graph, vector embeddings in a database, or human-readable documents.
2. **Who writes it?** Automatic extraction from conversation, or deliberate notes the agent is instructed to write.
3. **Where does it live?** A file on one laptop, a managed cloud, or a server you host.

## The options, honestly compared

| | Official memory server | Mem0 / OpenMemory | trip2g |
|---|---|---|---|
| Memory format | knowledge graph in a local JSONL file | extracted facts + embeddings | markdown notes |
| Human-readable | partly (raw JSON) | via dashboard | yes: Obsidian, web page, git diff |
| Auto-capture from chat | no | **yes** | no: agent writes notes deliberately |
| Setup | **one `npx` line** | pip/Docker or hosted account | one Docker command (memcli) |
| Works across machines | no (local file) | yes (cloud or self-hosted) | yes (server; token-scoped access) |
| Team access control | no | workspace-level | per-note, subscription and subgraph scoped |
| Version history | no | limited | every edit in `note_versions` + git mirror |
| Doubles as a website | no | no | **yes** |
| Vector search | no | yes | yes (hybrid full-text + semantic) |
| Infrastructure | none | vector store (or hosted) | single Go binary + SQLite |

trip2g loses two rows it cares about least: it will not extract memories from your chat automatically, and the official server is faster to install. If you want zero-effort capture on one machine, take one of the first two columns and stop reading.

Pick trip2g when the memory itself matters: you want to audit what the agent remembers, correct it by editing a file, share it with teammates under access control, or publish parts of it.

## Memory as markdown: how the trip2g route works

The design is simple: memory is a folder of `.md` files. A sync watcher pushes every file change to a local trip2g server within about half a second, the server indexes it, and the agent recalls it over MCP with three tools: `search`, `expand`, `note_html`.

Boot it with one command from a repo checkout ([[en/user/memcli|memcli]]):

```bash
git clone https://github.com/trip2g/trip2g
node trip2g/cli/memcli/dist/memcli.js up --folder ./memory-vault
```

When it prints `memory live — web: http://localhost:24081`, you have:

- an MCP endpoint at `http://localhost:24081/_system/mcp`;
- a watched vault folder: any `.md` file the agent writes becomes searchable memory in ~500 ms;
- a web UI on the same port where you read the memory as a website.

Full setup, including the manual Docker path and MCP client registration, is in [[en/user/agent-memory|Long-term memory for AI agents]].

### Verify it works

Ask the endpoint for its tools:

```bash
curl http://localhost:24081/_system/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"tools/list","id":1}'
```

You should see `search`, `similar`, `note_html`, `expand` and the `federated_*` family in the response. Then have the agent call `search("test")` and read one result with `note_html`. If both return content, recall works.

### Why retrieval stays cheap

Recall follows `search → expand → note_html`: search returns a `toc_path` pointing at the exact section, and the agent reads only that section instead of the whole note. On this site's own notes that is about 15× cheaper at the median, up to 37× on long notes. The benchmark is reproducible with a stdlib-only Python script: [[en/user/token-economy-bench|token economy, measured]].

### One memory, many surfaces

Because memory is notes on a server, the same base is:

- **an agent memory** over `/_system/mcp`, for every machine and teammate with a token;
- **a website** humans browse, with per-note access control for private parts;
- **a git repository**: every edit is snapshotted, so memory history is a readable diff;
- **a federation hub**: one query can fan out to peer knowledge bases over [[en/user/federation|MCP federation]].

That last point separates a memory server from a memory silo. `memcli up` writes a `hub.md` note that federates your local memory to trip2g.com's public knowledge base, and the same mechanism links your teammates' hubs.

## Where the other servers are better

Be clear about the tradeoffs before choosing:

- **The official memory server** needs no Docker, no account, no server. For a single machine and modest memory, it is genuinely the simplest option, and it is free.
- **Mem0** captures memories automatically. trip2g memory only grows when the agent (or you) writes a note; that is a feature if you want auditable memory, a chore if you don't.
- Purpose-built memory servers score themselves on recall benchmarks like LongMemEval; trip2g has no such score. Its measurable number is retrieval cost, not recall accuracy.

## FAQ

**Do I need a vector database?**
No. trip2g runs as a single Go binary with SQLite; the vector index is built in. The official server needs nothing at all; Mem0 self-hosted typically wants a vector store.

**Can I read what my agent remembers?**
Yes, three ways: open the `.md` file in Obsidian, open the same note in the browser, or `git log` the mirror. Editing the file edits the memory.

**Does it work with Claude Code / Cursor / Claude Desktop?**
Any MCP client works. For Claude Code specifically, there is a step-by-step tutorial: [[en/user/claude-code-persistent-memory|persistent memory for Claude Code]].

**Is the memory private?**
By default notes are visible only with authentication. A note becomes public only if you add `free: true` to its frontmatter. Access for teammates is scoped by subscription and subgraph; see [[en/user/mcp|the MCP server reference]].

**Can several agents share one memory?**
Yes. The server is the single source; each agent connects with its own token and sees the notes its access allows. Machines don't need to be online at the same time.

**What happens to my memory if I stop using trip2g?**
Nothing dramatic: it is a folder of markdown files plus a git repository. Move the folder, and the memory moves with you.

**How is this different from RAG?**
RAG retrieves chunks into a session and forgets the synthesis. Here the agent writes the synthesis back as notes, so the base compounds across sessions. The pattern is described in [[en/user/llm-wiki|LLM Wiki]].

## Related

- [[en/user/agent-memory|Long-term memory for AI agents]]: the full setup guide
- [[en/user/claude-code-persistent-memory|Persistent memory for Claude Code]]: client-specific tutorial
- [[en/user/trip2g-vs-mem0-vs-memgpt|trip2g vs Mem0 vs MemGPT vs Zep]]: detailed comparison
- [[en/user/obsidian-mcp-server|Obsidian MCP server]]: your vault as the agent's knowledge source
- [[en/user/mcp|MCP server reference]]: all tools, tokens, access control
- [[en/user/federation|MCP federation]]: fan one query across many bases
