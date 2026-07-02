---
mcp_method: instructions
mcp_description: Full tool reference for the trip2g documentation base
free: true
---

# trip2g docs — MCP tool reference

This base is the trip2g documentation: user guides under `en/user/` and `ru/user/`, changelog, developer docs under `dev/`, design plans under `plans/`. The `demo/` folder is a Meditations showcase, not trip2g docs.

Core loop: `search` → read one section with `note_html(toc_path=...)` → navigate structure with `expand` when the pointer misses. Details per tool below.

## search

`search(query, limit?, detail_limit?)` — hybrid (keyword + vector) search over this base.

- `limit`: max results, default 6, cap 20.
- `detail_limit`: how many results include `matches[]` snippets, default 3; the rest are lightweight previews (title, path, score).
- Each result: `title`, `note_id`, `note_path`, `href`, `url`, `kind`, `score`.
- Each match: `match_id`, `chunk_index`, `snippet`, `toc_path` — the breadcrumb of the section that matched, e.g. `["Adding a private peer (two-step exchange)"]`.
- `note_id` is the stable id; `note_html`, `expand`, and `similar` accept it as either `pid` or `note_id`.
- A result with `kind: "federation_kb"` is a pointer to a connected base — switch to `federated_search` with the `kb_id` it names.

Query tips: use content words from the question ("private federation peer HMAC secret"), not full sentences. If nothing relevant comes back, rephrase once with synonyms before reaching for `federated_search`.

## note_html

`note_html(pid | note_id | path | href, toc_path?, match_id?)` — read a note or one section of it.

- With `toc_path`: returns only that section's HTML. This is the default way to read.
- With `match_id` (form `pN:cM` only): returns a focused text window around that chunk. Ids in the `pN:mM` form do not resolve — use the `toc_path` instead.
- With neither: returns the whole note. Do this only for notes `expand` shows to be sectionless.
- Failure mode: a `toc_path` that matches no heading silently falls back to the whole note. A response far longer than one section means the pointer missed — recover via `expand`, do not retry blind.

## expand

`expand(pid | note_id | path | href, toc_path?)` — progressive disclosure of a note's table of contents.

- No `toc_path` (or `[]`): the top-level sections.
- With `toc_path`: that section's direct subsections.
- Each child: `title`, `level`, `path` (ready to pass on), `has_children`.
- Descend until a leaf (`has_children: false`), then `note_html(toc_path=child.path)`.
- Use it to survey a note's structure before reading, and to recover from a `toc_path` miss.

## similar

`similar(pid | note_id | path | href, limit?)` — related notes for a known note (vector similarity). Default limit 10.

Trigger: you have one good note and want its neighbors — related guides, the other-language twin, follow-up material.

## Federated tools

This base can be connected to peer knowledge bases. The federated variants mirror the local ones, with a `kb_id` targeting a peer:

- `federated_search(query, kb_id?, kb_ids?, limit?, detail_limit?)` — omit `kb_id` to fan out across all connected bases in parallel; pass `kb_id` for one base, `kb_ids` for a chosen set.
- `federated_note_html(kb_id, ...)`, `federated_expand(kb_id, ...)`, `federated_similar(kb_id, ...)` — same arguments as their local twins, `kb_id` required.

Trigger: local `search` came up empty after one rephrase, or a local result had `kind: "federation_kb"`. Results from a peer name their `kb_id` — keep passing it on every follow-up call.

## Efficiency rules

1. Section reads first: `search` → `note_html(toc_path=...)`. One section is ~300 tokens; a full note is 3,000+.
2. `toc_path` values are exact heading titles. Copy, never guess or translate.
3. Whole-note reads are the last resort, not the fallback.
4. Cite the `url` of every note you answer from.
