---
title: "Persistent memory for Claude Code: a self-hosted setup that survives sessions"
description: "Give Claude Code long-term memory it keeps between sessions: a local trip2g server over MCP, memory stored as markdown notes you can read and edit. Setup, verification, FAQ."
free: true
lang_redirect: "[[ru/user/claude-code-persistent-memory]]"
---

Every new Claude Code session starts blank. You explain the project layout again, the decision you made last Tuesday again, the gotcha in the build script again. Two minutes of re-explaining per session turns into hours per month, and the agent still repeats mistakes it already solved once.

This guide fixes that with a self-hosted memory server: at the end you have a local trip2g instance Claude Code talks to over MCP, where every durable fact is a markdown note. Claude writes notes as it works and searches them in later sessions. You can open the same notes in Obsidian or a browser and see exactly what your agent remembers.

*Updated: July 2026.*

## Ways to give Claude Code memory

Be honest about the alternatives first, because for some setups a simpler option is the right call:

| Method | Effort | Memory form | Cross-machine | Best for |
|---|---|---|---|---|
| `CLAUDE.md` (built-in) | none | one markdown file, loaded every session | via git | project conventions, short and stable facts |
| Official `@modelcontextprotocol/server-memory` | 1 minute | knowledge graph in a local JSONL file | no | quick local memory on one machine |
| Mem0 and similar | minutes to hours | auto-extracted facts + embeddings | yes | zero-discipline capture, managed cloud |
| trip2g (this guide) | ~10 minutes | markdown notes on your own server | yes | memory you read, edit, share, and version |

If all you need is "remember the coding style", put it in `CLAUDE.md` and stop here. Claude Code loads it automatically and it costs nothing.

Come back to this guide when memory outgrows one file: when you want search instead of one always-loaded blob, history of what changed, several machines or teammates sharing one memory, and the ability to audit it like normal documents.

## Prerequisites

- Docker running locally
- Node.js (to run memcli)
- Python 3 (for the stdio adapter; standard library only, no pip)
- Claude Code installed

## Step 1. Boot the memory server

One command from a trip2g checkout ([[en/user/memcli|memcli]] ships prebuilt in the repo):

```bash
git clone https://github.com/trip2g/trip2g
node trip2g/cli/memcli/dist/memcli.js up --folder ./memory-vault
```

Wait for:

```
memory live — web: http://localhost:24081  read/write .md in ./memory-vault
```

This started a trip2g server in Docker, minted an admin API key, and launched a sync watcher: any `.md` file written to `./memory-vault` is indexed and searchable within ~500 ms. Secrets are generated once and reused, so `up` is idempotent.

**Checkpoint:** open `http://localhost:24081` in a browser. You should see the (empty) site of your future memory.

## Step 2. Create an access token

The MCP adapter authenticates with a personal token:

1. Open `http://localhost:24081`, sign in, go to **User → Tokens**.
2. Click **Generate token**, name it `claude-code`, copy the `t2g_…` value.

The token shows once. Note it is different from the admin API key memcli minted; the API key drives sync, the personal token drives MCP. That split trips people up, details in [[en/user/mcp|the MCP reference]].

## Step 3. Register the server in Claude Code

Two paths; the adapter path is what we recommend for memory work.

### Path A: stdio adapter (recommended)

The adapter wraps search, TOC navigation, and section reading into one tool, so Claude retrieves the exact section that answers a question instead of whole notes. The script is `docs/en/user/trip2g_mcp_stdio_adapter.py` in the repo you already cloned; it needs only the Python standard library.

Add to your project's `.mcp.json` (or `~/.claude.json` for all projects):

```json
{
  "mcpServers": {
    "trip2g-memory": {
      "command": "python3",
      "args": ["/absolute/path/to/trip2g/docs/en/user/trip2g_mcp_stdio_adapter.py"],
      "env": {
        "TRIP2G_MCP_URL": "http://localhost:24081/_system/mcp",
        "TRIP2G_TOKEN": "t2g_your-token-here"
      }
    }
  }
}
```

### Path B: direct HTTP

Claude Code can also talk to the MCP endpoint directly:

```bash
claude mcp add --transport http trip2g-memory http://localhost:24081/_system/mcp \
  --header "Authorization: Bearer t2g_your-token-here"
```

