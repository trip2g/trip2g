# LLM Wiki / OKF Research

> **Status:** living document — last updated 2026-06-23
> **Origin:** Andrej Karpathy's "LLM Wiki" gist (April 4, 2026, 5 000+ stars) and its comment thread, plus OKF announcement (June 12, 2026).

---

## 1. OKF (Open Knowledge Format) v0.1

### Who made it and when

OKF v0.1 was published on **June 12–13, 2026** by **Google Cloud** (Data Analytics team). Authors named in the blog post are **Sam McVeety and Amir Hormati**. The "Google" attribution in the Karpathy gist thread is accurate; this is not a community-driven or independent spec — it came directly out of Google Cloud Platform.

The format was explicitly framed as a formalisation of the emergent "LLM wiki" pattern that Karpathy's gist (and Obsidian / CLAUDE.md users) had independently converged on. Google's framing: "OKF standardises what practitioners already discovered."

### Canonical locations

| Resource | URL |
|---|---|
| Official Google Cloud blog post | <https://cloud.google.com/blog/products/data-analytics/how-the-open-knowledge-format-can-improve-data-sharing> |
| Canonical spec (SPEC.md) | <https://github.com/GoogleCloudPlatform/knowledge-catalog/blob/main/okf/SPEC.md> |
| OKF directory (samples, tools) | <https://github.com/GoogleCloudPlatform/knowledge-catalog/tree/main/okf> |
| Pure-Rust implementation (W4G1/okf) | <https://github.com/W4G1/okf> |

