---
title: "trip2g and the Open Knowledge Format (OKF)"
free: true
lang_redirect: "[[ru/user/okf]]"
---

The Open Knowledge Format (OKF) is a portable standard for knowledge bases that an LLM reads and maintains. Google Cloud published v0.1 in June 2026 under Apache 2.0. It takes the LLM-wiki pattern people were already building by hand and pins down just enough structure for one agent's knowledge base to be readable by another tool.

A trip2g vault is already most of the way there. This page shows the mapping and what to set so your vault is a valid OKF bundle, which keeps your agent memory portable instead of locked to one product.

## What OKF specifies

OKF is deliberately thin. It fixes the parts that matter for interoperability and leaves the rest to you.

- One required frontmatter field: `type`. It labels what a note is (a concept, a source, a decision), as a plain string you choose.
- Five recommended fields: `title`, `description`, `resource`, `tags`, `timestamp`.
- Two reserved filenames: `index.md` (what this base covers) and `log.md` (running synthesis and open questions).
- Plain Markdown links are the edges. They are directed and untyped; the meaning of a link lives in the prose around it, not in a separate schema.

That is the whole interoperability surface. OKF calls itself "minimally opinionated": it does not dictate an ontology, a body structure, or a fixed list of types. Typed edges and seeded ontologies that some tools add sit on top of OKF, they are not part of it.

## Why a trip2g vault is already an OKF bundle

An OKF bundle is just a folder of Markdown files with frontmatter and an `index.md`. That is exactly what a trip2g vault is.

| OKF | trip2g |
|---|---|
| Folder of Markdown + frontmatter | The vault you sync |
| `index.md` / `log.md` | Already the two anchor files in the [[en/user/llm-wiki|LLM Wiki]] starter |
| Markdown links as edges | Your `[[wikilinks]]`, resolved to links when published |
| `type` frontmatter | Add one line (see below) |
| Tolerates missing or broken links | Lazy wikilink resolution: a dangling `[[link]]` is a "write this later" marker, not an error |

## Make your vault OKF-valid

Two steps.

**1. Add `type:` to each note's frontmatter.** Pick your own vocabulary: `concept`, `source`, `decision`, `query`, or whatever your `SCHEMA.md` already describes.

```yaml
---
title: "Idempotent ingestion"
type: concept
tags: [ingestion, exactly-once]
timestamp: 2026-06-23
---
```

**2. Keep `index.md` and `log.md` at the root.** The [[en/user/llm-wiki|LLM Wiki]] starter structure already does this.

Your `[[wikilinks]]` stay as they are. trip2g resolves them to links when it renders the page, and the vault folder of Markdown notes is itself the bundle. Nothing changes about how you write.

## Why this matters

Portability. Your agent's long-term memory is a folder you own, in a format another OKF tool can read, not rows in a database you can only reach through one product. You can move it, fork it, hand it to a teammate, or point a different agent at it.

It also composes with [[en/user/federation|federation]]: separate OKF bases can reference each other, and an agent can fan out across them through a single MCP endpoint. OKF makes the bases interchangeable; trip2g serves and federates them live.

## Related

- [[en/user/llm-wiki|LLM Wiki]]: the pattern OKF formalizes, and the starter folder structure
- [[en/user/agent-memory|Agent memory]]: run a trip2g vault as your agent's long-term memory
- [[en/user/federation|Federation]]: link multiple knowledge bases into one hub
- [[en/user/mcp|MCP server]]: methods, access control, and tokens
