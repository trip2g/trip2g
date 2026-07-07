# Launch kit — Show HN + Reddit + dev.to (drafts, human fires)

Date: 2026-07-02. Implements Step 4 of `docs/dev/2026-07-02_seo_growth_plan.md`. Everything here is a draft: **no post goes out until a human reads it, owns it, and presses the button.** Grounded in `README.md`, `docs/dev/2026-07-02_seo_content_research.md`, and the token-economy benchmark (`docs/dev/token_economy_bench.md`, `docs/en/user/token-economy-bench.md`).

## TL;DR

The strongest verifiable hook is the token-economy benchmark: **median 15× fewer tokens to read the answer vs dumping the whole note (up to 37×), measured on the live public endpoint, reproducible by anyone with a 50-line zero-dependency Python script.** Lead the HN first comment and the r/LocalLLaMA post with it. Titles are plain one-liners, no superlatives. Every post names its own limitations before commenters do — in every successful launch we dissected, the honesty *was* the conversion mechanism. Do not fire until Steps 1–3 of the growth plan (content batch, engine SEO, README GIF + topics) are live. Realistic outcome if it lands: ~100–300 stars in 48h; the "Show HN" tag itself confers no advantage — substance does ([Show HN by the numbers](https://danfking.github.io/blog/2026/04/23/show-hn-by-the-numbers/), [arxiv 2511.04453](https://arxiv.org/abs/2511.04453)).

---

## 0. Research: what made comparable launches land

Posts studied (points/comments from HN Algolia, 2026-07-02):

| Post | Result | What made it land |
|---|---|---|
| [Show HN: Obsidian – A knowledge base that works on local Markdown files](https://news.ycombinator.com/item?id=23324598) (2020) | 1087 pts / 477 c | Title is a plain factual one-liner. "Local markdown files" — the data-ownership promise — is *in the title*, not in the body. |
| [Show HN: Open-source obsidian.md sync server](https://news.ycombinator.com/item?id=37247767) (2023) | 416 pts / 152 c | Radical honesty: "still missing a few features… highly inefficient… I'm probably violating the TOS, and I'm sorry." A high-school grad who couldn't afford $8/mo. HN rewarded the candor and the underdog story, not polish. |
| [Show HN: Plausible – Self-Hosted Google Analytics alternative](https://news.ycombinator.com/item?id=24696145) (2020) | 351 pts / 139 c | Maker comment shape: proof numbers first ("battle-tested on 5,000+ sites, 180M page views"), then the exact tech stack (Elixir/Phoenix, Postgres, Clickhouse, Tailwind, React), one concrete stat (0.7 KB script), then "public roadmap, would love feedback". No adjectives, all nouns and numbers. |
| [Show HN: OpenWorkers – Self-hosted Cloudflare workers in Rust](https://news.ycombinator.com/item?id=46454693) (2026) | 500 pts / 158 c | "Self-hosted [thing you already pay for] in [language HN likes]" title shape still works in 2026. |
| [Launch guides: markepear](https://www.markepear.dev/blog/dev-tool-hacker-news-launch), [lucasfcosta](https://www.lucasfcosta.com/blog/hn-launch), [daily.dev](https://business.daily.dev/resources/hacker-news-marketing-developer-tools-show-hn-launch-day-sustained-coverage/), [syften](https://syften.com/blog/hacker-news-marketing/) | — | Consensus: title = `Show HN: Name – plain one-sentence description`; no superlatives; link the GitHub repo, not a landing page; maker comment within minutes (problem → backstory → stack → one honest limitation → invite feedback); reply to every comment in the first 60–90 min; never solicit upvotes or seed booster comments (rankings penalize voting rings); Tue–Thu 9:00–12:00 ET. |

Extracted rules applied below:

1. **Title states what it is, in nouns.** "Works on local Markdown files" beat every clever framing. Avoid "Markdown OS" in titles — it's our internal metaphor, and on HN unexplained metaphors read as marketing.
2. **First comment leads with a number someone can check.** Plausible's 5,000 sites; ours is the 15× benchmark with a script.
3. **Name a real limitation before the thread does.** The 416-point sync server *apologized in the post*. Our list is in §1.3.
4. **Repo link as the submission URL.** HN devs upvote things they can run; a landing page reads as a funnel.
5. **The author must live in the thread.** This is the human's irreducible 2 hours (§5).

---

## 1. Show HN

Submission URL: `https://github.com/trip2g/trip2g` (the repo, not trip2g.com — per rule 4; README must have the GIF by then, growth plan Step 3).

### 1.1 Title — three A/B options

- **A (data-ownership shape, mirrors the 1087-pt Obsidian title):**
  `Show HN: Trip2g – Self-hosted server that turns an Obsidian vault into a website and MCP server`
- **B (self-hosted + stack shape, mirrors OpenWorkers/Plausible):**
  `Show HN: Trip2g – Publish markdown notes to humans (web) and agents (MCP) – Go/SQLite/MIT`
- **C (benchmark-forward, riskier but the most differentiated):**
  `Show HN: Trip2g – MCP memory as plain markdown notes (15x fewer tokens than full-note reads)`

Recommendation: **A**. It names the audience's noun (Obsidian vault) and the two outputs. B if the human wants the stack in the title; C only if launching primarily to the agent-infra crowd. HN allows one shot per project every few months — pick one, don't actually A/B.

### 1.2 Post body

HN Show HN submissions with a URL have no body; the body below goes in the text field only if submitting as a text post — otherwise fold it into the first comment. Plan for URL submission + first comment.

### 1.3 First comment (the maker comment — post within 5 minutes)

> Hi HN, author here.
>
> trip2g is a self-hosted Go server (single binary, SQLite, MIT) that serves one set of markdown notes two ways: to humans as a website, to AI agents over MCP. An Obsidian vault syncs into it two-way — edit locally, the page is live in about a second; edit through the API, it flows back to your vault. Every edit is versioned and mirrored to git, so the vault is also `git clone`-able over Smart HTTP.
>
> The part I'd most like you to poke at is the retrieval economics. Search results carry a `toc_path` pointing at the exact section, so an agent reads ~200 tokens instead of the whole note. On the live docs vault that's a median 15× saving (up to 37× on long notes) versus loading the full note. It's measured against the public endpoint with a zero-dependency Python script you can run yourself in under a minute: https://trip2g.com/en/user/token-economy-bench — the script is on the page. To be fair about the baseline: 15× is against the naive whole-note dump, not against a tuned RAG pipeline. And the token saving turned out to be the easy half — deterministic section-picking failed on 8 of 9 real queries in our tests; landing on the *right* section is a navigation problem, which is why search returns a precise path instead of "somewhere in this file".
>
> Stack: Go + FastHTTP, gqlgen, SQLite (+ Litestream for streaming backup), bleve for full-text, Goldmark for markdown, embeddings via any OpenAI-compatible API for semantic search. Hubs can federate: one MCP query fans out to peer hubs with HMAC-signed, depth-limited calls.
>
> Honest limitations:
> - Semantic search needs an external embeddings API (or any OpenAI-compatible endpoint you host). Full-text works fully offline.
> - No graph view. If that's your favorite Obsidian feature, Obsidian Publish or Quartz will make you happier.
> - The note-triggered agent runtime you'll see in the README table is on a branch, not main. The table marks shipped vs branch vs planned honestly — hold me to it.
> - The frontend uses $mol, a framework you've probably never heard of. It works well for us; contributing to the admin UI has a learning curve.
> - Young project, small community. You'd be an early adopter, with everything that implies.
>
> It's `docker compose up` to self-host, or there's a free cloud instance if you don't want to run a server. Happy to answer anything about the sync protocol, the federation design, or the benchmark methodology.

(Human should rewrite the first line in their own voice and add one sentence of personal backstory — why *they* built it. The 416-pt sync-server post shows the personal motive is worth real points; an AI can't write it for them.)

### 1.4 FAQ crib sheet for the thread

Prepared answers — the human rephrases, never pastes verbatim (HN notices):

| Expected question | Answer sketch |
|---|---|
| "Why not just grep the files / use the filesystem MCP?" | We measured that too: naive grep+read cost ~23× more tokens than a focused section read, because grep finds files, not section boundaries (numbers on the bench page). A skilled human narrows with `grep -n`; an agent paying per token mostly doesn't. Also grep can't reach a remote/shared/ACL'd base at all. |
| "Isn't 15× just 'read less, pay less'? Trivial." | Yes — we say so on the bench page: the saving is the easy half. The hard half is landing on the right section; deterministic selection missed 8/9. The feature is the precise `toc_path` from search plus `expand` for tree navigation, not the arithmetic. |
| "How is this different from Obsidian Publish / Quartz?" | Publish: zero-setup, official, has graph view — if you never want to run a server, pay the $8. Quartz: free static hosting, big community. trip2g is for when you need a *live* server: two-way sync, paywalls, forms, MCP, federation. It's the only option here that makes you operate a server. |
| "vs Mem0 / Letta / Zep for agent memory?" | Different architecture: memory as human-readable markdown notes you can open in Obsidian, version, and publish — not automatic conversation capture. Mem0 auto-extracts and has a huge community; Zep does temporal reasoning. We have no LongMemEval score. Pick trip2g when you want memory you can *read and edit*. |
| "Why SQLite and not Postgres?" | One process, one file, no DB server to operate; Litestream streams backups. Read/write split in the app; read replicas exist on a branch (LiteFS) for horizontal reads. Multi-tenant hubs run fine on a small VM. |
| "What's $mol?" | A small reactive TS framework (mol.hyoo.ru). Chosen for tiny bundles and a tree-based component DSL. It's the most contested dependency choice in the repo; the server has zero JS dependencies. |
| "Does the sync violate Obsidian's ToS?" | No. It doesn't touch Obsidian's sync service or protocol — it's our own plugin talking to our own server over our own API. The vault stays local; Obsidian is just the editor. |
| "Business model?" | MIT core, self-host free forever. Paid: hosted cloud instances and the built-in paywall/subscription machinery for publishers. No open-core feature gating on main. |
| "Federation security? Sounds like SSRF/loops waiting to happen." | Calls are HMAC-signed with short-expiry tokens, per-hop depth counter (`X-MCP-Federation-Depth`) enforced against `max_depth`, and each hub controls per-base access. Loops bound the way IP TTL does. Poke at it — that's why we're here. |
| "The benchmark tokenizer isn't Claude's." | Correct — it's `\w+|[^\w\s]`, stated on the page. Absolute numbers are approximate; both arms use the same ruler, so the ratio is honest. Swap in tiktoken locally if you want; the ratio barely moves. |
| "Why a Go monolith in 2026?" | One binary is the deployment story. Every capability (search, render, MCP, Telegram, git) shares one process and one SQLite file; that's what makes `docker compose up` the whole install. |
| "AGPL would protect you better." | Deliberate: MIT maximizes adoption for a young project; the moat is the hosted service and the network (federation), not the code. |

---

## 2. Reddit posts

General etiquette (all subs): post from the human's real account with history, flair correctly, disclose "I'm the author", answer every comment for the first hours, never link-and-run. **Verify each sub's current rules on the day** — the rules below are from general familiarity, not fetched live (Reddit blocks anonymous API reads), and mods change them.

### 2.1 r/ObsidianMD — angle: two-way sync

**Rules/etiquette notes:** showcase-type flair for tools/plugins; self-promo is tolerated when the post is genuinely useful to Obsidian users and the author engages; the community is allergic to "yet another AI thing" framing — lead with the vault workflow, mention MCP once. The plugin is not yet in the community-plugin directory (growth-plan Step 5) — say so plainly or the first comment will.

**Title:**
`I built a self-hosted server with true two-way sync: edit in Obsidian → live on your site in ~1s, edit on the site → flows back to your vault`

**Body:**

> My vault is the source of truth for everything I publish, and I got tired of the export→build→deploy loop (and of Publish's template limits). So I built trip2g: a self-hosted server (Go, SQLite, MIT) your vault syncs into, two-way.
>
> What it does with the vault:
> - Edit a note in Obsidian, the web page updates in about a second. Edit through the site/API, the change lands back in your local vault.
> - Wikilinks work natively (Goldmark), frontmatter drives routing, layouts, forms.
> - Every edit is versioned; the whole vault is also served over git Smart HTTP, so `git clone` is your backup.
> - Notes can publish to a Telegram channel with links kept intact, and there's an MCP endpoint if you want AI tools to search the same notes.
> - Paywalls/subscriptions if you publish paid content.
>
> Honest caveats: no graph view; you run a server (or use the free cloud instance); the sync plugin is side-loaded for now — it's not in the community plugin directory yet, that review is pending. If you want zero-ops publishing, Obsidian Publish or Quartz are the right answers and I say so in the docs comparison pages.
>
> Repo: https://github.com/trip2g/trip2g · docs run on it (dogfood): https://trip2g.com
>
> Author here, happy to answer anything about the sync protocol.

### 2.2 r/selfhosted — angle: docker compose, SQLite, MIT, no external deps

**Rules/etiquette notes:** the sub's canonical questions are always: license? docker compose? external dependencies? phone-home? resource footprint? Answer all five *in the post*. Self-promo posts by authors are accepted when flaired as such and the project is actually self-hostable. Expect "why not just use [existing wiki]" — the comparison table link is the answer.

**Title:**
`trip2g – markdown publishing hub: Obsidian vault in; website, MCP, RSS, Telegram out. One Go binary + SQLite, MIT`

**Body:**

> Author post. trip2g is a self-hosted server for markdown notes: an Obsidian vault (or plain files over git) syncs in, and the same notes come out as a website, an RSS feed, Telegram channel posts, and an MCP endpoint for AI agents.
>
> The r/selfhosted checklist up front:
> - **License:** MIT, no open-core gating on main.
> - **Install:** `docker compose up`. One process.
> - **Storage:** SQLite only — no Postgres/Redis/vector-DB stack. Litestream for streaming backup. Assets go to any S3-compatible store (MinIO works).
> - **External dependencies:** none required. Full-text search works fully offline. *Semantic* search is optional and needs an embeddings API (OpenAI-compatible — a local endpoint works).
> - **Phone-home:** none.
> - **Footprint:** runs multi-tenant on a small VM in production (that's how the public instance runs).
>
> Extras that tend to matter here: multi-domain routing per note frontmatter, magic-link/OAuth/OIDC auth, encrypted secrets (AES-256-GCM), the whole knowledge base clonable over git Smart HTTP, and hub-to-hub federation (HMAC-signed, depth-limited) if you and friends each run one.
>
> Caveats: young project; the fancy note-triggered agent runtime in the README is on a branch, not main (the README marks shipped vs branch honestly); web admin UI is functional, not gorgeous.
>
> Repo: https://github.com/trip2g/trip2g · live instance (the docs run on it): https://trip2g.com

### 2.3 r/LocalLLaMA — angle: self-hosted MCP memory + the benchmark

**Rules/etiquette notes:** the sub respects benchmarks with methodology and code, and is hostile to cloud-tethered tools — lead with self-hosted + the offline story, and be upfront that semantic search wants an embeddings endpoint (point at local OpenAI-compatible servers). Posts that read as ads get buried; posts that read as measurements get discussed. Lead with the numbers, not the product.

**Title:**
`Measured: agents reading a focused section vs the whole note — median 15× fewer tokens (up to 37×). Reproducible script, self-hosted MCP memory backed by plain markdown`

**Body:**

> I run my agents' memory as plain markdown notes on a self-hosted hub (Go + SQLite, MIT) and measured what section-level retrieval actually saves over MCP.
>
> Setup: search results carry a `toc_path` pointing at the exact heading, so the agent does `search → note_html(toc_path)` and reads ~200 tokens instead of the full note. Eight real queries against the live docs vault:
>
> | method | tokens to answer | calls |
> |---|---|---|
> | MCP, focused section | ~200 | 2 |
> | MCP, whole note | ~2,700 | 2 |
> | naive grep files + read | ~4,600 | 3 |
>
> Median saving 15.4×, best case 37× (long notes), worst 3.7× (a note that was already short). Honest baseline note: this is vs the naive whole-note dump, not vs a tuned RAG pipeline — with good chunk retrieval the gap shrinks. And token saving turned out to be the easy half: deterministic section-picking failed 8/9 queries; the real work is landing on the *right* section, which is why search returns a precise path and there's an `expand` tool for tree navigation.
>
> Reproduce it: 50-line stdlib-only Python script on the page, hits the public endpoint anonymously — https://trip2g.com/en/user/token-economy-bench. Point `ENDPOINT` at your own instance to measure your vault.
>
> Why this shape of memory at all: the notes are human-readable, editable in Obsidian, versioned in git, and the same notes are a browsable website. It's deliberate memory, not automatic conversation capture — if you want auto-extraction, Mem0/Letta do that and I'm not claiming a LongMemEval score. Local story: full-text search is fully offline; semantic search takes any OpenAI-compatible embeddings endpoint, including a local one.
>
> Repo: https://github.com/trip2g/trip2g. Author, happy to defend the methodology.

---

## 3. dev.to cross-posts

Cross-post **after** the originals are indexed on trip2g.com (give Google ~1 week), with `canonical_url` in the dev.to front matter pointing back to the trip2g.com page — dev.to supports this natively, so the ranking equity stays home while dev.to's fast indexing catches the query early ([growth plan Step 4](2026-07-02_seo_growth_plan.md)).

Which Step-1 articles to cross-post (pick tutorial-shaped ones — dev.to is a how-to venue, comparisons underperform there):

| Step-1 page | dev.to title | Why this one |
|---|---|---|
| `claude-code-persistent-memory.md` | "Give Claude Code persistent memory with one MCP URL (self-hosted)" | The proven dev.to genre — the [self-hosted mem0 tutorial](https://dev.to/n3rdh4ck3r/how-to-give-claude-code-persistent-memory-with-a-self-hosted-mem0-mcp-server-h68) shows exactly this query converting there. Tags: `ai`, `mcp`, `claude`, `selfhosted`. |
| `publish-obsidian-vault-as-website.md` | "Publish an Obsidian vault as a website: all the ways, honestly compared" | dev.to already ranks for the GH-Pages variant of this query; survey shape travels well. Tags: `obsidian`, `markdown`, `webdev`, `selfhosted`. |
| Token-economy bench (adapted from `docs/en/user/token-economy-bench.md`) | "I measured MCP section-reads vs whole-note reads: median 15× fewer tokens (script included)" | Benchmark + runnable script = dev.to catnip; doubles as the r/LocalLLaMA post's durable home. Tags: `ai`, `llm`, `mcp`, `benchmark`. |

Front-matter template per post: `canonical_url: https://trip2g.com/en/user/<slug>`, cover image = the matching share image (§4), first line links "originally published on trip2g.com". Do **not** cross-post the vs-Quartz / vs-Publish comparisons — vendor comparisons on dev.to read as ads and get downvoted.

---

## 4. Share images (specs + text, no binaries here)

1. **Benchmark headline card** (OG image for the bench page, X/Reddit card, dev.to cover). 1200×630 PNG, dark bg matching trip2g.com. Left: two horizontal bars, dramatically unequal — `whole note: ~2,700 tokens` vs `focused section: ~200 tokens`, drawn to scale. Right/top text: `Reading the answer: 15× fewer tokens` and small print `median over 8 real queries, live endpoint, reproducible — trip2g.com/en/user/token-economy-bench`. No logo bigger than the number. The number is the hero.
2. **"Sync → live" GIF** (README hero, r/ObsidianMD, X). ~15s, ≤5 MB, 1000px wide, loop. Storyboard: (a) 0–4s Obsidian window, typing a heading + a line into a note; (b) 4–6s save; (c) 6–10s browser beside it refreshes to show the same text live on the site; (d) 10–15s bonus beat: the same note fetched via MCP in a terminal (`search` → section text), proving the two-audience story in one loop. Screen-record real product only — no mockups; the honesty extends to the pixels.
3. **Architecture card** (HN thread replies, dev.to, docs). 1200×630 PNG, monochrome + one accent. Center: one markdown file icon labeled `note.md — frontmatter + body`. Four arrows out: `website (HTML)`, `agent (MCP)`, `Telegram post`, `git clone`. Caption: `One note. Four consumers. Go + SQLite, MIT.` This is the "what is it" answer as a picture; keep it to ≤12 words of labels.

---

## 5. Sequencing + timing checklist

### Gate (hard rule — do not fire before all are true)

- [ ] Step 1 content batch live on trip2g.com (at minimum: bench page current, `publish-obsidian-vault-as-website`, `claude-code-persistent-memory`, one vs-page per lane) — launch traffic that lands on missing pages is spent once and wasted.
- [ ] Step 2 engine fixes deployed (meta-description fallback, robots sitemap pointer, `og_image` — the share cards in §4 need it to render on social).
- [ ] Step 3 README converts in 10 seconds: GIF above the fold, plain two-line promise, repo topics set, CONTRIBUTING + good-first-issues up.
- [ ] `python3 te_check.py` against production returns the table (someone on HN *will* run it within the hour — if prod is mid-deploy or rate-limits, the whole hook inverts).
- [ ] Prod box headroom checked (one small VM hosts everything; HN front page is a load event — schedule no deploys that day, warm the page cache).
- [ ] Free-cloud-instance signup path tested end-to-end.
- [ ] Each subreddit's current rules re-read on the day; flair chosen.

### Timing

| Platform | Best window | Rationale |
|---|---|---|
| Show HN | Tue/Wed/Thu, 9:00–12:00 ET (15:00–18:00 UTC) | Guide consensus; enough US morning traffic to escape /new, enough day left to ride the front page. |
| r/ObsidianMD | 2–4 days **after** HN, weekday morning US | Staggering lets each post link the previous discussion as social proof and spreads the author's attention. |
| r/selfhosted | same week as r/ObsidianMD, different day, Tue–Thu ~14:00 UTC | The sub's traffic is EU+US; midweek early-US works. |
| r/LocalLLaMA | its own day, any weekday | The benchmark post must not compete with the product posts for the author's comment time. |
| dev.to cross-posts | ~1 week after originals indexed | Canonical must already be crawled or dev.to can outrank the origin. |

One platform per day, HN first. If HN flops (< ~20 points by hour 3), that's fine and expected — the Reddit posts are independent shots, and a second Show HN of the same repo is acceptable months later with new substance.

### The ~2-hour human runbook (launch day, HN)

1. **T–30 min:** re-run `te_check.py` against prod; open the README, bench page, cloud signup in incognito — all render, GIF loads. Coffee.
2. **T=0:** submit repo URL with the chosen title (§1.1). Immediately post the first comment (§1.3), rewritten in own voice with the personal backstory line added.
3. **T+0…90 min (the decisive window):** reply to *every* comment — fast, technical, zero defensiveness; concede valid criticisms explicitly ("fair — that's on the roadmap / that's a real limitation"). Use the FAQ crib (§1.4) as notes, never paste. Do not ask anyone to upvote; do not have friends comment.
4. **T+90…120 min:** keep replying at lower cadence; note recurring questions → they become docs FAQ entries and the next day's tweet material. Snapshot star count (star-history.com) for the growth-plan measurement log.
5. **Rest of day:** check the thread hourly; a slow answer to a hostile technical question at hour 5 costs more than one at hour 1.
6. **Day 2:** write the "what we learned from the thread" note (feeds Step-5 recurring presence), fix anything embarrassing that surfaced, then schedule the first Reddit post.

### Expected outcomes — candid

- If HN lands (front page): **~100–300 stars in 48h** at ~1.4 stars/upvote ([arxiv 2511.04453](https://arxiv.org/abs/2511.04453)) plus a durable traffic step; most impact is day-1.
- Base rate honesty: most Show HN posts get <10 points; the tag itself confers no advantage after controls ([Show HN by the numbers](https://danfking.github.io/blog/2026/04/23/show-hn-by-the-numbers/)). What moves the needle is a runnable claim + author presence — which is why the benchmark leads and the human owns the thread.
- Reddit posts individually smaller (tens of upvotes each is a win in these subs) but they compound: comment threads there rank in Google for the exact Step-1 queries.
- None of this replaces Steps 1–3; the launch spends attention that the content and README must convert.

---

## Sources

Launch-craft: [markepear — How to launch a dev tool on HN](https://www.markepear.dev/blog/dev-tool-hacker-news-launch) · [lucasfcosta — How to do a successful HN launch](https://www.lucasfcosta.com/blog/hn-launch) · [daily.dev — HN marketing for dev tools](https://business.daily.dev/resources/hacker-news-marketing-developer-tools-show-hn-launch-day-sustained-coverage/) · [syften — HN posting guide](https://syften.com/blog/hacker-news-marketing/) · [Show HN by the numbers](https://danfking.github.io/blog/2026/04/23/show-hn-by-the-numbers/) · [arxiv 2511.04453 — HN impact on GitHub stars](https://arxiv.org/abs/2511.04453)

Dissected launches (HN Algolia, 2026-07-02): [Obsidian (1087 pts)](https://news.ycombinator.com/item?id=23324598) · [open-source obsidian sync server (416 pts)](https://news.ycombinator.com/item?id=37247767) · [Plausible self-hosted (351 pts)](https://news.ycombinator.com/item?id=24696145) · [OpenWorkers (500 pts)](https://news.ycombinator.com/item?id=46454693)

Internal: `docs/dev/2026-07-02_seo_growth_plan.md` (Step 4 spec, gate rule) · `docs/dev/2026-07-02_seo_content_research.md` (honesty-as-conversion pattern) · `docs/dev/token_economy_bench.md` + `docs/en/user/token-economy-bench.md` (all benchmark numbers) · `README.md` (feature claims; status column = source of truth)
