# SEO + GitHub stars growth plan

Date: 2026-07-02. Read-only research; grounded in `docs/dev/seo_audit.md`, `docs/dev/seo_plan.md`, `docs/dev/2026-07-02_user_docs_humanization_plan.md`, the engine code, and web research (cited inline).

## TL;DR

**The one guaranteed AI-solo move:** publish a batch of ~15 keyword-targeted comparison + how-to pages on the docs site itself (Step 1), backed by the last five engine-level SEO fixes (Step 2). The docs site runs on trip2g, so every page is both content that ranks and a live demo of the product — and an AI can ship the whole batch this week with no human in the loop.

**The honest ceiling:** an AI solo can reliably deliver *indexable, structurally sound, keyword-targeted pages* and a *conversion-ready GitHub surface*. That earns long-tail organic traffic over 2–6 months in low-competition niches (MCP memory, "Obsidian publish alternative") — realistically tens to low hundreds of visits/day, not thousands. It cannot guarantee stars: star spikes come from launch-day distribution (HN, Reddit, X), which needs a human account, human timing, and a human replying in the comments. Data point: HN exposure averages ~121 stars in 24h and ~1.4 stars per upvote ([arxiv 2511.04453](https://arxiv.org/abs/2511.04453)) — the AI can fully prepare that launch (Step 4) but cannot fire it. Order of operations: AI builds the compounding asset (Steps 1–3), human spends ~2 hours pressing the distribution buttons (Steps 4–5).

---

## Grounding: where trip2g's SEO stands today

Verified against code on 2026-07-02 (the June audit is partially stale — several P0s have since shipped):

| Capability | Status | Evidence |
|---|---|---|
| Server-side rendering, cached | shipped | whole pipeline; `docs/en/user/seo.md` |
| `<link rel="canonical">` | **shipped since audit** | `views.html:35-38` (reuses `og:url`) |
| JSON-LD (Article + BreadcrumbList, noindex-aware) | **shipped since audit** | `internal/defaulttemplate/jsonld.go`, `jsonld.html` |
| hreflang EN/RU in `<head>` | shipped | `endpoint.go` `buildHrefLangs` |
| sitemap.xml, multi-domain | shipped | `internal/sitemap/sitemap.go` |
| Meta description | **frontmatter-only, no fallback** | `endpoint.go:67` — only `resp.Note.Description` |
| robots.txt `Sitemap:` pointer | **missing** | grep `Sitemap:` in `cmd/`+`internal/` = 0 hits |
| `og_image` frontmatter key | **missing** (docs claim it exists) | `seo_audit.md` §4; grep = 0 hits |
| sitemap `xhtml:link hreflang` alternates | missing | `seo_audit.md` §5 |
| `llms.txt` | **missing** | grep `llms` in `cmd/`+`internal/` = 0 hits |
| SoftwareApplication JSON-LD on home | missing | only Article/Breadcrumb emitted |

Positioning context (from the humanization plan): three surfaces say three different things — README says "Markdown Operating System", the live landing says "Join the network", the docs home says "publish your Obsidian vault as a website". SEO content must pick lanes deliberately (see Step 1) rather than inherit the split.

One more asset already in the vault: RU docs have comparison/integration pages (Quartz, WordPress, Obsidian Publish, Confluence, Super.so) **that EN lacks** — free EN backlog for Step 1.

---

## Step 1 — [AI-solo] Ship the SEO content batch on the docs site (dogfooding)

**The single highest-leverage thing an AI can ship this week.**

**Action.** Author ~15 pages under `docs/en/user/` (+ RU pairs, house rule), each with `description:` frontmatter, answer-first lead, an honest comparison table, and wikilinks into existing docs. Sync with `obsidian-sync`; sitemap/canonical/JSON-LD/hreflang apply automatically because the engine already ships them — this is the dogfooding loop: the pages demonstrate the product *and* rank.

**Two keyword lanes** (resolves the positioning split instead of fighting it):

*Lane A — Obsidian publishing (big audience, medium competition):*
- `obsidian-publish-alternative.md` — "trip2g vs Obsidian Publish" (backport + expand the existing RU page). Target: `obsidian publish alternative`, `obsidian publish self-hosted`.
- `trip2g-vs-quartz.md` — vs Quartz (RU exists). Target: `quartz alternative`, `obsidian digital garden self-hosted`. Honest angle: Quartz is static/free; trip2g adds paywalls, live sync, MCP.
- `trip2g-vs-hugo.md` — Target: `publish obsidian vault hugo` (a real pain workflow trip2g replaces).
- `trip2g-vs-notion-sites.md`, `trip2g-vs-gitbook.md` — Target: `notion sites alternative self-hosted`, `gitbook open source alternative`.
- `publish-obsidian-vault-as-website.md` — the head how-to. Target: `how to publish an obsidian vault as a website` (top-of-funnel intent; the docs home already makes this promise).

*Lane B — MCP / agent memory (low competition, fastest to rank — full keyword table already in `docs/dev/seo_plan.md` §1):*
- `mcp-memory-server.md` hub page. Target: `MCP server memory`, `persistent memory MCP server`, `self-hosted agent memory`. Links agent-memory.md, llm-wiki.md, mcp.md, federation.md.
- `claude-code-persistent-memory.md` — tutorial. Target: `Claude Code memory MCP` (per seo_plan.md, a top community question).
- `obsidian-mcp-server.md` — Target: `Obsidian MCP server` (perfect-fit intent, thin competition).
- `trip2g-vs-mem0-vs-memgpt.md` — agent-memory comparison shoppers.

*Programmatic layer:* one use-case page per shipped capability that lacks an intent page — `telegram-blog-from-obsidian.md`, `sell-obsidian-notes.md` (monetization), `team-knowledge-base-mcp.md`, `self-hosted-wiki-sqlite.md`. Frontmatter widgets (`magazine_include_files`) can auto-index them — a small programmatic-SEO demo in itself ([pSEO guide](https://rankmehigher.co/learn/programmatic-seo-guide/)).

**Why it works.** Comparison ("X vs Y", "X alternative") pages catch bottom-of-funnel searchers already shopping; low-competition MCP terms can rank in weeks (seo_plan.md §1 rationale). Every page is served by trip2g itself, so it's also proof.

**Impact / timeframe.** Indexed in days; MCP-lane rankings in 2–8 weeks; Obsidian-lane in 2–6 months. Expect long-tail trickle, not a flood — but it compounds and every page links the GitHub repo (stars via qualified visitors).

**Guardrails.** Honest tables only (competitors win rows too — the existing `seo.md` tone is the model); no vapor: only shipped features (README status column is the source of truth); bilingual pairs; run `lint docs` after.

## Step 2 — [AI-solo] Close the last five engine SEO gaps + `llms.txt`

**Action.** Five small engine changes (each benefits *every* trip2g site — product feature, not just marketing):
1. robots.txt: append `Sitemap: {publicURL}/sitemap.xml` to the `opened` default (`cmd/server/main.go` `handleRobotsTxt`).
2. Meta-description fallback: first paragraph via `PartialRenderer().Introduce()` (already used at `views.html:380`), truncate ~155 chars, in `rendernotepage`. Kills the audit's biggest remaining P0.
3. `og_image:` frontmatter key — currently documented but **ignored** (docs actively mislead; `seo_audit.md` §4). Implement the key.
4. Sitemap `xhtml:link hreflang` alternates for EN/RU pairs (`internal/sitemap/sitemap.go`).
5. `/llms.txt` endpoint (or a plain note routed there) — generate from public notes: site summary + key-page list, per [llmstxt.org](https://llmstxt.org). Draft content already written in `seo_plan.md` §5. Add `SoftwareApplication` JSON-LD on the home page (extend `jsonld.go`, pattern established).

**Why it works.** Crawl discovery (robots pointer), snippet quality on every page without `description:` (most pages today), correct social previews, no EN/RU signal-splitting, and AI-crawler citability — the target audience researches tools in ChatGPT/Perplexity, so `llms.txt` + consistent factual phrasing is unusually high-ROI here ([LLM visibility signal set](https://www.sona.com/blog/best-llm-seo-tracker-tools-in-2026-compared)).

**Impact / timeframe.** 1–2 days of work; effects land with the next crawl cycle (1–4 weeks). Multiplies Step 1: content can't earn snippets/citations without this plumbing.

## Step 3 — [AI-solo, merge-dependent parts AI-assisted] GitHub surface: make the repo convert visitors it already gets

**Action.**
- **README hero:** keep "Markdown Operating System" as the technical hook but add the plain promise + primary keywords in the first two lines ("publish your Obsidian vault as a website · self-hosted MCP memory for AI agents"), per the humanization plan's positioning verdict. Add a ~15s GIF (sync in Obsidian → page live) above the fold — day-1 README polish is where star conversion happens ([HN diffusion study](https://arxiv.org/abs/2511.04453): "polish before you post"; [ToolJet stars guide](https://blog.tooljet.com/github-stars-guide/)).
- **Repo topics** (one settings change, AI-executable via `gh api`): `obsidian`, `obsidian-publish`, `mcp-server`, `ai-agent-memory`, `self-hosted`, `knowledge-base`, `markdown`, `digital-garden`. Topics feed GitHub search + Explore.
- **CONTRIBUTING.md + 5–10 `good first issue` labels** via `gh` — contributor-readiness is a ranked trust signal and awesome-list acceptance criterion.
- **Submissions as PRs** (AI writes and opens them; *merges* are at maintainers' discretion → that part is AI-assisted): `awesome-mcp-servers`, `awesome-selfhosted`, `awesome-obsidian`, `awesome-ai-agents`; MCP directories (`mcpservers.org`, `mcp.so`, `pulsemcp.com`) — per seo_plan.md §4 the single highest-leverage off-page action, since directories already rank for "MCP memory server" and are high-authority backlinks.

**Why it works.** Steps 1–2 route searchers to the repo; the repo must convert in 10 seconds (GIF + plain promise) or the click is wasted. Awesome lists + directories are the only backlink channel with a submission process an AI can drive end-to-end.

**Impact / timeframe.** README/topics/issues: this week, permanent. List inclusions: days–weeks per maintainer; each accepted listing is a durable backlink + steady star trickle (single-digit weekly, compounding).

## Step 4 — [AI-assisted] The launch kit: AI prepares 100%, a human presses the button

**Action.** AI produces, human fires:
- **Show HN draft** around the strongest verifiable asset — the token-economy benchmark (reproducible script; opinionated + verifiable is ideal HN material, seo_plan.md §3). Title A/B options, first comment (honest limitations + architecture), FAQ crib sheet for the thread.
- **Reddit posts** tailored per subreddit: r/ObsidianMD (two-way sync), r/selfhosted (`docker compose up`, SQLite, MIT), r/LocalLLaMA (self-hosted MCP memory + benchmark).
- **Social share images** with the headline stat; **dev.to cross-posts** of the Step-1 comparison articles with canonical URLs back to trip2g.com (dev.to ranks fast for dev queries and the canonical passes the equity home).

**Why human-gated.** HN/Reddit accounts, posting time, and — decisive — author presence in the comments are human. The data says timing matters and most impact is day-1 ([arxiv 2511.04453](https://arxiv.org/abs/2511.04453); [Show HN by the numbers](https://danfking.github.io/blog/2026/04/23/show-hn-by-the-numbers/) — the "Show HN" tag itself confers no advantage after controls, so substance > format). Expected value if it lands: ~100–300 stars in 48h at ~1.4 stars/upvote. **Sequence rule: do not fire until Steps 1–3 are live** — launch traffic hitting an unpolished README is spent once and wasted.

**Impact / timeframe.** Kit ready in 2 days; launch = ~2 human hours; the largest single star event available.

## Step 5 — [human-required] Recurring presence and relationships

**Action** (lowest priority precisely because no AI can guarantee it): an X/Twitter build-in-public presence (the benchmark thread; Karpathy's LLM-wiki audience is the natural niche); answering memory/publishing questions in the MCP and Obsidian Discords with doc links; an Obsidian community-plugin listing for the sync plugin (very high-authority backlink, but a human-owned review process); outreach to 5–10 AI-agent-architecture bloggers with the benchmark data; never ask for stars directly (violates GitHub ToS and reads as spam).

**Why last.** These channels drive the biggest long-run numbers for OSS dev tools, but they are irreducibly human: trust, timing, voice. The AI's role is feeding them — drafts, data, replies to review — not executing them.

**Impact / timeframe.** Months; compounding; the difference between a 500-star and a 5,000-star repo lives here, not in on-site SEO.

---

## What "guaranteed" honestly means

| AI can guarantee | AI cannot guarantee |
|---|---|
| Pages exist, are indexable, structurally sound (canonical/JSON-LD/hreflang/descriptions) | That Google ranks them above incumbents for contested terms |
| Engine SEO gaps closed for every trip2g site | Backlinks beyond PR-based list submissions |
| README/topics/issues conversion-ready; launch kit written | An HN front page, upvotes, or comment-thread goodwill |
| Long-tail impressions in low-competition niches (MCP lane) within weeks | A star spike without human distribution |

Measurement (AI-solo, ongoing): Google Search Console + Bing Webmaster (Bing feeds several AI search products), star-history.com snapshots, a monthly check that Step-1 pages hold their target-query positions.

Sources: [arxiv 2511.04453 — HN impact on GitHub stars](https://arxiv.org/abs/2511.04453) · [Show HN by the numbers](https://danfking.github.io/blog/2026/04/23/show-hn-by-the-numbers/) · [ToolJet GitHub stars guide](https://blog.tooljet.com/github-stars-guide/) · [llmstxt.org](https://llmstxt.org) · [Programmatic SEO guide 2026](https://rankmehigher.co/learn/programmatic-seo-guide/) · [LLM visibility signals](https://www.sona.com/blog/best-llm-seo-tracker-tools-in-2026-compared) · internal: `docs/dev/seo_audit.md`, `docs/dev/seo_plan.md`, `docs/dev/2026-07-02_user_docs_humanization_plan.md`
