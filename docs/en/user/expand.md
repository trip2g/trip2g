---
title: "expand: level-by-level TOC navigation"
free: true
lang_redirect: "[[ru/user/expand]]"
---

`expand` is an MCP tool that walks a note's table of contents **one level at a time** (progressive disclosure). An agent descends the section tree from the top and reads only the leaf it needs — without loading the full note or its entire flat table of contents.

Why it matters: this is token economy in practice. Reading one section instead of a full note is **orders of magnitude cheaper** (a reproducible benchmark is at [[en/user/token-economy-bench]]), and the answer stays at the top of the context where model recall is strong, not buried in the long tail where it degrades. Background: [[Token Economy]].

### How it works

`expand` returns the **direct children** of one TOC node:

- Omit `toc_path` (or pass `[]`) — get the **top-level sections**.
- Pass a node's `toc_path` — get its **subsections**.

Each child carries: `title`, `level`, `path` (the breadcrumb to pass on the next call), and `has_children` (whether it has further nesting).

What comes back depends on the section named by `toc_path`:

- A section **with subsections** returns the list of its children, as above.
- A section **without subsections** returns its content — the same text `note_html(path, toc_path)` would give, plus `section_html` in the structured payload. A client does not need a second call to read a leaf: the descent ends in the read.
- There is **no flag** to turn this off, and a client that only wants structure never needs one: the parent's listing already marks every child with `has_children`, so stop descending at `has_children: false` and nothing is read.

**Arguments.** One note identifier (`pid`, `note_id`, `path`, or `href` — taken from `search` results) plus an optional `toc_path`:

```json
{ "name": "expand", "arguments": { "pid": 42, "toc_path": ["Goroutines"] } }
```

The response is a structured list of children plus a short text summary ("N subsection(s)"); for a leaf it is the section content instead.

### Workflow

```
1. search(query)
   → results with snippet, breadcrumb, and matches[].toc_path

2. expand(pid=N)                    # top-level TOC
   expand(pid=N, toc_path=[...])   # drill into the right branch
   → repeat, picking the child by meaning; a leaf comes back in full

3. note_html(pid=N, toc_path=[...])
   → read a section whose path you already have, without descending
```

A shorter path: if `search` already returned an exact `matches[].toc_path`, read the section directly via `note_html(toc_path=...)`. Use `expand` when you want to **survey the structure** and navigate deeper without loading anything extra.

### Slim search

`search` results are **slim**: they don't carry a full flat `toc`. Structure unfolds on demand via `expand`, so you don't pay tokens for the whole table of contents — you take exactly the level you need.

### expand vs. alternatives

- **vs. reading the full note.** A leaf section costs far fewer tokens than the full note; the gain scales with note size (numbers at [[en/user/token-economy-bench|benchmark]]). The answer stays at the top of the context rather than in the "dead zone" at the bottom.
- **vs. `grep -A -B`.** `grep` gives ±N lines around a **lexical** match; `expand` returns a whole **section by heading structure**, and locates it **by meaning** (via vector `search`) rather than by string. More importantly, `expand` works **over MCP against a knowledge base that `grep` cannot reach at all**: hosted, shared with a team, remote, or someone else's. For a single person with a local vault, `grep` is perfectly adequate; `expand` is for when the base is large, shared, or remote.

### Access control

`expand` respects per-note permissions: an agent sees structure only for notes it is allowed to read. It cannot obtain a table of contents for notes outside its access scope.

### Federation

`federated_expand` does the same across **connected** knowledge bases — navigate the structure of a remote base through a single MCP endpoint. See [[en/user/federation|MCP Federation]].

### Try it yourself

No dependencies, Python 3 only:

```sh
python3 scripts/expand_check.py
```

The script hits `https://trip2g.com/_system/mcp`, navigates the TOC via `expand`, reads the target section, and prints a table comparing "navigation + read" against "full note".

### Related

- [[en/user/mcp|MCP Server]] — all methods and access
- [[en/user/federation|MCP Federation]] — federated_expand across connected bases
- [[en/user/token-economy-bench|Token economy benchmark]] — numbers and reproducible measurements
- [[Token Economy]] — why focused reading matters
- [[Fuzzy Pointer]] — how the breadcrumb locates a section
