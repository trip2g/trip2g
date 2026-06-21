# Token-economy benchmark — toc/section retrieval cost (TASK)

**Status:** TODO / spec. Not yet implemented.

## Why

`docs/en/user/Token Economy.md` and `docs/en/user/Fuzzy Pointer.md` make concrete
token-cost claims that **no benchmark backs**:

- "one section is ~300 tokens. The full note is ~3000 tokens" → ~10× saving
  (`Token Economy.md:21`, `Fuzzy Pointer.md:68`).
- loading the whole note "costs 10× more tokens and pushes the actual answer near the
  bottom of a long context window" (`Fuzzy Pointer.md:76`).
- GraphQL field selection saves tokens by dropping unused fields
  (`Token Economy.md:38-73`).

Retrieval **quality** is already measured — `docs/dev/retrieval_eval.md` and
`docs/en/thoughts/search-benchmark.md` (the heading breadcrumb / F4 feature lifts
long-doc nDCG +0.023, en-en +0.092). What's missing is retrieval **cost**: the actual
token count of the drill-down patterns. This task fills that gap and replaces the
hand-waved "~300 / ~3000" numbers in the user docs with measured ones.

**Keep it standalone.** This is a single-instance cost benchmark. Do **not** mix it
with federation or the whoami mesh (`docs/dev/whoami_test.md`) — `federated_note_html`
takes `match_id`, not `toc_path` (`internal/model/federation.go:54`), so the federated
cost variant is a separate follow-up.

## What to measure

A corpus of large multi-section notes + a query set with a known answer-section per
query. For each query, measure the tokens an agent would consume to reach the answer
under each retrieval pattern (all read the same hit):

| Arm | Calls | Tokens counted |
|-----|-------|----------------|
| `full` (anti-pattern) | `search` → `note_html(pid)` | search summary + whole note |
| `toc_path` (fuzzy pointer) | `search` → `note_html(pid, toc_path=section)` | summary + one section |
| `match_id` (focused window) | `search` → `note_html(pid, match_id)` | summary + chunk window |
| `gql_all` | GraphQL note query, all fields | body+frontmatter+raw_markdown+timestamps+tags |
| `gql_min` | GraphQL note query, minimal fields | e.g. `highlightedTitle`+`url` only |

Per arm: token count (use a fixed tokenizer — e.g. tiktoken `cl100k`; **record which**,
since absolute numbers depend on it, ratios less so). Headline outputs:
`ratio_section = section/full`, `ratio_window = window/full`, `ratio_gql = gql_min/gql_all`.
Primary goal: confirm or correct the ~300/~3000 ≈ 10× claim.

Also log a correctness guard per arm (does the returned text actually contain the
answer token?) so a cheap arm that drops the answer isn't scored as a win.

## Setup

- Single trip2g instance. **Vector ON via the mock embedding server**
  (`scripts/mock-embedding-server.mjs`): the real chunk pipeline (`internal/mdchunk`,
  `chunkTargetTokens=450`) runs and produces precise chunk breadcrumbs, so `toc_path`
  resolves to the section — but the mock returns instant 1024-dim vectors, so **no
  2.3 GiB model, no ~300s load**. Token cost is independent of embedding *quality*; you
  only need chunks to exist. (With vector OFF, `toc_path` degrades to the note title —
  verified manually — so FTS-only is not a valid setup for this bench.)
- Corpus: synthetic notes, sweep section count and section size (e.g. 5/10/20 sections
  × ~450-token sections). Each section embeds a unique answer token; queries map to a
  known section.
- Fresh isolated SQLite, fast queue (`GLOBAL_QUEUE_POLL_INTERVAL=100ms`), wait for
  embedding jobs before measuring — same isolation pattern as `scripts/bench-pushnotes.sh`.
- **Not federated. Not the whoami mesh.**

## Harness + output (match the house style)

Mirror `scripts/bench-pushnotes.{sh,mjs}`:

- `scripts/bench-token-economy.sh` drives `scripts/bench-token-economy.mjs`, env-driven
  (`APP_URL`, `INTERNAL_URL`, mock embedding), sweeps params, emits **one JSON line per
  run** to stdout → `tee` into `.jsonl`.