License: **Apache 2.0** (confirmed by the Rust derivative work's NOTICE file; matches upstream).

### Data model

#### File structure

```
bundle/
├── index.md          # reserved: directory listing; no frontmatter
├── log.md            # reserved: change history, date-grouped ISO 8601 entries
├── datasets/
│   ├── index.md
│   └── orders_db.md
└── tables/
    ├── orders.md
    └── customers.md
```

Rules:
- One concept = one `.md` file.
- Directory path = concept identity.
- `index.md` in each directory enables progressive disclosure.
- `log.md` tracks chronological history.
- Unknown filenames are tolerated by consumers.

#### YAML frontmatter fields

| Field | Required? | Notes |
|---|---|---|
| `type` | **YES** | Only mandatory field. Short string; producer-defined, not centrally registered. |
| `title` | Recommended | Display name; consumers may derive from filename if absent. |
| `description` | Recommended | Single-sentence summary for previews / search snippets. |
| `resource` | Optional | URI identifying the underlying asset; absent for abstract concepts. |
| `tags` | Optional | YAML list of strings for cross-cutting categorisation. |
| `timestamp` | Optional | ISO 8601 datetime of last meaningful modification. |
| Custom keys | Allowed | Consumers **must** preserve unknown frontmatter keys on round-trip. |

The root `index.md` (and only that file) may carry `okf_version: "0.1"` in its frontmatter to declare the target spec version.

**Example:**
```yaml
---
type: BigQuery Table
title: Orders
description: One row per completed customer order.
resource: https://console.cloud.google.com/bigquery?p=acme&d=sales&t=orders
tags: [sales, revenue]
timestamp: 2026-05-28T14:30:00Z
---
```

#### Relationship / edge model

Links between concepts are **standard Markdown links** (absolute bundle-relative `/path` or relative `../path`). There is **no typed-edge syntax** in the spec — relationship semantics are conveyed entirely by surrounding prose ("FK to [customers](/tables/customers.md) on customer_id"). All links are treated as untyped, directed edges. Broken links are explicitly **tolerated** (they function as "write this later" stubs).

This is the most important "minimally opinionated" choice in the spec: the format pins down the interoperability surface (frontmatter structure + file layout) but leaves ontology, edge types, and content organisation to producers.

#### What "minimally opinionated" means concretely

OKF does **not** prescribe:
- What `type` values exist (producer-defined).
- Any other required frontmatter fields beyond `type`.
- Body section structures or headings.
- Edge/relationship taxonomy.
- Naming conventions beyond the two reserved filenames.

A conforming bundle must only: (1) every non-reserved `.md` file has parseable YAML frontmatter, (2) every frontmatter block has a non-empty `type` field, (3) reserved filenames follow their specified structures when present.

### Versioning and evolution

- Current: **v0.1 (Draft)** — explicitly a starting point.
- Minor version bumps: backward-compatible additions.
- Major version bumps: breaking changes permitted.
- Contribution model: open PRs and issues on GitHub.

### Tooling

| Tool | Language | Notes |
|---|---|---|
| Reference Python implementation | Python | In the `knowledge-catalog` repo; enrichment agent + static HTML graph visualiser. |
| W4G1/okf | Rust (Apache 2.0) | Zero-dependency; parses frontmatter, builds link graph, validates conformance, CLI for validation/visualisation. |
| equationalapplications expo-llm-wiki | TypeScript | `parseOkfBundle` adapter for foreign OKF bundles; adds `okf_type` column + typed `llm_wiki_edges` table on top. |
| Google Cloud Knowledge Catalog | GCP service | Updated to ingest OKF bundles and serve context to agents. |

### Mapping trip2g → OKF (and back)

trip2g already uses a markdown-vault model that aligns closely. The concrete mapping:

| trip2g concept | OKF equivalent | Notes |
|---|---|---|
| Vault directory | OKF bundle root | 1:1 |
| Note file (`*.md`) | Concept document | 1:1 |
| YAML frontmatter | OKF frontmatter | trip2g can add `type:` field to conform; trip2g's existing fields survive (consumers must preserve unknown keys). |
| `[[wikilinks]]` | Untyped directed edges | OKF uses standard Markdown links; `[[wikilink]]` → `[title](/path/to/note.md)` on export. Lazy resolution is OKF-compatible (broken links tolerated). |
| `index.md` / TOC sections | OKF `index.md` + body sections | trip2g's focused-section retrieval maps onto OKF's progressive-disclosure `index.md` intent. |
| Session log entries | OKF `log.md` | trip2g agent status writes could emit to `log.md` format. |
| Federation hub | Multi-bundle composition | OKF does not spec cross-bundle links; trip2g's federation is additive. |

**Export:** Add `type:` to every note's frontmatter (can default to the directory name or a configured mapping). Convert `[[wikilinks]]` to absolute bundle-relative Markdown links. Done — valid OKF bundle.

**Import:** Parse YAML frontmatter, treat `type` as a tag/filter dimension. Resolve Markdown links back to wikilinks lazily. Preserve all unknown frontmatter keys.

---

## 2. Valuable knowledge from the LLM-Wiki thread

### 2.1 Concept identity / taxonomy drift (Synto — kytmanov)

**What it is:** Synto introduces a stable `entity_id` separate from display names. Surface forms change ("qubit" / "Qubit" / "qubits"), but the underlying concept identity does not. Curation commands (`merge`, `split`, `rename`, `unmerge`, `keep`) let an operator restructure the concept graph without breaking links or losing sources.

- `merge LOSER WINNER [--absorb-edits]`: moves sources and edges to winner, retires loser article, keeps loser labels as aliases.
- `split NAME --sense SENSE SOURCE_PATH ...`: partitions a concept's sources across senses; creates a disambiguation page at original name.
- `keep SURFACE ENTITY`: resolves homonym ambiguity, updates state DB.
- Supports `--dry-run` and `synto undo`.
- No vector DB required; works with local LLMs (Ollama / LM Studio).

**Why it matters:** Without stable IDs, rename + merge operations silently break cross-references. At scale (hundreds of concepts) homonym drift is the main cause of inconsistency.

**Source:** <https://github.com/kytmanov/synto>

### 2.2 Synthesis decay / hallucination-as-truth mitigation (Eidetic — LARIkoz; OpenClerk — yazanabuashour)

#### Eidetic drift detection

**What it is:** Eidetic tracks three signals to suppress stale memories at retrieval time (without mutating files):

1. **Wikilink decay** — References to renamed/deleted files are flagged immediately.
2. **Age staleness** — Memories untouched for 30+ days receive a 0.5× ranking multiplier.
3. **Confidence escalation** — Agent-extracted updates without human confirmation receive a 0.3× penalty after 3 events.

Key empirical observation: *"After ~day 60 the real failure mode is confident-but-stale memory making the agent worse, not better."*

Auto-extraction at session end: a small model extracts signals (`Decision:/Rule:/Worked:/Failed:/Knowledge:`) from the transcript. Agent-extracted memories carry a 0.5× weight multiplier to prevent the agent reinforcing its own hallucinations. Human-authored knowledge always outranks automatic extraction.

**Compounding over duplication:** Search-before-write — if an existing memory on the same topic exists, append a dated `## Update` section; otherwise create a new entry. Prevents file-count bloat.

**Source:** <https://github.com/LARIkoz/eidetic>

#### OpenClerk provenance model

**What it is:** Hard boundary between three write authorities:
1. **Canonical markdown** (human-readable authority, immutable by agents without explicit approval).
2. **Derived state** (SQLite / FTS index — projections only, disposable and rebuildable).
3. **Agent write authority** (explicitly scoped; requires provenance).

Every retrieval result carries `doc_id`, `chunk_id`, and citation path. Agents cannot silently rewrite source material; all durable writes go through `compile_synthesis` or explicit repair requests. Read-only report primitives (`planned_no_write: true`) surface relationship graphs and duplicate candidates without mutating anything.

**Source:** <https://github.com/yazanabuashour/openclerk>

### 2.3 Contradiction detection without O(n²) (hmbseaotter)

**What it is:** Three-tier approach that keeps contradiction checks cheap at scale by exploiting the wikilink graph structure.

**Tier 1 — Per-source detection at ingest (high frequency, ~8–15 pages):**
- Compares incoming claim against only the pages touched by this ingest.
- Classifies as: `soft` / `scope-mismatch` / `hard` / `none`.
- Hard contradictions stop ingestion and block the commit.
- Machine-readable severity token in the file: `"Contradiction severity: hard / Status: Unresolved — flagged for user review"`.

**Tier 2 — Deterministic commit gate (zero LLM cost):**
- Python `os.walk` + regex grep for `"Status: Unresolved"` — no context window needed.
- Runs before every commit; scales to any wiki size via cheap disk I/O.

**Tier 3 — Periodic lint backstop (scoped to graph neighbourhood):**
- Deterministic checks (orphan detection, missing-page detection, unreferenced images) via shell/grep — no LLM.
- LLM invoked only for reasoning-heavy cases: contradiction backstop, causal-chain gaps, thin-page judgment.
- Contradiction surface bounded by wikilink graph: only check nodes changed since last pass + their 1st- and 2nd-degree link neighbours. *"Contradiction can only exist between claims about the same entity or relationship"* — the graph hands you the neighbourhood for free.

**The intentional gap:** A contradiction between two old unchanged pages that have never landed in the same lint neighbourhood may slip through until a periodic full sweep. This is an explicit tradeoff: the per-source check and commit gate filter the real-world corruption cases; rare cold contradictions are caught by periodic sweeps.

**Source:** Karpathy gist comment thread — <https://gist.github.com/karpathy/442a6bf555914893e9891c11519de94f>

### 2.4 Boundaries: canonical source / derived index / agent write authority (OpenClerk + distorx)

**OpenClerk** (see §2.2): canonical markdown → derived SQLite → agent write authority, with `doc_id`/`chunk_id` provenance.

**distorx dual-index (production at ~7,500 notes):**

- **FTS5 trigram index** for keyword/substring search: ~0.07 s latency.
- **1024-d vector embeddings** (mxbai-embed-large via Ollama) for semantic search: ~1.1 s cold.
- **Content-hash authority:** embedding and re-index triggered by content hash change, not mtime. Prevents spurious re-embeds on filesystem touch events; prevents vector-space mixing when the embedding model is swapped (embedding-identity guard).
- **MOC hubs** (Map of Content): written once per area via graph analysis; member notes don't churn. Reduced orphan rate from 31% to 19%.

**Source:** Karpathy gist comment thread — distorx comment; <https://gist.github.com/karpathy/442a6bf555914893e9891c11519de94f>

### 2.5 Concurrency-safe writes (Matryca-Plumber — MarcoPorcellato)

**What it is:** Shifts the storage model from flat Markdown text to a **block-based outliner AST** (Logseq format) with **UUID per block**. LLMs target block UUIDs rather than guessing where to inject text. Write safety via **Optimistic Concurrency Control (OCC)**:
- Snapshot `st_mtime` before inference.
- Acquire page lock (`page_rmw_lock` + platform `flock`) only for the write.
- If the file was edited while the model was thinking, the commit aborts: *"your typing always wins."*
- Single shared mutation plane (`graph_dispatch`) — CLI commands, MCP tools, and background daemon all use the same write path, eliminating race conditions between operators.

**Source:** <https://github.com/MarcoPorcellato/matryca-plumber>; <https://github.com/MarcoPorcellato/logseq-matryca-parser>

### 2.6 Memory-as-external-MCP-service positioning (Smriti-MCP — deepak-bhardwaj-ps)

**What it is:** Memory externalised from the agent lifecycle as a standalone MCP server. Multiple agents (Claude, OpenAI, local models) share one wiki by pointing to the same `--memory-root` directory. Memory survives model changes and agent restarts because the storage is plain markdown with YAML frontmatter. Authorship tracked via optional `source_agent` metadata field.

12 MCP tools exposed: `create_memory`, `get_memory`, `append_memory`, `update_memory`, `delete_memory`, `remember`, `consolidate_memory`, `search_memory`, `list_memories`, `archive_memory`, `build_memory_index`, `rebuild_memory`.

**Source:** <https://github.com/deepak-bhardwaj-ps/smriti-mcp>

### 2.7 Typed relationship edges via seeded ontology (equationalapplications)

**What it is:** On top of OKF's untyped edges, three configurable modes for extracting typed relationships:

1. **Strict (Seeded):** Allowed edge types supplied upfront (e.g., `['client', 'employed_by']`). LLM forced to stick to schema — guarantees predictable data structures for hard-coded dashboards. Maps to `okf_type` column + `llm_wiki_edges` table.
2. **Emergent (Autogenerated):** LLM freely invents relationship types, tracked in an `ontology_manifest`. Maximum expressivity, unpredictable schema.
3. **Off (Disabled):** Semantic fact extraction only, no relationship traversal.

A `parseOkfBundle` adapter is in development for consuming foreign OKF bundles and projecting them into this typed-edge model.

**Source:** Karpathy gist comment thread; <https://github.com/equationalapplications/expo-llm-wiki>

---

## 3. Action list for trip2g (ADOPT / ALREADY-COVERED / SKIP)

| # | Idea | Verdict | Rationale |
|---|---|---|---|
| 1 | **OKF export conformance** — add `type:` field to frontmatter, emit standard Markdown links on export | **ADOPT** | One-field addition makes trip2g vaults importable by any OKF-aware tool (distorx, equationalapplications, Knowledge Catalog). Zero runtime cost. |
| 2 | **Eidetic-style drift detection** — staleness ranking suppression (age × confidence multipliers) without mutating files | **ADOPT** | trip2g serves content to agents; stale-but-confident retrieval is the exact failure mode. Suppress stale sections in ranked results, not in storage. The 30-day age threshold is a calibrated starting point. |
| 3 | **hmbseaotter commit gate** — deterministic regex scan for `Status: Unresolved` before agent writes commit | **ADOPT** | Zero LLM cost, catches the real corruption cases. trip2g already has an append model; adding a pre-commit gate is a one-script addition to the write path. |
| 4 | **Content-hash authority for index** (distorx pattern) — trigger re-index on hash change, not mtime | **ADOPT** | Prevents spurious re-embeds on `git checkout` / filesystem events. Critical when trip2g eventually adds a semantic index sidecar. |
| 5 | **Search-before-write / `## Update` section compounding** (Eidetic) — append dated update sections rather than creating new notes | **ADOPT** | Directly prevents file-count bloat. trip2g's agent write API should expose this as the default; new-note creation requires explicit override. |
| 6 | **OpenClerk write-authority boundaries** — canonical markdown / derived index / agent write authority as distinct layers | **ALREADY COVERED** | trip2g already separates the markdown vault (canonical) from the SQLite queue and index (derived). Agent write authority is already scoped. The `doc_id`/`chunk_id` provenance in retrieval responses is worth adding if not present. |
| 7 | **Focused section retrieval** (trip2g's own 15–23× token savings) | **ALREADY COVERED** | This is trip2g's core differentiator — cited approvingly in the thread by distorx. |
| 8 | **Federation hub** (trip2g's multi-vault MCP) | **ALREADY COVERED** | Mentioned in thread as a first-class pattern. OKF does not specify cross-bundle links, so trip2g's approach is additive and standards-compatible. |
| 9 | **Smriti-MCP style shared-memory-as-service** — single MCP server shared by multiple agents | **ALREADY COVERED** | trip2g is already an MCP service. The `source_agent` metadata field on writes is worth adding for auditability. |
| 10 | **Synto stable `entity_id`** — decouple concept identity from display name | **ADOPT (deferred)** | Important at scale (hundreds of concepts). Not urgent for trip2g v1 where `[[wikilinks]]` by filename serves as identity. Plan for v2 when merge/split operations become needed. |
| 11 | **Three-tier seeded ontology** (equationalapplications) — `okf_type` column + `llm_wiki_edges` table | **SKIP** | trip2g's minimally-opinionated stance (lazy wikilinks) is intentional and matches OKF's own philosophy. Typed edges add schema rigidity. Revisit if agents need traversable relationship graphs. |
| 12 | **Block-AST + UUID + OCC** (Matryca-Plumber) — Logseq block model for concurrent writes | **SKIP** | trip2g targets plain Markdown vaults, not Logseq's block format. The concurrency problem is already handled by trip2g's single-process queue model. Flat Markdown with atomic file writes is sufficient at current scope. |
| 13 | **Passive RLHF from UI interactions** (Matryca-Plumber) — transclusion/zoom = reward, scroll-past = penalty | **SKIP** | Interesting idea but requires a UI instrumentation layer trip2g does not have. Revisit if a web UI is added. |
| 14 | **hmbseaotter graph-neighbourhood lint** — periodic LLM contradiction check over changed nodes + 1st/2nd-degree wikilink neighbours | **ADOPT (deferred)** | Powerful for large vaults. Not urgent at current scale. Implement as an optional `trip2g lint` command, not inline on every write. |
| 15 | **Dual FTS5 + vector index** (distorx) — trigram FTS for keyword, semantic embeddings for concept search | **ADOPT (deferred)** | FTS5 first (can be added with zero new deps in SQLite). Vector index is a natural follow-on when semantic retrieval quality matters more than latency. |

---

## Sources

- [LLM Wiki gist — Andrej Karpathy (April 4, 2026)](https://gist.github.com/karpathy/442a6bf555914893e9891c11519de94f)
- [How the Open Knowledge Format can improve data sharing — Google Cloud Blog (June 13, 2026)](https://cloud.google.com/blog/products/data-analytics/how-the-open-knowledge-format-can-improve-data-sharing)
- [OKF SPEC.md — GoogleCloudPlatform/knowledge-catalog](https://github.com/GoogleCloudPlatform/knowledge-catalog/blob/main/okf/SPEC.md)
- [GoogleCloudPlatform/knowledge-catalog — OKF directory](https://github.com/GoogleCloudPlatform/knowledge-catalog/tree/main/okf)
- [W4G1/okf — Pure-Rust OKF implementation (Apache 2.0)](https://github.com/W4G1/okf)
- [kytmanov/synto — Stable entity_id, merge/split/rename](https://github.com/kytmanov/synto)
- [LARIkoz/eidetic — Drift detection, compounding pages](https://github.com/LARIkoz/eidetic)
- [yazanabuashour/openclerk — Canonical/derived/agent write authority](https://github.com/yazanabuashour/openclerk)
- [deepak-bhardwaj-ps/smriti-mcp — Memory as external MCP service](https://github.com/deepak-bhardwaj-ps/smriti-mcp)
- [MarcoPorcellato/matryca-plumber — Block AST + OCC](https://github.com/MarcoPorcellato/matryca-plumber)
- [MarcoPorcellato/logseq-matryca-parser — Logseq block parser](https://github.com/MarcoPorcellato/logseq-matryca-parser)
- [equationalapplications/expo-llm-wiki — Typed edges + OKF adapter](https://github.com/equationalapplications/expo-llm-wiki)
- [Google Just Standardized Karpathy's LLM Wiki Pattern — themenonlab.blog](https://themenonlab.blog/blog/google-okf-open-knowledge-format-karpathy-llm-wiki-standard)
- [OKF explained — WitsCode](https://witscode.com/open-knowledge-format)
- [Google Cloud Introduces OKF — MarkTechPost (June 16, 2026)](https://www.marktechpost.com/2026/06/16/google-cloud-introduces-open-knowledge-format-okf-a-vendor-neutral-markdown-spec-for-giving-ai-agents-curated-context/)
- [What is OKF — GitBook Blog](https://www.gitbook.com/blog/what-is-okf-open-knowledge-format)
