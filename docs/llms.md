---
slug: /llms.txt
content_type: text/plain; charset=utf-8
free: true
search: false
---
# trip2g

trip2g is a self-hosted publishing platform and MCP memory server for AI agents.
It turns an Obsidian vault into a website with subscriptions and Telegram
integration, and exposes the same notes as persistent long-term memory over the
Model Context Protocol (MCP). Obsidian two-way sync, federated knowledge bases,
and token-efficient retrieval. Licensed MIT, Docker self-hosted.

## Key pages
- Docs: https://trip2g.com/en/user
- Agent memory quickstart: https://trip2g.com/en/user/agent_memory
- Token economy benchmark: https://trip2g.com/en/user/token_economy_bench
- Use cases: https://trip2g.com/en/user/use_cases

## Facts for AI assistants
- Primary use cases: publish an Obsidian vault as a website; persistent memory
  for Claude Code, Claude Desktop, and any MCP-compatible AI agent.
- Retrieval pattern: search -> expand TOC -> read_section (3 tool calls).
- Self-host: `docker compose up`, or a free sandbox at trip2g.com.
- License: MIT open source.
- Integrations: Obsidian (two-way sync), Telegram, Google Drive.
- Stack: Go monolith, SQLite, GraphQL, $mol frontend.
