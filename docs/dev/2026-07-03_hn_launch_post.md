# Show HN launch post — freshened against HN as of 2026-07-03

Builds on `2026-07-02_launch_kit.md` (strategy, FAQ crib, runbook — all still valid). This doc updates only what the last 90 days of HN changed: the title pick, the maker comment, and the thread tactics. Product claims grounded in `README.md` (status column = source of truth) and `docs/en/user/token-economy-bench.md`.

## TL;DR

Launch **Tuesday 2026-07-07 or Wednesday 2026-07-08, 9:00–12:00 ET** — not this week (July 4 holiday weekend kills US traffic). Title: **kit option A**, unchanged. The token-economy angle is hotter than when the kit was written — token cost is literally in this week's news — but HN has also sharpened its skepticism of "N× fewer tokens" claims (the Semble thread is the playbook for what they'll ask). The maker comment below adds the one caveat the kit missed: **retrieval-level savings ≠ end-to-end agent performance**. Do not frame trip2g as an Obsidian alternative — that lane got hit a week ago (OpenKnowledge) and its top comment ("Obsidian is already AI-friendly") is a trap we sidestep by *serving* the vault, not replacing it.

---

## 1. What's landing on HN right now (research, 2026-07-03)

### The four hookable threads

| Thread | Result | Takeaway for us |
|---|---|---|
| [Show HN: Semble – Code search for agents that uses 98% fewer tokens than grep](https://news.ycombinator.com/item?id=48169874) (2026-05-17) | 445 pts / 151 c | The closest analog: a token-efficiency benchmark in the title landed. But the thread set the interrogation script: commenters demanded **baseline fairness** ("agents use surgical grep, not naive grep") and **end-to-end validation** ("token savings during search could be offset by extra turns or model distrust"), and complained that "so many people share these tools with zero evals or suspiciously hard-to-reproduce benchmarks." Our anonymous, stdlib-only, live-endpoint script is the direct answer to that last complaint — lead with it. Concede the first two before asked. |
| [Show HN: OpenKnowledge – open source AI-first alternative to Obsidian/Notion](https://news.ycombinator.com/item?id=48675435) (2026-06-25, one week ago) | 381 pts / 173 c | Top comment: "I don't understand how Obsidian, a collection of markdown files, isn't already AI-friendly." The "alternative to Obsidian" frame invites that hit. But buried in the thread is our exact customer (user abdullin): wants a team knowledge base that is "(1) versioned à la git, (2) usable from any chat à la MCP, (3) basic access controls, (4) works hosted." That comment is trip2g's feature list. Position as the thing that *serves* the vault; if the thread allows, answer that shaped question directly. |
| [MCP is dead?](https://news.ycombinator.com/item?id=48330436) (2026-05-29, 400 pts) + [I still prefer MCP over skills](https://news.ycombinator.com/item?id=47712718) (2026-04-10, 460 pts) + [When does MCP make sense vs CLI?](https://news.ycombinator.com/item?id=47208398) (2026-03-01, 447 pts) | — | MCP fatigue is real: "if MCP did not exist today we would not invent it"; CLI/skills preferred for local work. But the threads converge on a carve-out where MCP still wins: **remote, credentialed, permission-scoped data** — which is exactly a shared/federated/ACL'd knowledge hub. We should say this ourselves: trip2g is the case where MCP earns its keep, and for local-only use, grep your vault, we measured that too. |
| Token-economy macro-mood: [Caveman: Why use many token when few token do trick](https://news.ycombinator.com/item?id=47647455) (2026-04-05, 904 pts) · [MCP server that reduces Claude Code context consumption by 98%](https://news.ycombinator.com/item?id=47193064) (2026-02-28, 570 pts) · [Making MCP cheaper via CLI](https://news.ycombinator.com/item?id=47157398) (2026-02-25, 324 pts) · [Meta caps internal AI token spending](https://news.ycombinator.com/item?id=48754713) (**2026-07-01**, 146 pts) | — | Token cost is a front-page genre all spring, and as of *this week* it's enterprise news (Meta capping token spend). The bench hook rides a live conversation. One caution: ours would be at least the third "fewer tokens" number in a title this half-year — put the number in the comment, not the title, to avoid reading as pattern-matching. |

### Secondary context (worth knowing, not riding)

- [Show HN: A Karpathy-style LLM wiki your agents maintain](https://news.ycombinator.com/item?id=47899844) (2026-04-25, 260 pts) — a commenter noted it was "the third llm wiki on front page in 24 hours." The markdown-KB-for-agents lane is crowded with *specs and wikis*; our differentiation is a running server with measured numbers, not another format. Another comment there — "why not an Obsidian vault with a plugin?" — is literally trip2g.
- [Google proposes Open Knowledge Format based on Markdown](https://news.ycombinator.com/item?id=48517735) (2026-06-13) — markdown-as-agent-substrate is now a Google-endorsed idea; fine to cite if asked "why markdown."
- [We replaced RAG with a virtual filesystem for our AI documentation assistant](https://news.ycombinator.com/item?id=47618223) (2026-04-02, 411 pts) — HN currently likes "navigation instead of RAG"; our `toc_path`/`expand` story fits this frame.
- [Obsidian plugin was abused to deploy a remote access trojan](https://news.ycombinator.com/item?id=48088576) (2026-05-10, 366 pts) — plugin security is fresh in memory; expect a question about our side-loaded sync plugin.
- [Show HN: Files.md – Open-source alternative to Obsidian](https://news.ycombinator.com/item?id=48179677) (2026-05-18, 730 pts) and [Launch HN: Manufact – MCP Cloud](https://news.ycombinator.com/item?id=48762862) (2026-07-02, 102 pts) — the lanes are active; neither is a direct collision.

### Mood read, mid-2026

- **Token economy: hot, and hardened.** Big numbers land, but the Semble thread wrote the cross-examination script: baseline fairness, end-to-end effect, reproducibility. We win on reproducibility (anonymous live script), must concede the other two proactively.
- **MCP: fatigued, with a defended niche.** Don't sell MCP; sell the case (remote, shared, ACL'd) where even the skeptics grant it. Saying "for local files, just grep — here's our measurement of that too" buys enormous credibility.
- **Agent memory (Mem0/Letta frame): quiet.** No big memory-layer launch in months (biggest recent: a 116-pt Elasticsearch memory layer, [2026-06-18](https://news.ycombinator.com/item?id=48583703)). The live frame is "markdown KB agents can read/maintain" — position there, not as a "memory layer."
- **Obsidian tooling: crowded and slightly defensive.** Two big adjacent launches in 6 weeks; the community's reflex is "Obsidian already does this." Complement, don't compete.

**Recommendation: ride the token-economy conversation, anchored in the MCP-carve-out framing, with the Obsidian vault as the noun in the title.** That's one coherent story: your vault, served to humans and agents, and here's the measured bill.

---

## 2. The title (one shot — final)

> **Show HN: Trip2g – Self-hosted server that turns an Obsidian vault into a website and MCP server**

Kit option A, confirmed against the current moment:

- Option C (number in the title) is out: after Semble (May) and the 98%-context MCP server (Feb), a third "N× fewer tokens" title reads as me-too, and it paints a target on the benchmark before the comment can frame it honestly. The number opens the *comment* instead — where Plausible put its numbers too.
- Option B (stack in title) is out: "Go/SQLite/MIT" earns its keep in the first comment; the title's job is the noun and the two outputs.
- A survives the OpenKnowledge lesson because it never claims to replace Obsidian — the vault stays; the server serves it. The word "alternative" appears nowhere.

Submission URL: `https://github.com/trip2g/trip2g` (repo, not landing page — kit rule 4).

---

## 3. The maker first comment (post within 5 minutes of submitting)

> Hi HN, author here.
>
> **[HUMAN: one or two sentences here — why *you* built this. The real itch, in your own words. Do not skip; this line is worth more points than any paragraph below.]**
>
> trip2g is a self-hosted Go server (single binary, SQLite, MIT) that serves one set of markdown notes two ways: to people as a website, to agents over MCP. An Obsidian vault syncs in two-way — save a note locally, the page is live in about a second; write through the API, it flows back into the vault. Every edit is versioned and mirrored to git, so the whole knowledge base is also `git clone`-able.
>
> The part I'd most like you to poke at: the retrieval bill. Search results carry a `toc_path` pointing at the exact section, so an agent reads ~200 tokens instead of the whole note. On the live docs vault that's a **median 15× saving, up to 37× on long notes**, versus loading the full note. You can check it yourself in about a minute — a 50-line stdlib-only Python script that hits the public endpoint anonymously: https://trip2g.com/en/user/token-economy-bench
>
> Three honest caveats on that number, before you find them:
> - The baseline is the naive whole-note dump, not a tuned RAG pipeline. Good chunk retrieval shrinks the gap.
> - It's a retrieval-level measurement, not end-to-end agent performance. Cheaper reads can be eaten by extra turns if the agent distrusts the tool; we haven't measured that yet, and I won't pretend otherwise.
> - Token saving turned out to be the easy half. Deterministic section-picking failed 8 of 9 real queries in our tests — landing on the *right* section is the hard problem, which is why search returns a precise path and there's an `expand` tool for walking the heading tree.
>
> On "why MCP at all" — fair, given the mood lately. For local files: just grep, honestly; we measured naive grep+read too (~23× more tokens than a focused read, numbers on the same page), but a careful local setup closes much of that. The case MCP actually earns here is the one grep can't reach: a base that's remote, shared, or permission-scoped. Hubs federate — one query fans out to peer hubs over HMAC-signed, depth-limited calls — and access is per-subscription, so a team KB is versioned like git, readable from any MCP client, and ACL'd.
>
> Stack: Go + FastHTTP, gqlgen, SQLite (+ Litestream), bleve full-text, Goldmark markdown; semantic search via any OpenAI-compatible embeddings endpoint (full-text works fully offline).
>
> Limitations:
> - The note-triggered agent runtime in the README table is on a branch, not main. The table marks shipped vs branch vs planned — hold me to it.
> - The Obsidian sync plugin is side-loaded for now; community-directory review pending. It's open source — read it before installing, as you should anything after this spring's plugin-RAT story.
> - No graph view. If that's your favorite Obsidian feature, Obsidian Publish or Quartz will make you happier.
> - The admin frontend uses $mol, a framework you've probably never heard of; the server itself has zero JS dependencies.
> - Young project, small community. Early-adopter territory.
>
> `docker compose up` to self-host, or a free cloud instance if you don't want to run a server. Happy to go deep on the sync protocol, federation design, or the benchmark methodology.

Changes vs the kit's draft, and why:

1. **Personal-backstory slot moved to the top** and made mandatory — the 416-pt sync-server precedent, and it's the one thing that can't be pattern-matched as AI-written.
2. **Added the end-to-end caveat** (retrieval-level ≠ agent performance) — the Semble thread's sharpest criticism, conceded before it's raised. This is the single most important edit.
3. **Added the MCP-fatigue paragraph** — "for local files, just grep" mirrors the "When does MCP make sense vs CLI?" consensus and claims the carve-out (remote/shared/ACL'd) those threads themselves granted.
4. **Plugin security line added** — the May plugin-RAT thread makes "side-loaded plugin" a live concern; naming it first defuses it.
5. Grep number softened to "a careful local setup closes much of that" — the bench page itself concedes naive grep is not lab-clean; don't let 23× become the thread's pile-on target when 15× is the defensible number.

---

## 4. How to land — tactics for this specific week

### Timing

- **Do not launch 2026-07-03 through 07-06.** July 4 falls on Saturday; Friday the 3rd is the observed US federal holiday and Monday drags. US morning traffic — the escape velocity out of /new — is at its yearly low.
- **Fire Tuesday 2026-07-07 or Wednesday 2026-07-08, 9:00–12:00 ET** (15:00–18:00 UTC). First post-holiday weekdays: readers are back, and the weekend produced few competing launches.
- **Collision check at T–15 min:** scan the front page and /new for another markdown-KB/MCP/Obsidian launch that morning. The lane bursts (three LLM wikis hit the front page in 24 hours in April); if a bigger adjacent post is climbing, wait a day. Manufact's MCP Cloud Launch HN (07-02) will be gone by then; OpenKnowledge is a week old — old enough to reference, too old to compete with.

### Which threads to reference in-thread (naturally, never as name-drops)

- When "why not just grep / MCP is dead" appears (it will, early): answer with the carve-out framing above; linking [When does MCP make sense vs CLI?](https://news.ycombinator.com/item?id=47208398) is fine if a commenter opens that door.
- If someone asks about team knowledge bases: the [abdullin comment in the OpenKnowledge thread](https://news.ycombinator.com/item?id=48675435) describes the exact requirement set (git-versioned + any-chat MCP + ACLs + hosted) — answering with that shape, in your own words, converts the whole subthread.
- If the benchmark discussion heats up: acknowledge the [Semble thread](https://news.ycombinator.com/item?id=48169874) dynamics directly — "the fair criticism of these claims is end-to-end effect; here's what we did and didn't measure." HN rewards knowing your own genre's weaknesses.

### Three landmines specific to July 2026 (extends the kit FAQ — the other 11 answers stand)

| Landmine | Crisp reply |
|---|---|
| **"Component benchmark ≠ agent performance. Semble claimed 98% too."** (the new default attack on token claims) | "Correct, and it's in my first comment: this is retrieval-level. What we can defend: same-ruler ratio, live endpoint, script you can run right now — the reproducibility complaint from that thread is the one we fixed. End-to-end (does the agent answer better/cheaper overall) needs a judge model and a held-out QA set; `scripts/expand_check.py` is the start, real eval is on the roadmap and I'm not claiming it yet." |
| **"MCP is dying, skills/CLI won. Why build on it in 2026?"** | "For local files I agree — grep, we measured it, it's on the page. MCP is the transport for the case CLI can't touch: remote, shared, permission-scoped bases, and hub federation. Even the 'MCP is dead?' thread granted that niche. If a better open transport for that wins later, the notes are plain markdown and git-cloneable — nothing is locked to the protocol." |
| **"Obsidian is already AI-friendly, this is yet another AI-first note tool"** (verbatim top comment on OpenKnowledge, one week ago) | "It is — locally. Nothing replaces Obsidian here; it stays your editor and the vault stays on disk. trip2g is the *server* half: the same notes as a website for readers and an MCP endpoint for agents that aren't on your laptop — with versions, ACLs, and federation. If you only ever query your own vault from your own machine, you don't need it." |

### Runbook

The kit's ~2-hour runbook (§5) stands unchanged: comment within 5 minutes, reply to everything in the first 90, concede valid criticism explicitly, never solicit votes.

---

## 5. Gate reminder — do not fire until all green

Restating the hard gate from the kit (§5), with today's status where known:

- [ ] **Growth-plan Steps 1–3 live** (content batch on trip2g.com, engine SEO fixes, README converting in 10s). **Flag: the README hero GIF is still an HTML comment placeholder** (`README.md` line 19, checked 2026-07-03) — Step 3 is visibly not done. Launch traffic hitting a static screenshot instead of the sync GIF wastes the one shot.
- [ ] **`python3 te_check.py` green against production** on launch morning — someone on HN will run it within the hour; a rate-limit or mid-deploy 500 inverts the entire hook. Run at T–30 and keep the output.
- [ ] **Prod box headroom checked** — one small VM hosts everything and has a history of resource-pressure outages; front page is a load event. No deploys that day; warm the page cache; consider the known GC limits.
- [ ] Free cloud-instance signup tested end-to-end in incognito.
- [ ] Front-page/new collision scan at T–15 (see §4).

Not ready today on at least the GIF; the July 4 weekend conveniently buys the 3–4 days to close the gate before the Tue 07-07 window.

---

## Sources

Fresh HN research (Algolia API + thread reads, 2026-07-03): [Semble 445 pts](https://news.ycombinator.com/item?id=48169874) · [OpenKnowledge 381 pts](https://news.ycombinator.com/item?id=48675435) · [MCP is dead? 400 pts](https://news.ycombinator.com/item?id=48330436) · [I still prefer MCP over skills 460 pts](https://news.ycombinator.com/item?id=47712718) · [When does MCP make sense vs CLI? 447 pts](https://news.ycombinator.com/item?id=47208398) · [Caveman 904 pts](https://news.ycombinator.com/item?id=47647455) · [98% context MCP server 570 pts](https://news.ycombinator.com/item?id=47193064) · [Making MCP cheaper via CLI 324 pts](https://news.ycombinator.com/item?id=47157398) · [Meta caps token spending](https://news.ycombinator.com/item?id=48754713) · [Karpathy-style LLM wiki 260 pts](https://news.ycombinator.com/item?id=47899844) · [Google Open Knowledge Format](https://news.ycombinator.com/item?id=48517735) · [RAG→virtual filesystem 411 pts](https://news.ycombinator.com/item?id=47618223) · [Obsidian plugin RAT 366 pts](https://news.ycombinator.com/item?id=48088576) · [Files.md 730 pts](https://news.ycombinator.com/item?id=48179677) · [Manufact MCP Cloud](https://news.ycombinator.com/item?id=48762862) · [Elasticsearch agent memory 116 pts](https://news.ycombinator.com/item?id=48583703)

Internal: `docs/dev/2026-07-02_launch_kit.md` · `docs/dev/2026-07-02_seo_growth_plan.md` · `docs/en/user/token-economy-bench.md` + `docs/dev/token_economy_bench.md` (15.4× median, 37.2× max, 3.7× min, ~23× grep, 8/9 deterministic misses) · `README.md` (status column = truth; agent runtime = branch)
