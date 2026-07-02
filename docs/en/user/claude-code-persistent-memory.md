---
title: "Persistent memory for Claude Code: a self-hosted setup that survives sessions"
description: "Claude Code has native memory (CLAUDE.md + auto memory) that is machine-local. This guide covers native memory first, then a self-hosted trip2g server for memory shared across machines and a team, over MCP."
free: true
lang_redirect: "[[ru/user/claude-code-persistent-memory]]"
---

Every new Claude Code session starts blank. You explain the project layout again, the decision you made last Tuesday again, the gotcha in the build script again. Two minutes of re-explaining per session turns into hours per month, and the agent still repeats mistakes it already solved once.

The honest first answer surprises most people: **Claude Code already has memory built in.** Before reaching for any server or plugin, you should know what native memory does, and the one thing it does not do. This guide covers native memory first, then shows the self-hosted trip2g route for the case native memory cannot cover: memory that is durable, shared across machines, and shared with a team.

*Updated: July 2026.*

## Start here: Claude Code's native memory

Claude Code has two native memory layers, and for a solo developer on one machine they are often enough.

**`CLAUDE.md` (you write it).** A markdown file in your project (or `~/.claude/CLAUDE.md` for global rules) loaded into every session. This is where your instructions live: coding style, commit conventions, commands to run. You maintain it; it travels with the repo through git.

**Auto memory (Claude writes it).** Since late 2025, Claude Code also keeps its own notebook. As it works it saves build commands, debugging insights, and architecture notes to `~/.claude/projects/<encoded-path>/memory/MEMORY.md`, and injects that file into the system prompt at the start of every session. `MEMORY.md` stays a concise index under 200 lines (that is the load cap at startup); when notes pile up, Claude spills detail into topic files like `debugging.md` next to it. It is on by default; run `/memory` to browse or toggle it. You can also scope rules to paths with `.claude/rules/*.md`.

**Be clear about what native memory does not do**, because this is exactly the gap the tools below fill:

- **It is machine-local.** Auto memory lives under `~/.claude/` on one computer and never touches git. Switch laptops and your agent's accumulated notes do not come with you.
- **It is not shared with a team.** There is no built-in way for a teammate's Claude to read what yours learned.
- **It is best-effort.** Anthropic's own docs note there is no guarantee of strict compliance with what is written; Claude decides what is worth saving and may not reload it perfectly.
- **It does not scale by search.** `MEMORY.md` is loaded whole up to the cap, not queried; past a couple hundred lines the tail is not guaranteed in context.

If you work solo on one machine, enable auto memory, keep your conventions in `CLAUDE.md`, and you may not need anything else. Local plugins like MemPalace (MIT, on-device, roughly a millisecond-scale query) or claude-mem (AGPL-3.0, hooks plus SQLite/FTS5) add search on top while staying single-machine.

## When you need more than native memory

Reach past native memory when one of these is true:

- your memory must **follow you across machines** (desktop, laptop, CI);
- a **team** should share one memory the agent reads;
- you want memory that is **durable and versioned**, auditable like documents, and recoverable;
- you want the same memory **queryable over MCP and browsable as a website**.

That is where a self-hosted server earns its place. The rest of this guide sets up trip2g for exactly that: memory as markdown notes on a server you control, shared across machines and teammates, read by the agent over MCP.

## The options at a glance

| Method | Effort | Memory form | Cross-machine / team | Best for |
|---|---|---|---|---|
| `CLAUDE.md` (built-in) | none | one markdown file, loaded every session | via git | your instructions and conventions |
| Auto memory (built-in) | none (on by default) | `MEMORY.md` + topic files, machine-local | no | zero-effort per-machine learning |
| MemPalace / claude-mem | minutes | local store (MIT / AGPL-3.0) | no | on-device search, still one machine |
| Official `@modelcontextprotocol/server-memory` | 1 minute | knowledge graph in a local JSONL file | no | quick local graph memory |
| Mem0 | minutes to hours | auto-extracted facts + embeddings | yes (cloud or self-host) | zero-discipline capture, managed cloud |
| trip2g (this guide) | ~10 minutes | markdown notes on your own server | **yes** | shared, durable, published memory you can read and edit |

trip2g's honest lane is the last row's difference: not "another local memory," but memory that is shared and publishable. For a single machine, native auto memory or a local plugin is often the right, cheaper call, and the concession is real.

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

For a stronger, cross-session probe (the test that actually proves persistence): tell Claude to remember something specific, then close the session entirely and open a fresh one:

```
Session 1: "Remember that this project deploys with `make ship`, not `make deploy`.
            Write it to memory."
(close Claude Code, reopen in the same project)
Session 2: "How do I deploy this project?"
```

If Session 2 answers `make ship` without you re-explaining, memory survived a real session boundary. Run the same probe on a second machine (with `TRIP2G_MCP_URL` pointed at a shared instance) and the answer still holds, which is the thing native auto memory cannot do.

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
- **Nothing is captured into the shared base automatically.** Native auto memory does save notes on its own, but only to the local machine. For memory to reach the shared, cross-machine base, you (or the agent, once instructed) write a note. Mem0 extracts facts from conversation without being asked; trip2g trades that convenience for memory you can audit and share.
- **Two credentials.** The admin API key (sync) and the personal token (MCP) are different things, and the error messages when you swap them are not great yet.

## Beyond one machine

Everything above runs on localhost, but the server does not have to be local. Point `TRIP2G_MCP_URL` at any trip2g instance and the same memory follows you across machines, with access scoped per token. A team can share one memory base where each member sees the subgraphs their subscription allows. And through [[en/user/federation|federation]], one search can fan out from your memory to peer knowledge bases; `memcli` even wires a default hub note for you.

## FAQ

**How is this different from CLAUDE.md?**
`CLAUDE.md` is one file loaded into every session whole; it costs context every time and does not scale past a few hundred lines. The memory server is searched on demand: hundreds of notes cost nothing until one is actually needed.

**How is this different from Claude Code's auto memory?**
Auto memory (`~/.claude/projects/<hash>/memory/MEMORY.md`) is machine-local, best-effort, and loaded whole up to a ~200-line cap; it is not shared with other machines or teammates and is not searched. A memory server is a database with search, versions, and access control, shared across machines and agents. They combine well: let auto memory handle per-machine habits, and use the server for durable, shared knowledge.

**If Claude Code already has memory, why run a server at all?**
For a solo developer on one machine, you often should not; enable auto memory and keep conventions in `CLAUDE.md`. Run a server when memory must cross machines, be shared with a team, be versioned and auditable, or double as a browsable website. Those four are the whole reason this page exists.

**Does Claude write memories on its own?**
Into native auto memory, yes, on the local machine. Into the shared trip2g base, only if instructed (Step 5), which is deliberate: the shared memory contains what was intentionally recorded. If you want automatic extraction from conversation, Mem0 is the better tool and we say so above.

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
