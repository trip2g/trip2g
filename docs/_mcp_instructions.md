---
mcp_method: instructions
mcp_description: Full tool reference for the trip2g documentation base
free: true
---

# trip2g docs — MCP tool reference

This base is the trip2g documentation: user guides under `en/user/` and `ru/user/`, changelog, developer docs under `dev/`, design plans under `plans/`. The `demo/` folder is a Meditations showcase, not trip2g docs.

Core loop: `search` → read one section with `note_html(toc_path=...)` → navigate structure with `expand` when the pointer misses. `expand` on a section without subsections returns that section in full, so a descent ends in the read. Details per tool below.

## search

`search(query, limit?, detail_limit?)` — hybrid (keyword + vector) search over this base.

- `limit`: max results, default 6, cap 20.
- `detail_limit`: how many results include `matches[]` snippets, default 3; the rest are lightweight previews (title, path, score).
- Each result: `title`, `note_id`, `note_path`, `href`, `url`, `kind`, `score`.
- Each match: `match_id`, `chunk_index`, `snippet`, `toc_path` — the breadcrumb of the section that matched, e.g. `["Adding a private peer (two-step exchange)"]`.
- `note_id` (integer) and `note_path` (string) both identify the note; `note_html`, `expand`, and `similar` accept the id as either `pid` or `note_id`. Prefer copying `note_path` into their `path` argument — it is the more common, self-describing choice.
- A result with `kind: "federation_kb"` is a pointer to a connected base; its text line names the `kb_id` to send: `kind: federation_kb · kb_id: <id> → federated_search(kb_id="<id>")`. Copy it verbatim — it is already in your frame, even for a base reached through another hub.

Query tips: use content words from the question ("private federation peer HMAC secret"), not full sentences. If nothing relevant comes back, rephrase once with synonyms before reaching for `federated_search`.

## note_html

`note_html(pid | note_id | path | href, toc_path?, match_id?)` — read a note or one section of it.

- With `toc_path`: returns only that section's HTML. This is the default way to read.
- With `match_id` (form `pN:cM` only): returns a focused text window around that chunk. Ids in the `pN:mM` form do not resolve — use the `toc_path` instead.
- With both: `toc_path` wins. Drop a `match_id` copied from an earlier call when you navigate by section, or every read returns the same chunk.
- With neither: returns the whole note. Do this only for notes `expand` shows to be sectionless.
- Failure mode: a `toc_path` that matches no heading returns an error listing the top-level sections. Recover via `expand`, do not retry blind.
- A note that is a federation pointer starts with `federation pointer · kb_id: <id> → federated_search(kb_id="<id>")` — that is the base to switch to.

## expand

`expand(pid | note_id | path | href, toc_path?)` — progressive disclosure of a note's table of contents.

- No `toc_path` (or `[]`): the top-level sections.
- With `toc_path` naming a section that has subsections: its direct subsections. Each child: `title`, `level`, `path` (ready to pass on), `has_children`.
- With `toc_path` naming a section that has no subsections: the section's content — the same text `note_html(path, toc_path)` returns, plus `section_html` in the payload. No second call is needed to read a leaf.
- There is no flag to turn that off, and you never need one: the parent's listing already marks each child with `has_children`, so a client that only wants structure stops at `has_children: false` instead of expanding further.
- Descend by `path` until the answer is content. `note_html(toc_path=...)` is for a section whose path you already have.
- Use it to survey a note's structure before reading, and to recover from a `toc_path` miss.

## similar

`similar(pid | note_id | path | href, limit?)` — related notes for a known note (vector similarity). Default limit 10.

Trigger: you have one good note and want its neighbors — related guides, the other-language twin, follow-up material.

## Federated tools

This base can be connected to peer knowledge bases. The federated variants mirror the local ones, with a `kb_id` targeting a peer:

- `federated_search(query, kb_id?, kb_ids?, limit?, detail_limit?)` — omit `kb_id` to fan out across all connected bases in parallel; pass `kb_id` for one base, `kb_ids` for a chosen set.
- Nested bases: a peer's own peers are addressed with `/` — `kb_id="philosophers/nietzsche"` routes through the `philosophers` peer into the base it federates (recursive; `kb_ids` accepts the same form), up to 3 levels deep by default (kb_id path segments; the `mcp-federation-max-depth` setting) — a deeper path is rejected. A base reached _through_ a hub is addressed `<hub>/<base>`: a `philosophers` hub note advertising `kb_id: montaigne` is `philosophers/montaigne` from an outer hub, not `montaigne`.
- `federated_note_html(kb_id, ...)`, `federated_expand(kb_id, ...)`, `federated_similar(kb_id, ...)` — same arguments as their local twins, `kb_id` required.

Trigger: local `search` came up empty after one rephrase, or a local result had `kind: "federation_kb"`. Every search result now carries an absolute `kb_id` in the caller's frame — use it verbatim to open or re-search that result; keep passing it on every follow-up call.

**Discovering connected bases.** For a browsable directory of the peers instead of blind fan-out, read the hub index: `search("hub")` or `note_html(path="en/hub/_index.md")`. It links one note per base (e.g. `en/hub/foragent.md`); opening one with `note_html` prints its `kb_id` on the first line, and it shows up as a `kind: "federation_kb"` result when searched for. Use that to pick a target for `federated_search(kb_id="foragent")` deliberately. If you do not know a base's `kb_id` at all, call `federated_search` without `kb_id` to fan out: the `[kb_id]` headers in the response name the connected bases, then target one. A not-configured error lists the bases that do exist at the point where your `kb_id` stopped resolving — pick from that list rather than resending the same id.

## Efficiency rules

1. Section reads first: `search` → `note_html(toc_path=...)`. One section is ~300 tokens; a full note is 3,000+.
2. `toc_path` values are exact heading titles. Copy, never guess or translate.
3. Whole-note reads are the last resort, not the fallback.
4. Cite the `url` of every note you answer from.
