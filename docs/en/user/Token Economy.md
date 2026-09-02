---
title: "Token Economy: pull the exact section, not the whole file"
free: true
lang_redirect: "[[ru/user/Token Economy]]"
---

An AI agent that dumps whole notes into context wastes tokens on text it will never use. trip2g gives agents two tools to pull only what they need: structural section retrieval via `toc_path`, and GraphQL field selection that returns exactly the fields you request — nothing more.

This note covers both tools and explains why this approach is the right one.

---

### The problem with `grep -A/-B` and whole-file dumps

`grep -A 20 -B 20 "worker pool" note.md` returns a fixed 40-line window around each match. That window doesn't know about section boundaries. It may cut off mid-sentence, include unrelated paragraphs from an adjacent section, and return three overlapping windows for a term that appears three times. The agent pays for all of it.

Loading the entire note is worse. A 3000-token note contains one relevant paragraph. The agent reads all 3000 tokens to find it — and then the answer lands near the middle of the context window, where model recall is weakest.

**Anti-example:** agent calls `search("worker pool")`, gets a match, calls `note_html(pid=42)` with no `toc_path`. Loads the full note. Pays 3000 tokens, model recalls poorly.

**Example:** agent calls `search("worker pool")`, reads `toc_path: ["Goroutines", "Worker pool"]` from the result, calls `note_html(pid=42, toc_path=["Goroutines", "Worker pool"])`. Gets one section, ~300 tokens.

---

### Precise section retrieval

[[Fuzzy Pointer]] covers the full mechanism. The short version:

1. `search(query)` returns results. Each match carries `matches[].toc_path` — the path to the matching section. Results are slim: no full heading hierarchy, structure unfolds on demand via [[expand]].
2. `note_html(pid=N, toc_path=match.toc_path)` fetches only that section's HTML.

The token contrast: one section is ~300 tokens. The full note is ~3000 tokens. If you need a sibling section, call `expand(pid=N)` — it lists the note's sections one level at a time, so you pay for one TOC level, not the whole tree.

Read [[Fuzzy Pointer]] for the complete drill-down, including what `pid` is and how `toc_path` resolves when a section has no heading.

---

### GraphQL field selection

The MCP `graphql_request` tool runs a read-only GraphQL query against the vault. The result arrives in the MCP result's `structuredContent` field — the `{data, ...}` envelope. The `content[0].text` block is a short stub string ("structured result"); the actual payload is in `structuredContent`. Read `structuredContent`, not the stub.

GraphQL lets you request exactly the fields you need. Compare:

**Anti-example:** an agent greps search results or fetches whole note objects. It gets `body`, `frontmatter`, `raw_markdown`, `created_at`, `updated_at`, `tags` — all the fields that exist, including the ones it doesn't need. Every unused field is tokens paid for nothing.

**Example:**

```graphql
{ search(input: { query: "worker pool" }) {
    nodes {
      highlightedTitle
      url
    }
  }
}
```

This returns only `highlightedTitle` and `url` per result, in `structuredContent.data.search.nodes`. When you need more context, add fields:

```graphql
{ search(input: { query: "worker pool" }) {
    nodes {
      highlightedTitle
      url
      document { path title }
    }
  }
}
```

Add `document { path title }` when you need to navigate to the note. Omit it when you don't. The payload appears once — in `structuredContent` — not duplicated in the text block. That's by design: single-copy, no redundancy.

Field selection is composable. You pay only for what you ask for.

---

### Why this matters

This is not just an efficiency trick. Three pieces of research ground it:

Anthropic's guide [Effective context engineering for AI agents](https://www.anthropic.com/engineering/effective-context-engineering-for-ai-agents) (2025) describes loading context "just in time" via lightweight identifiers rather than pre-loading whole documents. The goal: the "smallest possible set of high-signal tokens." The guide also names *context rot* — a model's recall degrades as the context window fills with low-relevance material.

Anthropic's [Contextual Retrieval](https://www.anthropic.com/engineering/contextual-retrieval) (Sept 2024) shows that prepending chunk-specific context before embedding — so a chunk is not orphaned from its document hierarchy — dramatically improves retrieval accuracy. trip2g's heading-breadcrumb-per-chunk design implements exactly this: each indexed chunk carries its full section path, so the vector model sees `Go Concurrency Patterns > Goroutines > Worker pool`, not just the paragraph text.

Liu et al., ["Lost in the Middle"](https://arxiv.org/abs/2307.03172) (TACL 2023) measured that models recall content at the beginning and end of a context window far better than content buried in the middle. Dumping a whole file puts the answer in the middle. Pulling just the relevant section puts it at the top.

`toc_path` and GraphQL field selection are trip2g's implementation of the "just in time, smallest high-signal set" principle.

---

### Recommended agent workflow

```
1. search(query)
   → results with matches[].toc_path

2. For content: read toc_path on the best match
   → note_html(pid=N, toc_path=match.toc_path)
   → returns only that section

3. For metadata: graphql_request with only the fields needed
   → result in structuredContent.data

4. If you need a sibling section,
   expand(pid=N, toc_path=[...]) lists that level's sections
   → pick one; expand on a section without subsections returns
     its content, same as note_html(toc_path=[...])
```

Full MCP tool reference: [[mcp|MCP server]]. Section navigation details: [[Fuzzy Pointer]]. Level-by-level TOC walking: [[expand]].
