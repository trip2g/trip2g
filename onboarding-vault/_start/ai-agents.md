---
free: true
title: AI agents and this vault
---

This vault ships ready for coding agents. Open the folder in Claude Code, Codex, or any MCP client and it can already search your published notes, read them, and publish changes.

### What is already wired

| File | For |
|------|-----|
| `AGENTS.md` | The agent's own briefing — publishing model, sync commands, git round-trip, MCP tools |
| `CLAUDE.md` | Points Claude Code at `AGENTS.md` |
| `.mcp.json` | Claude Code MCP servers, with your instance URL and key filled in |
| `codex.json` | The same for Codex |
| `antigravity-mcp-config.json` | The same for Antigravity — copy it to `~/.gemini/antigravity/mcp_config.json` |

Two MCP servers are configured. `my-trip2g-instance` does semantic search over **your** notes. `trip2g-docs-public-hub` is the platform documentation — an agent can look up how any trip2g feature works without you pasting docs at it.

> [!warning] `.mcp.json` and `data.json` hold a live API key
> The key has full write access to your site. Do not commit these files to a public repo, do not share the downloaded archive. If one leaks, revoke it in the admin panel under **API Keys** and download a fresh vault.

### Try it

Open this folder in Claude Code and ask:

```
Search my notes for anything about deployment, then write a summary note and publish it.
```

It will use `search` to find material, write a `.md` file here, and run the bundled sync script to push it live.

### Sync from a terminal

The plugin's sync button has a command-line twin. From the vault root:

```bash
node .obsidian/plugins/trip2g/trip2g-sync.mjs --folder .
```

It reads the URL and key from the plugin settings, so no flags are needed. Add `--dry-run` to preview, `--two-way` to also pull server-side changes. This is what CI pipelines and agents use.

### Your notes as agent memory

Because search is semantic and every note is addressable by section, a trip2g instance works well as long-term memory for an agent — it retrieves the paragraph that answers the question rather than the whole note. `AGENTS.md` explains the token-efficient retrieval pattern to the agent directly.

You can also make the site itself an assistant for your readers: a note with `mcp_method: instructions` in its frontmatter tells any connecting AI client what role to play and how to answer from your content.

Deeper: [AI assistant (MCP)](https://trip2g.com/en/user/mcp) · [Agent memory](https://trip2g.com/en/user/agent-memory)

### Next

[[on-your-phone]] — the same vault, on iPhone or Android.
