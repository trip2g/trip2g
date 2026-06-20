---
title: "Fuzzy Pointer: how search finds the exact section"
free: true
lang_redirect: "[[ru/user/Fuzzy Pointer]]"
---

Vector search does not return the exact place in a document — it returns a *fuzzy pointer*: a heading breadcrumb that says "the answer is somewhere around here." This is intentional. The pointer is precise enough to identify the right section, and the system can resolve it to the exact paragraph on demand.

This note explains what a fuzzy pointer looks like, how to resolve it to a section, and how to use it in an AI agent workflow.

---

### What a fuzzy pointer is

When trip2g indexes a note, it splits the note into chunks. Each chunk gets a heading breadcrumb prepended:

```
{title} > {h1} > {h2}

...body of the section...
```

For example, a note titled *Go Concurrency Patterns*, with a top-level section *Goroutines* and a subsection *Worker pool*, produces a chunk that starts:

```
Go Concurrency Patterns > Goroutines > Worker pool

A worker pool is a set of goroutines waiting for tasks...
```

That breadcrumb is embedded together with the body text, so the vector model sees the full section path, not just the paragraph in isolation. When a user searches "pool of workers Go", the vector model finds this chunk — and the search result shows the breadcrumb in the snippet.

The breadcrumb is the fuzzy pointer. It tells you *which note* and *which section* matched, but not which exact line or paragraph.

---

### The three-step drill-down

**Step 1 — fuzzy match.** Vector search returns the top results. Each result includes:
- the note's full table of contents (`toc` field — a structured list of every heading with its `path` array)
- the breadcrumb path of the section that matched (`matches[].toc_path` — the innermost heading containing the matching chunk)

**Step 2 — locate via TOC.** The `toc_path` from the match pinpoints the section in the note's structure. For the example above it would be:

```json
["Goroutines", "Worker pool"]
```

**Step 3 — fetch the section.** Pass that path to `note_html` to retrieve only that section's HTML, not the full note:

```
note_html(pid=42, toc_path=["Goroutines", "Worker pool"])
```

The response contains just the *Worker pool* section — a few hundred tokens instead of the entire note.

---

### Concrete example

A user asks an AI agent: "How do you limit concurrency in Go?"

1. `search("limit concurrency Go")` returns a match inside *Go Concurrency Patterns*. The match carries `toc_path: ["Goroutines", "Worker pool"]`.
2. The agent reads: the relevant section is *Worker pool* under *Goroutines*.
3. `note_html(pid=42, toc_path=["Goroutines", "Worker pool"])` returns the section HTML.
4. The agent answers from that text, citing the note.

Total tokens consumed: the search result summary + one section (~300 tokens) instead of the full note (~3 000 tokens).

---

### Anti-example: what not to do

An agent calls `search`, gets a result, and immediately calls `note_html(pid=42)` — *without* a `toc_path`. It loads the entire note to find one paragraph.

This works, but it costs 10× more tokens and pushes the actual answer near the bottom of a long context window, which hurts model recall. The `toc_path` from the search result already tells you exactly where to look — use it.

---

### Why the pointer is fuzzy, not exact

The breadcrumb identifies the *section*, not the *sentence*. This is a deliberate trade-off:

- **Sentence-level pointers** are unstable: any edit to the note shifts line numbers, and stored pointers go stale.
- **Section-level pointers** are stable: as long as the heading exists, the path resolves correctly even after the section body is rewritten.
- The breadcrumb also travels with the chunk during embedding, so the vector model understands the section's position in the document hierarchy — not just the words in the paragraph.

---

### How AI agents use this

The recommended agent workflow for a trip2g knowledge base:

```
1. search(query)
   → results with toc + matches[].toc_path

2. Read toc_path on the best match
   → identifies which section to load

3. note_html(pid=N, toc_path=match.toc_path)
   → returns only that section

4. If you need a sibling section,
   use toc items from the same search result
   → navigate without a second search call
```

This workflow is described in full in [[mcp|MCP server]].

---

### Limitations

- `pid` is the note's stable PathID returned in search results (distinct from `note_id`). It is the identifier you pass to `note_html`.
- The breadcrumb path must match the note's current headings. If a heading was renamed after indexing, the path may not resolve until the next re-index.
- If a section has no heading (body text between two headings), `toc_path` points to the nearest parent heading, not the paragraph itself.
- Notes with no rendered heading sections at all produce a `toc_path` of `[]` (empty). This is distinct from single-chunk notes: a short note that has at least one heading still gets a non-empty path. The empty path means the note has no `data-header` sections, so the caller should read the whole note.
