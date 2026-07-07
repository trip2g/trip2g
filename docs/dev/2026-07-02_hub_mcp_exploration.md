# Hub MCP exploration — fresh-agent test, 2026-07-02

**Verdict: the current instructions are almost enough for the happy path, but not for federated search in practice.** A fresh agent finds the hub index and learns to target `kb_id`s — that part works. What breaks it: fan-out failures are invisible in the text output, one peer type returns result bodies only in `structuredContent`, remote `expand` is not implemented by peers, and the flagship example peer (foragent) was down the whole session. An agent that trusts the text output will confidently report "no results" while five of six peers timed out. The instructions need a "failure modes" extension more than anything else.

Method: everything below was discovered through `https://trip2g.com/_system/mcp` only (JSON-RPC over curl), following the server's own `initialize` instructions. No repo or vault reading until the wording proposals at the end.

## What a fresh MCP agent finds

**The base.** trip2g is a Markdown publishing platform (Obsidian vault → website with subscriptions, Telegram publishing, MCP for agents). The base holds user guides (`en/user/`, `ru/user/`), changelog, dev docs, plans, and `thoughts/` essays. `search` → `note_html(toc_path)` → `expand` all work as documented; section reads land at ~300–2,500 chars.

**Discovery path that worked:**

1. `initialize` → instructions describe the retrieval loop and mention `federated_search` + `kind: "federation_kb"`.
2. `search("hub federation connected bases index")` → `en/user/hub.md` ("Public hub of curated bases") in position 5, without a snippet.
3. The `instructions` tool (full reference) names the hub index explicitly: `note_html(path="en/hub/_index.md")`. That call returns the roster.

**Public peers (from `en/hub/_index.md`):**

| kb_id | Hub note | Content | Status during test |
|---|---|---|---|
| `nicksenin_journal` | en/hub/nicksenin_journal | Nikolai Senin's journal + Code with Claude 2026 case index (RU) | Up, ~0.7s |
| `markavrelii` | en/hub/markavrelii | Marcus Aurelius, Meditations (RU, Книги/Цепочки/комментарии Унта) | Up, ~1.3s |
| `telegram_public` | en/hub/telegram | Telegram channel adapter (Denis Sexy IT, AI и грабли, …) | Up, ~1.0s — but see below |
| `foragent` | en/hub/foragent | Agent skills from GitHub (superpowers etc.), standalone adapter node | **Down all session**: `post json: timeout` on every attempt |

Two more `kb_id`s exist that the index does not list — they surfaced only in fan-out error payloads: `peer` (DNS failure on `app-peer`, looks like a dead internal test entry) and `telegram_demo` (SSRF-blocked, resolves to localhost). Mild info leak, and pure noise for an agent.

**Peer-specific question answered.** "What does Marcus Aurelius say about death and time?" → `federated_search(kb_id="markavrelii", query="смерть и время")` → 14 notes, then `federated_note_html(kb_id, pid=33, match_id="p33:c32")` returned a focused ~1KB window from Книга 02. This is the full loop working end to end, and it's genuinely good when the peer is up.

## Friction / gaps for federated discovery and search

Ordered by how badly each would mislead a weak agent.

1. **Fan-out errors are text-invisible.** `federated_search` with no `kb_id` renders only peers that answered. One run returned literally `[nicksenin_journal]\nNo results found` as the entire text — while `structuredContent.errors` held timeouts for markavrelii, telegram_public, and foragent plus two unreachable peers. A text-only agent concludes the whole network has nothing. This is the single worst trap.

2. **telegram_public results are structured-only.** Text output: `Found 20 results for "vibecoding"` — 33 characters, no bodies. The actual 20 results (snippets, `t.me` links, proof metadata) live only in `structuredContent`. For a text-only client the peer is search-positive but content-blind.

3. **`kind: "federation_kb"` pointers work, but only structurally.** The marker and its helpful `agent_instruction` ("call federated_search with kb_id=...") exist in `structuredContent`; the text rendering of the same search result shows nothing that distinguishes a KB-note from a regular note. The documented "a result had kind: federation_kb → federate" trigger never fires for a text-only agent.

4. **Remote `expand` is not implemented by peers.** `federated_expand(kb_id="markavrelii", pid=33)` → `federation rpc error -32601: Method not found: expand`. The progressive-disclosure loop the instructions drill into you breaks silently on remote bases, with no documented fallback.

5. **`federated_note_html` argument mismatch.** `match_id` alone is rejected remotely (`one of pid, note_id, path, or href is required`) even though the local reference presents `match_id` as a self-sufficient pointer. `pid` + `match_id` together works well. Undocumented.

