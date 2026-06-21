---
title: "How search works"
lang_redirect: "[[ru/user/search]]"
free: true
---

When you type a query, the system runs two independent algorithms simultaneously.

**Text search** finds notes that contain all the words from your query. Russian morphology is handled correctly: "tired" and "tiring" are treated as the same word. Technically this is a BM25 index with a morphological analyzer.

**Semantic search** works differently. Each note is pre-encoded by an OpenAI neural network into a numerical vector — a coordinate in a space of meanings. Your query is encoded the same way, and the system finds notes whose vectors are closest in meaning. This lets you find content where the exact words may not appear, but the topic matches.

The final ranking is built with RRF (Reciprocal Rank Fusion): results from both searches are merged by their positions in each list. This is a standard academic method for hybrid search, robust to differences in scoring scales.

```mermaid
flowchart LR
    Q[Your query] --> T[Text search<br/>BM25 + morphology]
    Q --> S[Semantic search<br/>OpenAI vector, nearest in meaning]
    T --> R[RRF<br/>Reciprocal Rank Fusion<br/>merge by position]
    S --> R
    R --> F[Final ranking]
```

### What this means in practice

- Searching "deploy docs automatically" can surface notes about CI/CD pipelines even if those exact words aren't there
- Short queries with common words still work because text search anchors the results
- Both Russian and English content is indexed; semantic search works across languages

### Excluding notes from search

Some notes are excluded from search automatically:

- Notes whose filename or any folder in the path starts with `_` (for example `_footer.md` or `_layouts/base.md`) are treated as system notes and never appear in search results.

To hide any other note, add `search: false` to its frontmatter:

```yaml
---
search: false
---
```

To exclude a whole section at once, use [[en/user/frontmatter-patches|frontmatter patches]]. For example, to hide all developer docs from search:

```yaml
# frontmatter-patches.yaml
- glob: "dev/**/*.md"
  patch:
    search: false
```

```mermaid
flowchart TD
    N[Note] --> U{Filename or any<br/>path folder starts with _ ?}
    U -->|Yes| EX[Excluded - system note]
    U -->|No| SF{search: false in frontmatter?<br/>direct or via frontmatter patch glob}
    SF -->|Yes| EX
    SF -->|No| IDX[Indexed - appears in search]
```

### MCP server search

The [[en/user/mcp|MCP server]] uses the same semantic search to let AI assistants query your knowledge base. The `search(query)` method runs a vector search and returns the most relevant notes.
