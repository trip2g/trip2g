---
title: MCP Server
free: true
lang_redirect: "[[ru/user/mcp]]"
---

The MCP server turns your knowledge base into an AI consultant. Connect it to any MCP-compatible client — Claude Desktop, Claude Code, Cursor, GitHub Copilot, Gemini CLI — and chat with your knowledge base directly.

### How it works

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│  AI client      │────▶│   MCP server    │────▶│  Knowledge base │
│                 │◀────│   trip2g.com    │◀────│  (your notes)   │
└─────────────────┘     └─────────────────┘     └─────────────────┘
     Question            Vector search           Notes + instructions
     Answer              Relevant notes
```

1. Ask a question in the chat
2. The MCP server finds relevant notes in the knowledge base
3. Returns note text and author instructions
4. The AI client composes an answer grounded in your knowledge

### Methods

| Method | Description |
|--------|-------------|
| `search(query)` | Vector search across the knowledge base |
| `note_html(path)` | Full content of a specific note |
| `similar(path)` | Notes similar to the given note |
| `instructions()` | Author-defined AI instructions |
| `editor_role()` | Answer style instructions |

### Setting up your own MCP knowledge base

#### Step 1. Publish your notes

Move notes to Obsidian and publish via trip2g. The service automatically builds a vector index for search.

#### Step 2. Add AI instructions

Create a note with instructions for the AI and add to its frontmatter:

```yaml
---
mcp_method: instructions
---
```

Example instructions:

```markdown
# Role
You are a virtual assistant powered by a personal knowledge base.
Your answers MUST be grounded in the knowledge base content.

## Workflow
1. search(query) → find relevant notes
2. Pick the 3 most relevant notes
3. Ask a clarifying question to confirm direction
4. Load full content with note_html(path)
5. Synthesize an answer through the lens of these notes
6. Cite sources with links
```

#### Step 3. Configure access

Enable the MCP server in site settings. Access can be:
- **Open** — for everyone
- **Subscription-only** — for paying subscribers only

### Use cases

**Expert knowledge consultant** — an expert built a knowledge base on a topic. You connect their MCP and get a consultant that answers in the expert's style, citing their materials.

**Selling access to knowledge** — authors build bases with instructions and sell subscriptions. Subscribers get current knowledge, prompt updates, and a consultant in their own chat. Authors get recurring income and motivation to keep the base current.

### Privacy

- The MCP server returns only text from the knowledge base
- The server receives search queries but does not see your chat context, replies, or files
- Instructions run locally on your client, not on the knowledge base server
- No request history is stored
