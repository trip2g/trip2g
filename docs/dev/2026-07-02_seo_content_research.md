# SEO content research: model articles for the Step-1 batch

Date: 2026-07-02. Companion to `docs/dev/2026-07-02_seo_growth_plan.md` Step 1 and `docs/dev/seo_plan.md` §1. Web research with every source cited inline. trip2g feature claims verified against `README.md` (status column) and `docs/en/user/` on 2026-07-02.

## TL;DR

Three best models overall:

1. **[Unmarkdown — "Obsidian Publish Alternatives"](https://unmarkdown.com/blog/obsidian-publish-alternatives)** — the best vendor-written comparison found. ~2,100 words, 15-row feature matrix where competitors win rows, vendor introduced late as "a different kind of alternative", explicit "it doesn't have graph view" admission. The honesty *is* the conversion mechanism.
2. **[Particula Tech — "Agent Memory Frameworks Tested: Mem0 vs Zep vs Letta"](https://particula.tech/blog/agent-memory-frameworks-tested-mem0-zep-letta-cognee-2026)** — the best neutral comparison. Benchmark numbers (LongMemEval 63.8% vs 49.0%), named tradeoffs per tool, and a "Pick X when:" verdict per section instead of "best overall". This is the shape for `trip2g-vs-mem0-vs-memgpt.md`.
3. **[Mem0 — "Add Persistent Memory to Claude Code (5-Minute Setup)"](https://mem0.ai/blog/claude-code-memory)** — the best vendor tutorial. Friction-first intro, prerequisites block, three installation paths (easiest first), verification step, 8-question FAQ. This is the shape for `claude-code-persistent-memory.md`.

**The one reusable pattern:** every winning piece leads with the searcher's pain, gives the answer/verdict before the detail, uses a scannable table where the author's own tool visibly *loses* some rows, and closes each option with "best for / pick when". Honest tables are not a handicap — in every dissected example they are the ranking and conversion engine.

---

## Reusable template A — "X vs Y / X alternative" comparison page

Distilled from [Unmarkdown](https://unmarkdown.com/blog/obsidian-publish-alternatives), [Particula](https://particula.tech/blog/agent-memory-frameworks-tested-mem0-zep-letta-cognee-2026), [Docmost](https://docmost.com/blog/gitbook-alternatives/), [MCPBundles](https://www.mcpbundles.com/blog/obsidian-mcp-vault-ai), [ssp.sh](https://www.ssp.sh/brain/open-source-obsidian-publish-alternatives/).

1. **Title** = exact-match query + freshness ("Obsidian Publish alternatives (self-hosted, 2026)"). Visible "Updated" date.
2. **Intro (2–4 sentences), pain-first then answer-first.** Name the cost/limit that sent the reader here ("$8–10/month and no custom templates"), then state the verdict up front: "Quartz is the best free static option; trip2g is the option if you need a live server with paywalls and MCP."
3. **Quick-verdict block** right after the intro: a 3–5 line "best for" list (best free / best zero-setup / best self-hosted live). This is the featured-snippet bait.
4. **Feature matrix**: rows = 10–15 decision criteria (price, hosting model, setup effort, two-way sync, custom templates, paywalls, MCP/agent access, search, graph view, RSS, Telegram, data ownership), columns = tools. **trip2g must visibly lose rows** (graph view, free static hosting, community size). Unmarkdown's trick: own a column dimension nobody else addresses (for us: "agents can query it over MCP" and "paywalled sections").
5. **Per-alternative sections**, one consistent shape each (50–150 words): *what it does well → the tradeoff → best for*. Link the competitor's own docs/pricing page (credibility + accurate claims).
6. **The trip2g section as a reframe, not a row-by-row brag** — "a different kind of alternative: a live server, not a static build" — and open with what it *doesn't* do ("no graph view; you run a server"). Late placement (after 2–3 alternatives) reads as editorial, not ad; Docmost's first-slot placement works too but reads saltier.
7. **"When to choose what"** decision framework: one "Pick X when: …" line per tool, including competitors (Particula pattern).
8. **FAQ**, 4–8 questions in natural long-tail phrasing ("Can I self-host Obsidian Publish?", "Is Quartz really free?"). Maps 1:1 to FAQPage JSON-LD once the engine emits it.
9. **CTA**: soft and layered — docs quickstart wikilink, GitHub repo, free cloud instance. No hard sell; the tutorial momentum converts (Mem0 pattern).
10. **Internal links**: wikilinks to `two-way-sync`, `mcp`, `monetization`, `templates`, `selfhosted`, and to the sibling comparison pages (topical cluster). ~1,500–2,500 words.

## Reusable template B — "how to …" tutorial page

Distilled from [Mem0](https://mem0.ai/blog/claude-code-memory), [Bryan Hogan](https://bryanhogan.com/blog/obsidian-website), [Juhis/hamatti](https://notes.hamatti.org/technology/building-a-digital-garden-with-obsidian-and-quartz), [dev.to self-hosted mem0](https://dev.to/n3rdh4ck3r/how-to-give-claude-code-persistent-memory-with-a-self-hosted-mem0-mcp-server-h68), [Jacob Kaplan-Moss](https://jacobian.org/til/hugo-obsidian/).

1. **Title** with a true time/effort promise ("… in 10 minutes") — only if it's actually true.
2. **Intro = one concrete friction scene** ("every session you re-explain the project"; "two hours lost re-debugging") then answer-first: what the reader will have at the end, in one sentence.
3. **Options-overview table before committing to one method** (Bryan Hogan's move): method / complexity / cost / who it's for. Include competitors honestly — the page then ranks for the head query, not just our tool, and the reader self-selects.
4. **Prerequisites block** (3–5 items, each a link or command).
5. **Numbered steps**, easiest path first, with alternate paths (Docker vs binary vs cloud) as subsections — multiple entry points cut bounce (Mem0). Code blocks scoped and copy-paste ready, never walls.
6. **A verification step** after each major stage ("run `/mcp` and you should see…") — prevents silent failure, and "it worked" moments are what get linked.
7. **Friction admitted** in its own subsection (Jacob Kaplan-Moss admits Hugo needs "knitting together"; hamatti complains about Quartz's client-side JS). Admissions are the E-E-A-T signal that separates a tutorial from marketing.
8. **What's next** (2–3 wikilinks deeper into docs) + short FAQ + soft CTA. ~800 words for a narrow task, up to ~2,500 for a full setup guide.

---

## Per-target research

### Lane A — Obsidian publishing

#### 1. `obsidian-publish-alternative.md` — "obsidian publish alternative", "obsidian publish self-hosted"

**Models.**
- [Unmarkdown — Obsidian Publish Alternatives](https://unmarkdown.com/blog/obsidian-publish-alternatives) — vendor comparison done right (see TL;DR). 15-row matrix, "different kind of alternative" reframe, three soft CTAs.
- [Simon Späti — Open-Source Obsidian Publish Alternatives](https://www.ssp.sh/brain/open-source-obsidian-publish-alternatives/) — ~400 words, ranks on freshness ("last updated" since 2022), aggressive internal wikilinking, and the page *is* a published digital garden. This is exactly our dogfooding play; his vault is our closest structural precedent.
- [Obsidian Forum thread](https://forum.obsidian.md/t/obsidian-publish-alternatives/22886) — what searchers actually ask; mine it for FAQ phrasing.

**Why they rank.** Exact-match long-tail title, pain-first intro ("Publish is great but expensive"), freshness signals, and either a feature matrix (Unmarkdown) or a live-demo effect (ssp.sh).

**Searcher intent.** Obsidian user paying or about to pay $8–10/mo, wants cheaper/self-hosted/more control. Bottom-of-funnel, tool-shopping.

**trip2g lead-with (all shipped).** Self-hosted, MIT, $0 license; two-way sync so the vault stays local and edits flow both ways; custom Jet templates (Publish has CSS snippets only); paywalls/monetization; Telegram publishing; MCP endpoint on the same notes; multi-domain.

**Where Obsidian Publish legitimately wins.** Zero setup, official and maintained by the Obsidian team, polished graph view, no server to run or update, predictable hosting. Say so plainly: "if you never want to think about a server, pay the $8 and stop reading."

#### 2. `trip2g-vs-quartz.md` — "quartz alternative", "obsidian digital garden self-hosted"

**Models.**
- [Juhis — Building a digital garden with Obsidian and Quartz](https://notes.hamatti.org/technology/building-a-digital-garden-with-obsidian-and-quartz) — honest user write-up: admits Quartz's heavy client-side JS and doc gaps. His friction list is our comparison's raw material.
- [Simon Späti — Quartz: Publish Obsidian Vault](https://www.ssp.sh/brain/quartz-publish-obsidian-vault/) — pro-Quartz reference point ("huge community, features merged daily"); cite it as the fair steelman.
- [XDA — How I Turned My Obsidian Notes into a Website Using Quartz](https://www.xda-developers.com/turned-obsidian-vault-into-website/) — big-publisher how-to; shows the "no coding needed" hook Quartz owns.

**Why they rank.** First-person experience with real friction (hamatti), topical authority + freshness (ssp.sh), domain authority + benefit-hook headline (XDA).

**Searcher intent.** Knows Quartz, either evaluating it or hitting its walls (rebuild loop, git-per-edit, JS weight, no dynamic anything).

**trip2g lead-with.** No build step: edit in Obsidian → live in ~a second via two-way sync (vs edit → build → commit → deploy); dynamic features a static site can't have — paywalls, forms, subscriptions, RSS per base, live editing; MCP so agents query the same garden; per-domain routing.

**Where Quartz legitimately wins.** Completely free hosting (GitHub Pages), no server to operate or secure, huge community and theme/plugin velocity, graph view, static-site speed and simplicity. Honest close: "static + free beats a server if your garden is read-only."

#### 3. `trip2g-vs-hugo.md` — "publish obsidian vault hugo"

**Models.**
- [Jacob Kaplan-Moss — Publishing an Obsidian vault with Hugo](https://jacobian.org/til/hugo-obsidian/) — high-authority author; the article is a candid catalog of the pain (wikilink conversion, `_index.md` workarounds, pre-commit hooks, "knitting together" tools) and even opens by admitting "the easy way is $100/yr on Obsidian Publish". The friction he documents is precisely the pipeline trip2g deletes.
- [Sagar Behere — Publishing Obsidian vault with Hugo](https://sagar.se/notes/computers/hugo/digital-garden/publishing-obsidian-vault-with-hugo/) and [obsidian-to-hugo](https://github.com/devidw/obsidian-to-hugo) — the converter-script genre; shows how much custom glue the workflow demands.

**Why they rank.** Author authority (jacobian), exact-match workflow query, real code and admitted tradeoffs.

**Searcher intent.** Technical user mid-struggle: wants Obsidian authoring + a website, is fighting wikilinks/frontmatter/deploy glue.

**trip2g lead-with.** The whole conversion pipeline disappears: wikilinks are native (Goldmark), frontmatter is the routing/config, sync replaces the export-convert-commit-deploy loop; git still available (vault served over git Smart HTTP at `/_system/git`), so scripted workflows survive.

**Where Hugo legitimately wins.** Enormous theme ecosystem, total control of output HTML, free static hosting, battle-tested at scale, no server. If the reader enjoys the build pipeline, Hugo is the more flexible machine.

#### 4. `trip2g-vs-notion-sites.md` — "notion sites alternative self-hosted"

**Models.**
- [selfh.st — Self-Hosting Guide to Alternatives: Notion](https://selfh.st/alternatives/notion/) — the authority hub for "self-hosted alternative to X" queries (fetch blocked us, but it consistently tops these SERPs; study in browser).
- [Docmost — 5 Open-Source Alternatives to Notion](https://docmost.com/blog/open-source-notion-alternatives/) — vendor listicle: consistent per-tool shape (1–2-sentence intro, 5–7 feature bullets, GitHub link), vendor first, ~1,200 words, single CTA. Efficient but saltier than Unmarkdown; copy the section shape, not the self-first placement.
- [openalternative.co — Open Source Notion Alternatives](https://openalternative.co/alternatives/notion) — programmatic-SEO directory shape; shows what we're up against for the head term (go long-tail: "notion **sites** alternative").

**Why they rank.** selfh.st = topical authority + audience trust; docmost = clean scannable structure + high-intent keyword; openalternative = programmatic scale.

**Searcher intent.** Notion user who liked one-click "Sites" publishing but wants out of per-seat pricing/lock-in, or needs self-hosting for compliance.

**trip2g lead-with.** Your content is markdown files you can `git clone` out at any moment (vs Notion export lossiness); self-hosted, no per-seat price; Obsidian as the editor; forms cover the "collect input" use Notion Sites is often chosen for; multi-domain.

**Where Notion legitimately wins.** WYSIWYG editing for non-technical teammates, databases/boards as first-class objects, zero infrastructure, real-time collaboration. If the team won't touch markdown, Notion is the right call — say it.

#### 5. `trip2g-vs-gitbook.md` — "gitbook open source alternative"

**Models.**
- [Docmost — Top 5 GitBook Alternatives](https://docmost.com/blog/gitbook-alternatives/) — dissected above: ~2,000 words, vendor first with screenshot + feature list + CTA, competitors get pros/cons only. Works via primacy, but its bias is visible; blend its structure with Unmarkdown's honesty.
- [apidog — Top 10 Best GitBook Alternatives](https://apidog.com/blog/gitbook-alternatives/) and [AlternativeTo — Open Source GitBook Alternatives](https://alternativeto.net/software/gitbook/?license=opensource) — the listicle/directory incumbents; confirm the term is contested, so our page should target the narrower "gitbook alternative **self-hosted**"/"**open source** gitbook alternative for Obsidian".

**Searcher intent.** Team docs shopper: outgrew GitBook free tier or wants self-hosting.

**trip2g lead-with.** MIT self-hosted, one Go binary on SQLite (no Postgres/Redis stack to run — contrast even with other self-hosted wikis); authors write in Obsidian, not a web editor; custom Jet templates for branded docs; MCP means the docs are directly queryable by the team's AI agents — a row no GitBook alternative fills; git access to the whole base.

**Where GitBook legitimately wins.** Polished hosted UX, change-request/review workflow, API-docs tooling (OpenAPI), enterprise SSO tiers, zero ops. For a large team wanting managed docs with review gates, GitBook earns its price.

#### 6. `publish-obsidian-vault-as-website.md` — head how-to: "how to publish an obsidian vault as a website"

**Models.**
- [Bryan Hogan — Making a Website with Obsidian](https://bryanhogan.com/blog/obsidian-website) — the best head-term shape: answer-first opening, immediate method/complexity/cost table (Publish / Quartz / Astro Starlight / Custom), then 2–4 paragraphs per method, live example sites as proof. ~700–800 words.
- [Obsidian Help — Set up Obsidian Publish](https://help.obsidian.md/publish/setup) — the official incumbent; our page must acknowledge it as method #1 to be credible.
- [dev.to — Host your Obsidian notebook on GitHub Pages for free](https://dev.to/defenderofbasic/host-your-obsidian-notebook-on-github-pages-for-free-8l1) — dev.to ranks fast for dev how-tos; also the cross-post channel per growth-plan Step 4.

**Why they rank.** Bryan Hogan covers *all* methods for a query where every other page covers one; the options table matches "which way should I do this" intent exactly.

**trip2g angle.** Write it as an honest survey (Publish → Quartz → Hugo/Astro → trip2g), each with complexity/cost/best-for, then a full walkthrough of the trip2g path (install → connect vault → sync → live site) with a verification step. Favorable framing = trip2g is the only "live server" row (two-way sync, paywalls, MCP, forms); honest framing = it's also the only row that requires running a server (or the free cloud instance).

#### 7. `digital-garden-self-hosted.md` — "digital garden self-hosted", "obsidian digital garden"

**Models.**
- [Ness Labs — How to set up your own digital garden](https://nesslabs.com/digital-garden-set-up) — the canonical intro that owns the concept query: defines digital garden, then no-code vs technical paths.
- [Maggie Appleton — Digital Gardening for Non-Technical Folks](https://maggieappleton.com/nontechnical-gardening) — the community's most-linked author on the topic; audience-segmented structure (by skill level) worth copying.
- [Flowershow — Self publish your digital garden](https://flowershow.app/docs/self-hosted/publish-tutorial) — a docs-site tutorial ranking for the "self-hosted" modifier, i.e. exactly our slot.

**Searcher intent.** Concept-first, then tool-shopping; higher-funnel than the pages above.

**trip2g lead-with.** Gardens are tended continuously — two-way sync makes editing *be* publishing; monetization lets a garden have paid beds (unique among garden tools); RSS + Telegram give the garden distribution; agents can tend it over MCP.

**Where competitors legitimately win.** Static generators + GitHub Pages = $0 and zero ops (the default recommendation in every model article); Obsidian Publish = fastest no-code path. Position trip2g for the gardener who wants the garden to *do* things.

### Lane B — MCP / agent memory

#### 8. `mcp-memory-server.md` hub — "MCP server memory", "persistent memory MCP server", "self-hosted agent memory"

**Models.**
- [ChatForest — Best Memory & Knowledge MCP Servers](https://chatforest.com/guides/best-memory-mcp-servers/) and [Atlan — Best AI Agent Memory Frameworks](https://atlan.com/know/best-ai-agent-memory-frameworks-2026/) — the roundup shape that currently owns these SERPs: ranked list, per-tool capsule, "no single best — depends on use case" verdict.
- [PulseMCP directory (memory query)](https://www.pulsemcp.com/servers?q=memory) — directories rank for these terms; being listed (growth plan Step 3) matters as much as the page.
- [Mem0 — OpenMemory](https://mem0.ai/openmemory) — vendor landing that ranks by owning a sub-category name; supports our "own a dimension" move.

**Why they rank.** Thin competition (per `seo_plan.md` §1), exact-match titles, scannable capsule-per-server format.

**trip2g lead-with.** The memory is human-readable markdown notes — auditable, editable in Obsidian, versioned (`note_versions` + git mirror); token-efficient retrieval (search → expand → read_section; the 15–37× benchmark on `token-economy-bench.md`); SQLite single binary, no vector-DB stack; the same memory is a browsable website; federation fans a memory query across hubs.

**Where competitors legitimately win.** Purpose-built memory servers (ai-memory, agentmemory, OMEGA, Engram per the roundups) do automatic capture/scoring/promotion with zero authoring discipline; Mem0 cloud is zero-setup. trip2g memory is deliberate notes, not automatic conversation capture — say that; it's also the feature for users who *want* to read their agent's memory.

#### 9. `claude-code-persistent-memory.md` — "Claude Code memory MCP", "claude code persistent memory"

**Models.**
- [Mem0 — Add Persistent Memory to Claude Code (5-Minute Setup)](https://mem0.ai/blog/claude-code-memory) — the benchmark vendor tutorial (dissected in TL;DR): friction intro, 3 install paths, verification, 8-Q FAQ, ~2,500 words.
- [dev.to — How to give Claude Code persistent memory with a self-hosted mem0 MCP server](https://dev.to/n3rdh4ck3r/how-to-give-claude-code-persistent-memory-with-a-self-hosted-mem0-mcp-server-h68) — ~3,500 words; "self-hosted" appears 12+ times (deliberate modifier density); opens with a real debugging scene; discloses the LLM-call cost of graph mode and mitigations. Our closest intent match.
- [MemPalace — How to Add Persistent Memory to Claude Code](https://www.mempalace.tech/blog/add-memory-to-claude-code) — small-vendor page ranking on exact-match title + year, proving the term is winnable.

**Searcher intent.** Claude Code user tired of re-explaining the project every session. One of the top community questions (per `seo_plan.md`).

**trip2g lead-with.** One MCP URL (`/_system/mcp`) — no local process per machine; memory shared across machines/teammates (subscription-scoped); memcli/quickstart; memory you can open in Obsidian and edit; token-economy numbers as the differentiator stat (the dev.to model shows cost disclosure is a feature).

**Where competitors legitimately win.** Anthropic's official memory MCP is free, local, zero-account; Mem0 auto-extracts facts without the agent being told to write notes; MemPalace stores conversations verbatim. If the user wants fully-local single-machine memory, the official server is simpler — recommend it for that case.

#### 10. `obsidian-mcp-server.md` — "Obsidian MCP server"

**Models.**
- [MCPBundles — Obsidian MCP Server: Connect Your AI to Your Vault](https://www.mcpbundles.com/blog/obsidian-mcp-vault-ai) — ~2,800 words; hook = "most Obsidian MCP servers give you six tools… this one does things the others can't"; 7×4 feature matrix vs three named competitors; 3-step setup; 6-question FAQ. The best vendor model in this niche.
- [Awesome Claude — 3 Ways to Use Obsidian with Claude Code](https://awesomeclaude.ai/how-to/use-obsidian-with-claude) — ~800 words; three methods (MarkusPfundstein REST-API server / Anthropic Filesystem MCP / cyanheads plugin) ranked by GitHub stars with a stars/best-for/pros-cons table. The neutral survey shape.
- [MarkusPfundstein/mcp-obsidian](https://github.com/MarkusPfundstein/mcp-obsidian) — the incumbent README that actually ranks; whatever we write must position against it accurately.

**Searcher intent.** Obsidian user wanting Claude/agents to read (and write) the vault.

**trip2g lead-with.** The existing servers all require the laptop on and Obsidian (or a local process) running; trip2g is the *hosted* variant — vault synced to a hub, agents query it from anywhere, access scoped by subscription/ACL, plus hybrid full-text + semantic search and version history. Also the honest "4th way" framing: extend Awesome Claude's 3-way survey with the server-hosted option.

**Where competitors legitimately win.** Fully local (nothing leaves the machine), no server to run, 2-minute setup, direct file access. For a single user on one machine, a local MCP server is genuinely simpler — recommend it for that case and link it.

#### 11. `trip2g-vs-mem0-vs-memgpt.md` — agent-memory comparison shoppers

**Models.**
- [Particula Tech — Agent Memory Frameworks Tested](https://particula.tech/blog/agent-memory-frameworks-tested-mem0-zep-letta-cognee-2026) — the gold standard (see TL;DR): decision matrix, benchmark citations, per-tool "pick when", named weaknesses, "run your own eval" advice.
- [TokenMix — Mem0 vs Letta vs MemGPT](https://tokenmix.ai/blog/ai-agent-memory-mem0-vs-letta-vs-memgpt-2026) and [dev.to — AI Agent Memory in 2026](https://dev.to/agdex_ai/ai-agent-memory-in-2026-mem0-vs-zep-vs-letta-vs-cognee-a-practical-guide-cfa) — the crowded "vs" field; note several pieces already add a small newcomer to the title (e.g. [Rohit Raj adds MemPalace](https://rohitraj.tech/en/notes/open-source-ai-agent-memory-mem0-vs-zep-letta-2026)) — the exact move for us: "Mem0 vs Letta vs Zep vs trip2g".
- Architecture one-liners to reuse from the SERP consensus: "Mem0 is a memory layer you bolt on; Letta is a runtime where the agent is its memory; Zep builds a temporal knowledge graph."

**Searcher intent.** Developer choosing memory infrastructure; comparison-shopping with benchmarks.

**trip2g lead-with.** A fourth architecture: "memory as a markdown knowledge base" — human-readable/editable, git-versioned, served to humans as a website and to agents over MCP; no extraction pipeline or vector DB to operate; token-economy benchmark as our quantifiable stat (the genre runs on numbers — LongMemEval cites are what make Particula's piece memorable).

**Where competitors legitimately win.** Mem0: automatic fact extraction, huge community (~47k stars), managed cloud. Letta/MemGPT: agent self-managed memory hierarchy, research pedigree. Zep: temporal reasoning / point-in-time correctness (Graphiti, 63.8% LongMemEval). trip2g doesn't auto-extract from conversations and has no LongMemEval score — state both; our "pick trip2g when" line: *you want memory you can read, edit in Obsidian, and publish.*

---

## Per-target summary table

| Target keyword | Best model URL(s) | Why it ranks | trip2g honest-favorable angle |
|---|---|---|---|
| obsidian publish alternative | [unmarkdown.com](https://unmarkdown.com/blog/obsidian-publish-alternatives) · [ssp.sh](https://www.ssp.sh/brain/open-source-obsidian-publish-alternatives/) | matrix + reframe + admitted gaps; freshness + live-garden dogfooding | lead: self-hosted, Jet templates, paywalls, two-way sync, $0 · concede: Publish = zero setup, official, graph view |
| quartz alternative / vs quartz | [hamatti.org](https://notes.hamatti.org/technology/building-a-digital-garden-with-obsidian-and-quartz) · [ssp.sh quartz](https://www.ssp.sh/brain/quartz-publish-obsidian-vault/) · [XDA](https://www.xda-developers.com/turned-obsidian-vault-into-website/) | first-person friction; topical authority; DA + benefit hook | lead: no build loop, live sync, paywalls/forms, MCP · concede: Quartz = free hosting, no server, big community, graph view |
| publish obsidian vault hugo | [jacobian.org](https://jacobian.org/til/hugo-obsidian/) · [sagar.se](https://sagar.se/notes/computers/hugo/digital-garden/publishing-obsidian-vault-with-hugo/) | author authority + candid pain catalog | lead: deletes the convert/deploy pipeline, native wikilinks, git endpoint kept · concede: Hugo = themes, full HTML control, free static |
| notion sites alternative self-hosted | [selfh.st](https://selfh.st/alternatives/notion/) · [docmost](https://docmost.com/blog/open-source-notion-alternatives/) | authority hub; clean per-tool capsule shape | lead: markdown you own + git out, no per-seat, Obsidian editor, forms · concede: Notion = WYSIWYG, databases, zero infra, realtime collab |
| gitbook open source alternative | [docmost](https://docmost.com/blog/gitbook-alternatives/) · [apidog](https://apidog.com/blog/gitbook-alternatives/) | primacy-slot vendor listicle; keyword coverage | lead: one binary + SQLite, Obsidian authoring, MCP-queryable docs, MIT · concede: GitBook = review workflow, OpenAPI tooling, managed hosting |
| how to publish an obsidian vault as a website | [bryanhogan.com](https://bryanhogan.com/blog/obsidian-website) · [help.obsidian.md](https://help.obsidian.md/publish/setup) · [dev.to GH Pages](https://dev.to/defenderofbasic/host-your-obsidian-notebook-on-github-pages-for-free-8l1) | covers ALL methods + upfront options table | survey all methods honestly; trip2g = the "live server" row (sync, paywalls, MCP) vs the only row needing a server |
| digital garden self-hosted | [nesslabs.com](https://nesslabs.com/digital-garden-set-up) · [maggieappleton.com](https://maggieappleton.com/nontechnical-gardening) · [flowershow tutorial](https://flowershow.app/docs/self-hosted/publish-tutorial) | concept ownership; skill-segmented paths | lead: editing = publishing (sync), monetized garden, RSS/Telegram, agents tend it · concede: static + Pages = $0 and simpler |
| MCP server memory / persistent memory MCP | [chatforest](https://chatforest.com/guides/best-memory-mcp-servers/) · [atlan](https://atlan.com/know/best-ai-agent-memory-frameworks-2026/) · [pulsemcp](https://www.pulsemcp.com/servers?q=memory) | thin competition; exact-match roundups + directories | lead: memory = readable markdown, versioned, 15–37× token bench, SQLite-only, doubles as website · concede: purpose-built servers auto-capture with zero discipline |
| claude code persistent memory | [mem0 blog](https://mem0.ai/blog/claude-code-memory) · [dev.to self-hosted mem0](https://dev.to/n3rdh4ck3r/how-to-give-claude-code-persistent-memory-with-a-self-hosted-mem0-mcp-server-h68) · [mempalace](https://www.mempalace.tech/blog/add-memory-to-claude-code) | friction intro, tiered paths, verification, FAQ; modifier density | lead: one MCP URL, cross-machine/team memory, editable in Obsidian · concede: official memory MCP simpler for single-machine local; Mem0 auto-extracts |
| obsidian mcp server | [mcpbundles](https://www.mcpbundles.com/blog/obsidian-mcp-vault-ai) · [awesomeclaude](https://awesomeclaude.ai/how-to/use-obsidian-with-claude) · [MarkusPfundstein repo](https://github.com/MarkusPfundstein/mcp-obsidian) | competitor matrix + FAQ; neutral 3-way survey | frame as "the 4th way": server-hosted vault, ACL-scoped, hybrid search, works with laptop off · concede: local servers = fully private, simpler for one machine |
| mem0 vs memgpt (agent memory comparison) | [particula.tech](https://particula.tech/blog/agent-memory-frameworks-tested-mem0-zep-letta-cognee-2026) · [tokenmix](https://tokenmix.ai/blog/ai-agent-memory-mem0-vs-letta-vs-memgpt-2026) · [rohitraj (adds self to title)](https://rohitraj.tech/en/notes/open-source-ai-agent-memory-mem0-vs-zep-letta-2026) | benchmarks + "pick when" verdicts + named weaknesses | join the title ("… vs trip2g"); 4th architecture: memory-as-markdown-KB, token bench as our number · concede: Mem0 auto-extraction/community, Zep temporal, Letta self-managed memory |

---

## Credibility guardrails

1. **Competitors must win visible rows.** Every dissected high performer (Unmarkdown, Particula, jacobian, hamatti) earns trust by conceding; the one that hides it (Docmost) reads as advertorial even while ranking. Concessions are listed per target above — keep them in the shipped pages.
2. **Only shipped features.** README status column is the source of truth: fleet/agentruntime is **branch, not main** — it may appear as an outlook sentence, never as a comparison-table row. Same for read replicas.
3. **Cite competitor claims from their own docs/pricing pages** (e.g. [help.obsidian.md](https://help.obsidian.md/publish/setup), Quartz repo, Mem0 docs) — never from our paraphrase. Numbers dated ("as of 2026-07").
4. **"Pick X when" for every tool including ours** — recommendation-by-constraint, not "best overall".
5. **Our numbers must be reproducible**: lead with the token-economy benchmark only with the script link; no invented benchmark scores against LongMemEval-measured tools.
6. **Freshness discipline**: visible updated-date, and a quarterly re-check that competitor prices/features still hold (stale claims are the fastest credibility leak in this genre).
7. House rules: bilingual EN/RU pairs, `description:` frontmatter, answer-first lead, wikilinks into existing docs (`two-way-sync`, `mcp`, `monetization`, `templates`, `token-economy-bench`).

## All sources

Comparison/alternative models: [unmarkdown.com/blog/obsidian-publish-alternatives](https://unmarkdown.com/blog/obsidian-publish-alternatives) · [ssp.sh/brain/open-source-obsidian-publish-alternatives](https://www.ssp.sh/brain/open-source-obsidian-publish-alternatives/) · [ssp.sh/brain/quartz-publish-obsidian-vault](https://www.ssp.sh/brain/quartz-publish-obsidian-vault/) · [forum.obsidian.md/t/obsidian-publish-alternatives/22886](https://forum.obsidian.md/t/obsidian-publish-alternatives/22886) · [docmost.com/blog/gitbook-alternatives](https://docmost.com/blog/gitbook-alternatives/) · [docmost.com/blog/open-source-notion-alternatives](https://docmost.com/blog/open-source-notion-alternatives/) · [apidog.com/blog/gitbook-alternatives](https://apidog.com/blog/gitbook-alternatives/) · [alternativeto.net (gitbook, open source)](https://alternativeto.net/software/gitbook/?license=opensource) · [selfh.st/alternatives/notion](https://selfh.st/alternatives/notion/) · [openalternative.co/alternatives/notion](https://openalternative.co/alternatives/notion) · [xda-developers.com/flowershow-is-fantastic-free-alternative-obsidian-publish](https://www.xda-developers.com/flowershow-is-fantastic-free-alternative-obsidian-publish/)

How-to models: [bryanhogan.com/blog/obsidian-website](https://bryanhogan.com/blog/obsidian-website) · [jacobian.org/til/hugo-obsidian](https://jacobian.org/til/hugo-obsidian/) · [notes.hamatti.org (obsidian + quartz)](https://notes.hamatti.org/technology/building-a-digital-garden-with-obsidian-and-quartz) · [xda-developers.com/turned-obsidian-vault-into-website](https://www.xda-developers.com/turned-obsidian-vault-into-website/) · [help.obsidian.md/publish/setup](https://help.obsidian.md/publish/setup) · [dev.to (GH Pages)](https://dev.to/defenderofbasic/host-your-obsidian-notebook-on-github-pages-for-free-8l1) · [sagar.se (hugo)](https://sagar.se/notes/computers/hugo/digital-garden/publishing-obsidian-vault-with-hugo/) · [github.com/devidw/obsidian-to-hugo](https://github.com/devidw/obsidian-to-hugo) · [nesslabs.com/digital-garden-set-up](https://nesslabs.com/digital-garden-set-up) · [maggieappleton.com/nontechnical-gardening](https://maggieappleton.com/nontechnical-gardening) · [flowershow.app self-hosted tutorial](https://flowershow.app/docs/self-hosted/publish-tutorial)

MCP/memory models: [mem0.ai/blog/claude-code-memory](https://mem0.ai/blog/claude-code-memory) · [dev.to (self-hosted mem0 MCP)](https://dev.to/n3rdh4ck3r/how-to-give-claude-code-persistent-memory-with-a-self-hosted-mem0-mcp-server-h68) · [mempalace.tech/blog/add-memory-to-claude-code](https://www.mempalace.tech/blog/add-memory-to-claude-code) · [mcpbundles.com/blog/obsidian-mcp-vault-ai](https://www.mcpbundles.com/blog/obsidian-mcp-vault-ai) · [awesomeclaude.ai/how-to/use-obsidian-with-claude](https://awesomeclaude.ai/how-to/use-obsidian-with-claude) · [github.com/MarkusPfundstein/mcp-obsidian](https://github.com/MarkusPfundstein/mcp-obsidian) · [particula.tech (memory frameworks tested)](https://particula.tech/blog/agent-memory-frameworks-tested-mem0-zep-letta-cognee-2026) · [tokenmix.ai (mem0 vs letta vs memgpt)](https://tokenmix.ai/blog/ai-agent-memory-mem0-vs-letta-vs-memgpt-2026) · [dev.to (agdex practical guide)](https://dev.to/agdex_ai/ai-agent-memory-in-2026-mem0-vs-zep-vs-letta-vs-cognee-a-practical-guide-cfa) · [rohitraj.tech (adds MemPalace to title)](https://rohitraj.tech/en/notes/open-source-ai-agent-memory-mem0-vs-zep-letta-2026) · [atlan.com (memory frameworks ranked)](https://atlan.com/know/best-ai-agent-memory-frameworks-2026/) · [chatforest.com (best memory MCP servers)](https://chatforest.com/guides/best-memory-mcp-servers/) · [pulsemcp.com (memory servers)](https://www.pulsemcp.com/servers?q=memory) · [mem0.ai/openmemory](https://mem0.ai/openmemory)
