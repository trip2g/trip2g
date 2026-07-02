# How to write good MCP instructions for your base

**TL;DR:** every base needs its own `mcp_method: initialize` note, or agents get the wrong identity (trip2g.com served the demo's Marcus Aurelius prompt for months). Write it for the weakest model that will connect: identity in 2-3 sentences, the retrieval loop as numbered steps, one worked example with real values, one-line triggers for the other tools, and guardrails built from the actual failure paths. The reference example is `docs/_mcp_initialize.md` + `docs/_mcp_instructions.md`.

## The mechanism

Two frontmatter keys, both handled in `internal/case/mcp/resolve.go`:

- `mcp_method: initialize` — the note body (frontmatter stripped) is injected as the `instructions` field of the MCP `initialize` response. Every client sees it at connect time. This is the note that matters.
- `mcp_method: <name>` (any non-reserved name, e.g. `instructions`) — the note becomes a callable tool named `<name>` that returns its body. Use it for the longer reference the initialize note points to. `mcp_description` sets the tool description in `tools/list`.

Both need `free: true` to be visible to anonymous MCP callers.

**Shadowing gotcha:** when several notes share an `mcp_method`, the first one in path-sorted order wins (`handleInitialize` / `handleDynamicMethod` iterate `LatestNoteViews().List`, which `ExtractNoteList` sorts by path). That is why root-level `_mcp_initialize.md` shadows `demo/_mcp_initialize.md` (`_` < `d`) — and why the leak happened in the first place: with no root note, the demo's was the only match. Don't rely on shadowing across sibling folders; keep one authoritative note per method.

## The recipe

Order matters — a weak model weights the top of the prompt heaviest:

1. **Identity, 2-3 sentences.** What this base is and what topics live here. The agent uses it to decide whether a question is even answerable locally.
2. **The retrieval loop, as numbered steps.** For trip2g bases: `search` → `note_html(toc_path=match.toc_path)` → `expand` when the pointer misses → whole-note read only for sectionless notes. State the payoff in numbers ("one section ~300 tokens, full note 3,000+") so the model has a reason to comply.
3. **One worked example with real values.** A real query, a real `note_id`, a real `toc_path` from your base. Weak models copy examples far more reliably than they follow abstract rules.
4. **One-line triggers for the other tools.** "`similar` — you found one good note and want its neighbors. `federated_search` — local search came up empty, or a result had `kind: federation_kb`." One trigger per tool; no essays.
5. **Guardrails from the failure paths.** Write these from how the system actually fails, not how it should work. For trip2g: a wrong `toc_path` silently returns the *whole note* (no error), so teach the recovery — "response much longer than a section = pointer miss; call `expand`, re-read with an exact heading"; `toc_path` must match headings exactly; `pN:mM` match_ids don't resolve.

## Weak-model rules

- **Tight beats complete.** A haiku-class model degrades with wall-of-text; the initialize note should fit in ~50 lines. Push the full argument reference into a callable `instructions` tool-note.
- **Exact names only.** The tool is `expand`, not "extend"; the argument is `toc_path`, not "section". Copy names from `handleToolsList` in `resolve.go` — if the note and the schema disagree, the agent flips a coin.
- **Imperative, not descriptive.** "Never open a note without `toc_path` on the first read", not "it is generally more efficient to…".
- **Don't document fields that no longer exist.** The docs taught a removed `toc` search field for months (see `2026-07-02_vector_search_token_economy.md`, finding 3); an agent hunting for a phantom field burns a turn and then falls back to the expensive path.

## Validate against the live endpoint

Don't ship instructions you haven't run:

```sh
# identity: does initialize return YOUR note?
curl -s https://your-base/_system/mcp -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","method":"initialize","id":1}'

# names: does tools/list match what your note claims?
curl -s https://your-base/_system/mcp -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","method":"tools/list","id":1}'

# the loop itself: search -> expand -> note_html(toc_path) on a real term
curl -s https://your-base/_system/mcp -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","method":"tools/call","id":2,"params":{"name":"search","arguments":{"query":"<real docs term>","limit":3}}}'
```

Then walk one drill-down by hand: take a `toc_path` from the search result, `note_html` it, and compare the section length to the full note. If you can, point a cheap model at the endpoint and watch whether it follows the loop; every place it stumbles is a line your note is missing.

## Related

- `docs/dev/mcp.md` — endpoint and dynamic-method basics (predates `note_html`/`expand`/federation; the tool list there is stale)
- `docs/en/user/Token Economy.md`, `Fuzzy Pointer.md`, `expand.md` — the drill-down pattern the instructions teach
- `docs/dev/2026-07-02_vector_search_token_economy.md` — the failure-path audit the guardrails are built from