You get the raw tool set (`search`, `expand`, `note_html`, `similar`) instead of the composite adapter tool. Fine for experimenting; the adapter is leaner on tokens.

## Step 4. Verify

Start a new Claude Code session and run:

```
/mcp
```

`trip2g-memory` should be listed as connected. Then ask Claude:

```
Write a note "memory-test.md" into ./memory-vault saying the deploy user is
"deploy@prod-1". Then use the trip2g-memory tool to search for "deploy user"
and quote what you find.
```

If the search comes back with the fact you just wrote, memory works end to end: file → sync → index → MCP recall. If `/mcp` shows a connection error, check the adapter path is absolute and the token starts with `t2g_`.

## Step 5. Teach Claude to use it

Memory only compounds if the agent writes to it. Add rules to your `CLAUDE.md`:

```markdown
## Memory
- REMEMBER: write durable facts (decisions, gotchas, environment details)
  as markdown files into ./memory-vault. One topic per file.
- RECALL: before re-deriving anything about this project, search
  trip2g-memory first. Read only the section you need.
```

From now on, sessions start with recall instead of re-explanation. The retrieval chain (`search → expand → note_html`) reads one section instead of a whole note, which is roughly 15× cheaper at the median on real notes; measured numbers and a reproducible script are in [[en/user/token-economy-bench|the token economy benchmark]].

## The friction, admitted

- **You run a server.** Docker must be up for memory to be reachable. The official memory server and `CLAUDE.md` have no such dependency.
- **Nothing is captured automatically.** If neither you nor the agent writes a note, nothing is remembered. Tools like Mem0 extract facts from conversation without being asked; trip2g trades that convenience for memory you can audit.
- **Two credentials.** The admin API key (sync) and the personal token (MCP) are different things, and the error messages when you swap them are not great yet.

## Beyond one machine

Everything above runs on localhost, but the server does not have to be local. Point `TRIP2G_MCP_URL` at any trip2g instance and the same memory follows you across machines, with access scoped per token. A team can share one memory base where each member sees the subgraphs their subscription allows. And through [[en/user/federation|federation]], one search can fan out from your memory to peer knowledge bases; `memcli` even wires a default hub note for you.

## FAQ

**How is this different from CLAUDE.md?**
`CLAUDE.md` is one file loaded into every session whole; it costs context every time and does not scale past a few hundred lines. The memory server is searched on demand: hundreds of notes cost nothing until one is actually needed.

**How is this different from Claude Code's auto memory?**
Claude Code's built-in memory keeps its notes per project on one machine. A memory server is a database with search, versions, and access control, shared across machines and agents. They combine well: keep habits in `CLAUDE.md`, knowledge in the server.

**Does Claude write memories on its own?**
Only if instructed (Step 5). This is deliberate: the memory contains what was intentionally recorded, and nothing else. If you want automatic capture, Mem0 is the better tool and we say so above.

**Can I edit what Claude remembers?**
Yes. Memory notes are files in `./memory-vault`; edit them in Obsidian or any editor, and the watcher syncs the change in ~500 ms. Deleting the file deletes the memory.

**What about the history of a memory?**
Every edit is versioned server-side (`note_versions`) and mirrored to git, so you can diff how a memory evolved or recover an overwritten note.

**Does this work with Cursor, Codex, or other agents?**
Yes, anything that speaks MCP: the same adapter config works in Cursor and Claude Desktop. The memory base does not care which agent is talking to it, which also means all your agents share one memory.

**What does it cost?**
The software is MIT-licensed and self-hosted; you pay for whatever machine runs Docker. There is no per-seat or per-memory pricing.

**Is my memory sent anywhere?**
No. The server runs on your machine, and notes leave it only if you explicitly federate or publish them.

## Related

- [[en/user/agent-memory|Long-term memory for AI agents]]: the full reference this tutorial is based on
- [[en/user/mcp-memory-server|MCP memory server overview]]: how the memory-server options compare
- [[en/user/memcli|memcli]]: everything the one-command boot does
- [[en/user/ai-agent-mcp-adapter|The stdio adapter]]: one tool, just the right section
- [[en/user/llm-wiki|LLM Wiki]]: growing the memory into a knowledge base