6. **Fan-out is flaky, timeout budget ~2s.** Same query twice: first run markavrelii answered with 10 notes; second run it timed out. Wall time on timeouts was consistently ~2.4s. Nothing tells the agent that a missing peer might answer a targeted retry — which it did.

7. **Empty vs down is indistinguishable in text.** "No results found for X" (peer answered, has nothing) and a peer silently absent (down) read the same unless you parse `structuredContent.errors`.

8. **Hub-index discovery is fine only if you call the `instructions` tool.** The `initialize` instructions mention `federated_search` and the pointer kind but not the hub index; `search("hub")` surfaces `en/user/hub.md` without a snippet at rank 5, easy to skip. The explicit `en/hub/_index.md` path appears only in the long-form `instructions` tool output. A lazy agent that skips that tool discovers peers only by accident.

9. **foragent is the worked example and it's down.** Both the hub note and `en/user/hub.md` use `kb_id="foragent"` as the canonical example. Every call timed out. No instruction covers what to say or do about an unreachable peer.

Not instruction problems, but worth fixing server-side (bigger wins than any wording): render `errors` and the `federation_kb` marker into the text content; render telegram-adapter results into text.

## Recommended extensions to the Federated section of `_mcp_instructions.md`

The live served text already has a good "Discovering connected bases" paragraph (hub-index-first, per-peer `kb_id`). Keep it. Add the following, roughly in this order, after the existing trigger paragraph:

```markdown
**Discovery order.** Before any blind fan-out, read the hub index:
`note_html(path="en/hub/_index.md")`. It lists one note per connected base;
open the base's hub note (e.g. `en/hub/foragent.md`) to learn what it covers
and which questions it answers well, then target it:
`federated_search(kb_id="...", query=...)`. Fan out (no `kb_id`) only when
you genuinely don't know which base would hold the answer.

**Reading fan-out results.** A fan-out reply lists only the peers that
answered in time. Always check `structuredContent.errors` in the same reply:
peers time out routinely (the budget is a couple of seconds), and a peer that
is missing from the text may still hold the answer. For every peer you
expected but don't see, retry once with a targeted
`federated_search(kb_id=...)` before drawing any conclusion.

**Empty vs unreachable.** "No results found" means the peer answered and has
nothing — say so. A timeout or connection error means the base is currently
unreachable — report it as unreachable, never as "no results". If a targeted
retry also fails, stop retrying and tell the user which base you could not
reach.

**federation_kb pointers.** The `kind: "federation_kb"` marker and its
`kb_id` appear in the structured result payload, not in the plain-text
rendering. If you only see text, treat any result under `hub/` as a probable
base pointer and check its hub note for the `kb_id`.

**Remote reads.** Peers may not implement `expand` — if `federated_expand`
returns "Method not found", skip the TOC survey and go straight to
`federated_note_html`. On `federated_note_html`, always pass `pid` (or
`note_id`/`path`/`href`) from the search result; a `match_id` alone is
rejected by peers. `pid` + `match_id` together gives the focused window.

**Non-note peers.** Some bases are adapters over other systems (e.g.
Telegram channels). Their results use their own `kind` (like
`telegram_message`) and id formats, and their content may only be present in
the structured payload. Cite the `url` from the result (e.g. the `t.me`
link); don't assume `pN:cM` ids or `toc_path` sections exist there.
```

Two smaller wording tweaks outside that block:

- In the short `initialize` instructions (the one-line federated trigger), append: "…pass `kb_id="..."` to target one. The list of connected bases lives at `note_html(path="en/hub/_index.md"`)." Right now the hub index is only in the long-form reference, and agents that never call the `instructions` tool miss it.
- In `en/user/hub.md` / the hub notes, stop leaning solely on `foragent` for examples, or fix the adapter — the flagship example failing every call undermines trust in the whole mechanism.

## Session log (condensed)

- `initialize` → instructions with retrieval loop, federation trigger, worked example (pid 658 federation guide).
- `tools/list` → 8 core tools + `instructions`, `wiki`, RU `instructions` (duplicate name `instructions` appears twice in the list — minor).
- Local searches: platform overview (anatomy-15-months, contentless-cms), hub docs, federation docs — all clean.
- `expand`/`note_html(toc_path)` on `en/user/hub.md` — section reads behaved exactly as documented.
- Fan-outs: run 1 → telegram_public 20 + nicksenin empty, no error info in text; run 2 → markavrelii 10 notes then, on repeat, only nicksenin in text with 5 peers in `errors`.
- Targeted: markavrelii OK (search → note window → `federated_similar` OK), nicksenin_journal OK, telegram_public search OK but bodies structured-only, foragent timeout ×4.
- `federated_expand` on markavrelii → Method not found. `federated_note_html(match_id)` alone → rejected; with `pid` → OK.