- JSON fields: `sections`, `section_tokens`, `full_tokens`, `window_tokens`,
  `gql_all_tokens`, `gql_min_tokens`, `ratio_section`, `ratio_window`, `ratio_gql`,
  `answer_present` (per arm), `tokenizer`.
- Publish like `docs/dev/benchmark.md`: `.jsonl` → wide-form `.csv` →
  `docs/{en,ru}/user/token_economy_bench.datachart.csv`. Then update
  `Token Economy.md` / `Fuzzy Pointer.md` to cite the measured ratios instead of the
  current estimates.

## Out of scope / separate work

- **Retrieval quality** (Recall@10 / nDCG / MRR): already `docs/dev/retrieval_eval.md`
  + `docs/en/thoughts/search-benchmark.md`. This bench is cost, not quality.
- **Federation cost**: `federated_note_html(match_id)` window vs full note across a peer
  — a follow-up; `toc_path` is local-only.
- **Position-recall ("Lost in the Middle", `Token Economy.md:85`)**: whether an answer
  at the top of context beats one buried in a full note. This is an LLM-recall study
  (needs a judge model + a held-out QA set), not a deterministic token count — track it
  as its own experiment, not part of this bench.

## Validates

- `Token Economy.md:21` / `Fuzzy Pointer.md:68` — section ~300 vs full ~3000 (~10×).
- `Token Economy.md:38-73` — GraphQL field-selection savings.

---

## Update 2026-06-21 — what we actually built, and what it revealed

We did **not** build the synthetic mjs harness above. A simpler artifact answered the
question better: a **zero-dependency Python script** (`scripts/token_economy_check.py`)
that hits the LIVE public MCP endpoint (`trip2g.com/_system/mcp`) anonymously and
measures token economy on the real docs vault. Anyone can run it:
`python3 scripts/token_economy_check.py`.

Findings (live, real docs):

- A focused read costs **~11× fewer tokens** than the whole note (median across 9
  real queries). The "~300/3000 ≈ 10×" claim is directionally honest; the win scales
  with note size — long notes save most, short notes (already cheap) save little.
- The token saving is the **easy, nearly self-evident half**. The hard, interesting
  half is **selecting the right section**: deterministic selection (regex, then
  snippet-vs-HTML substring) failed on **8 of 9** real queries. Picking the
  answer-bearing `toc_path` is a *navigation* problem, not a token-count problem.
- Shareable chart: `docs/en/user/token-economy-bench.md` (+ ru) renders a live
  `datachart` from `docs/en/user/token_economy_bench.datachart.csv`.

## Next: `expand` — progressive-disclosure tree navigation (new MCP tool)

The benchmark surfaced the real feature. Today an agent gets the whole **flat** `toc`
from `search` and jumps straight to a leaf. Progressive disclosure means walking the
tree node by node **without loading content or searching every chunk**:

```
toc(pid)                       -> top-level sections          (~20 tokens)
expand(pid, toc_path)          -> direct children of a node   (~15 tokens)
note_html(pid, toc_path)       -> read the chosen leaf        (~300 tokens)
```

`expand(pid, toc_path)` returns the **direct children** of a TOC node — `title`,
`level`, `path`, `has_children` — cheap labels to decide which branch to enter. An
empty/omitted `toc_path` returns the top level. Vector search becomes the *entry
point*; structural navigation moves around from there. The heading-breadcrumb-in-chunk
(Contextual Retrieval, already shipped as F4) is the bridge: a fuzzy query lands on a
chunk whose embedded breadcrumb **is** a precise `toc_path`.

Two corollaries from review:

- **Drop the flat `toc` from `search` results.** Once `expand` exists, structure is
  fetched on demand, so the per-result `toc[]` that `search` ships today is dead
  weight in every response — removing it slims search too. Nothing depends on it
  after `expand` lands.
- **Federated parity.** Every navigation tool needs a `federated_*` twin
  (`federated_expand`) so a peer KB's tree navigates exactly like a local one,
  matching `federated_search` / `federated_note_html`.

External fit: a "detailed TOC of a 10-hour podcast" (knowlume distills long podcasts
and exposes exactly such a tree) is navigated with `expand` instead of dumping the
transcript — the same pattern, different corpus.

We start implementing `expand` (+ `federated_expand`, + search slimming) next.
