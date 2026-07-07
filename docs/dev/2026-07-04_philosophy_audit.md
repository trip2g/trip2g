# Philosophy section audit (docs/en|ru/thoughts) — 2026-07-04

## Вывод

The section is in better shape than feared: the core philosophy pages (knowledge-graph, three-zones, why-not-book, contentless-cms, declarativity, KaaS) are accurate and tight, and RU/EN pairs are mostly in sync. The real problems are three: (1) **markdown-operating-system is factually stale in both languages** — its whole framing is "agents are still on a branch", but fleet/agentruntime merged to main (PR #53); (2) **navigation rot** — 7 published pages are in neither `_sidebar` nor `_index`, EN has no pair for RU `testing-agent-skills`, and the two `_index` files list pages in unrelated orders; (3) a scatter of small inaccuracies (reranker "code kept" — it was removed; anatomy's SQLite-driver claim; one broken EN wikilink; RU `lang_redirect` missing on 11 pages). Expansion is worth it on exactly two pages: **digital-sovereignty** and **ai-changes-rules** — both are thin 36-line sketches while the product has since shipped the features that would make them concrete. The "paths not taken" piece is drafted at `docs/dev/2026-07-04_paths_not_taken_DRAFT.md`.

Related: `docs/dev/2026-07-02_landing_link_audit.md` already covers the mesh-landing link layer (footer `/knowledge_graph` etc. return 200; the broken things there are `/simple` CTA, `mcp:add`, the Meditations initialize note). Not duplicated here.

---

## A. Factual inaccuracies (fix these)

### A1. `markdown-operating-system.md` — "branch" tags are stale (both languages) — HIGHEST
The page's honesty device ("every row tagged shipped/branch/planned") is now itself wrong:
- `docs/en/thoughts/markdown-operating-system.md:9` — "agents as processes, is still on a branch. This is the honest version" — **false**: fleet/agentruntime is on main (`internal/agentruntime/`, `cmd/fleet/`; PR #53 merged, plus follow-ups #61, code-executor).
- `:25` "Branch runs on `feat/agent-runtime`", table rows `:48-52` (kanban agent wiring / process executor / package manager / resource limits = "branch"), `:84` "it is the layer that is not on main yet", `:100` "what the branch adds", `:132` "part of the branch", `:144` "only on the branch" + "there is no shell or exec tool for agents" — **also false now**: the code-executor (`executor: code` role + exec tool, `internal/agentruntime/coderun.go`) shipped, so even the "no exec tool" caveat is outdated.
- `:138` read-replica "lives on the `feat/read-replica` branch" — still true, keep.
- RU mirror `docs/ru/thoughts/markdown-operating-system.md:9,25,50-52,...` has the same stale tags.
Fix: retag the four rows to shipped, rewrite the lead and "Where the metaphor breaks" (the breaks are now: editor, single-VM resource pressure, read-replica branch). This is the flagship architecture page; it undersells the product today.

### A2. `search-benchmark.md` — reranker "kept the code" is false
- `docs/en/thoughts/search-benchmark.md:80` — "We shipped the reranker off by default (`vector_search.reranker.enabled=false`) and kept the code for per-deployment A/B testing" — the reranker was **removed entirely** in `adf84ce7` (PR #78, 2026-07-01; `internal/reranker/` + `reranker-server/` deleted; finding preserved in `docs/dev/reranker.md`). Same sentence in the RU pair.
Fix: one-sentence update ("later removed entirely; the negative result is recorded in …"). This page is otherwise the strongest tooling article in the section.

### A3. `anatomy-15-months.md` — two stale claims + one broken link
- `docs/en/thoughts/anatomy-15-months.md:82` — "`mattn/go-sqlite3` + `modernc.org/sqlite` — two SQLite drivers. CGO version for production" — go.mod now has mattn as `// indirect` only (`go.mod:253`); production runs modernc (see also `blaming-sqlite` which says so correctly). RU pair same.
- `:25` — "Vendor bundles (Tiptap, etc.) — 190 000 lines" — tiptap/milkdown legacy bundles were removed (`91ab6dae`); the count is a snapshot, fine, but the parenthetical now names a removed thing. Low priority (it's a dated report; a "figures as of 2026-05" note would cover it).
- `:73` — `[[ru/user/jet|Jet docs]]` on the **EN** page links into the RU tree; `docs/en/user/jet.md` does not exist. Either create the EN jet doc or link the RU one explicitly as "(Russian)".

### A4. Nav/index integrity
- **EN missing `testing-agent-skills`**: `docs/ru/thoughts/testing-agent-skills.md` exists, is in the RU sidebar (`docs/ru/thoughts/_sidebar.md:27`) and RU index (`_index.md:41-43`), and has **no `lang_redirect` and no EN pair**. Violates the bilingual-pairs rule. Either write the EN pair or add `lang: ru` and accept it as RU-only explicitly.
- **7 pages orphaned from nav in both languages** (published, reachable by URL, in neither `_sidebar.md` nor `_index.md`): `universal-tools`, `vibecoding`, `markdown-vault`, `env-pattern`, `anatomy-15-months`, `surviving-node-loss`, `mcp-instructions-make-cheap-models-faster`. Several are among the best pages in the section (universal-tools, mcp-instructions). Add a nav pass; the sidebar's existing **Philosophy / Architecture / Knowledge & Content / Tooling** grouping fits them all.
- **`_index` order diverges**: EN `_index.md` opens with knowledge-graph → three-zones → why-not-book (a deliberate funnel); RU `_index.md` opens with admin-panel → declarativity → obsidian (looks accidental). Align RU to the EN order unless the difference is intentional.
- `universal-tools` returns **404 on trip2g.com** (`/en/thoughts/universal-tools`) — committed but apparently not synced to prod. Sync or confirm intentional.
- `vibecoding` (both langs) is the only pair with **no `free: true` and no `lang_redirect` in either direction** — language switcher won't work on it.
- 11 RU pages missing `lang_redirect` back to EN while their EN pair points at them: admin-panel, declarativity, obsidian-and-wikilinks, content-infrastructure, track-bot, personal-navigation, knowledge-bot, anatomy-15-months, env-pattern, search-benchmark, sync-benchmark (each `:1-6`). The newer pages (markdown-vault, public-hub, blaming-sqlite, etc.) do it correctly on both sides — backfill the old ones to the new convention.

### A5. RU/EN content drift (one real case)
- `docs/ru/thoughts/surviving-node-loss.md` — (a) missing the EN section "What's next: zero-downtime reads via a read replica" (EN:64-66); (b) internally inconsistent numbers: RU:7 intro and RU:69 bottom line say "в 2-3 раза" while RU:15 body and EN say 2x-10x.
- `vibecoding` RU is a native adaptation (good), but cites different specifics than EN: RU:19 "Claude Code … $2,5 млрд за девять месяцев" vs EN:21 "zero-to-billions inside a year"; RU:74 cites arxiv for Carnegie Mellon, EN:65 cites Harvard Gazette. Pick one set of claims and mirror it — divergent figures in a "verify facts" essay invite nitpicks.
- Everything else checked pair-by-pair: clean (structure, numbers, links match).

### A6. Claims to verify before anyone quotes them
- `three-zones.md:30` (both langs): "A reader without a subscription will see that a link exists, but won't be able to follow it." Verify this matches current paywall behavior (does the link render? teaser? 404?). If the actual UX differs, this is the kind of small overpromise the owner suspects.
- `track-bot`, `personal-navigation`, `knowledge-bot` all open with "This feature is in development" + a personal Telegram CTA (`t.me/brodfreed`). Check each against shipped reality: track-bot's Canvas→bot pipeline, the entry questionnaire, AI TOC. If still genuinely in development, fine — but these three are the pages most likely to have silently rotted into vaporware descriptions; if any part shipped (or was dropped), retag. Also decide whether a personal TG handle is the CTA you want on evergreen philosophy pages.
- `ai-changes-rules.md` — no false statement, but it describes as future-abstract what the product now does concretely (MCP tools, instructions note, per-agent scoping). See expand list.

---

## B. Which pages to EXPAND (ranked)

1. **`digital-sovereignty`** (36 lines) — the thinnest page relative to what shipped. It argues "your files, your server" abstractly while the product now has the concrete receipts: `git clone` of the whole vault over `/_system/git`, single-binary self-host, OIDC SSO, Litestream/object-storage durability (`surviving-node-loss` proves the "migration in a day" claim with a 35-second measured restore). Fold in one paragraph per proof and link them. It's also a footer link on the landing — high-visibility.
2. **`ai-changes-rules`** (36 lines) — written before MCP existed; still says "readers converse with your knowledge through AI" as a promise. It's now mechanism: MCP endpoint, author-written instructions note (with a measured 26%-fewer-calls benchmark on this very site), scoped agent access, fleet. Expanding this turns the weakest hand-wave into the best-evidenced page. Link `mcp-instructions-make-cheap-models-faster`, `content-infrastructure`, `public-hub`.
3. **`three-zones`** (30 lines) — good idea, no mechanism. One section on how zones are actually drawn (`free:` frontmatter, subgraphs, subscriptions, MCP access following the same zones) would ground it; also resolves A6.
4. **`knowledge-as-a-service`** (34 lines) — fine standalone, but pre-dates federation/public-hub. A short "and across bases" closer linking `public-hub` would connect the business page to the network story.
5. **`admin-panel` / `declarativity`** — short but complete; expansion optional. `knowledge-graph` / `why-not-book` / `contentless-cms` — leave alone, they're the crispest pages in the section.
Not expansion but promotion: `universal-tools` and `mcp-instructions-make-cheap-models-faster` are strong and invisible (no nav entry; universal-tools not even deployed). Cheapest win in the whole section.

---

## C. "Paths not taken" draft

Staged at `docs/dev/2026-07-04_paths_not_taken_DRAFT.md`. Five verified stories (cross-encoder reranker, AND→OR fallback, Qdrant, kanban's two homes, Telegram premium-bot economics) + a short goodbyes list, each with commit/doc evidence; framed as engineering discipline (measure, don't add moving parts), not a changelog. RU adaptation notes at the bottom of the draft.
