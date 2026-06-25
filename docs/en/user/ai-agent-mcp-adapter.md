---
title: "Connect your AI agent: one tool, just the right section"
free: true
lang_redirect: "[[ru/user/ai-agent-mcp-adapter]]"
---

Give your AI agent your knowledge base — and it reads only the section that answers the question, not the whole note. A few hundred tokens instead of a few thousand, with the answer at the top of the context window where the model reads it best.

trip2g speaks [[mcp|MCP]] (Model Context Protocol). This small adapter wraps the three steps — search, navigate the table of contents, read the section — into a **single tool** your agent calls. It never dumps a whole note or greps blindly.

### What you get

One Python script. **No dependencies, no `pip install`** — your agent launches it directly. Download it from GitHub:

[trip2g_mcp_stdio_adapter.py](https://github.com/trip2g/trip2g/blob/main/docs/en/user/trip2g_mcp_stdio_adapter.py)

### Set it up

1. Download the script above.
2. Add it to your MCP client (Claude Desktop, Cursor, Claude Code, or any agent that speaks MCP over stdio):

```json
{
  "mcpServers": {
    "trip2g": {
      "command": "python3",
      "args": ["/absolute/path/to/trip2g_mcp_stdio_adapter.py"],
      "env": { "TRIP2G_MCP_URL": "https://YOUR-SITE/_system/mcp" }
    }
  }
}
```

3. For private or subscriber-only content, add `"TRIP2G_TOKEN": "t2g_…"` to `env` (get a token in your account → Tokens).

That's it. Your agent now has one tool — *search and read the most relevant section* — with a description telling it exactly when to use it. Ask a question, get back the exact section, cheaply.

Why this beats dumping whole notes: [[Token Economy]].
