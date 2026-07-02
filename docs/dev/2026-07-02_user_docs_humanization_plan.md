# User docs humanization plan

Editorial plan. Not a rewrite. A follow-up worker executes it.

## TL;DR

The user docs are already good: nearly every page opens answer-first with a concrete reader outcome, the English is human and honest, and the Russian reads as native (not translationese). The corpus does not need a humanization sweep. The highest-leverage move is fixing the **positioning split** — three surfaces (README, live landing, docs home) each say a different thing about what trip2g is — and clearing a short list of **shipped staleness** (screenshot placeholders, TODO comments, a stale image pin). Tone is fine; the message and the freshness are the work.

---

## Audit findings

Sampled ~22 EN files and 13 RU files across getting-started, features, advanced, and ops, plus the landing surfaces. Cross-language link lint (`go run -tags dev ./cmd/server lint docs`) is clean for user docs — the only hits are three `info`-level broken links in `demo/canvas_link_test.md`, unrelated to `en/user` or `ru/user`. (Lint first needs `go generate ./onboarding-vault/` to build the embedded `vault.zip`, or it fails to compile.)

### What is already strong

- **Answer-first is the house style.** getting-started, two-way-sync, seo, federation, selfhosted, fly, user_management all lead with the reader's outcome and stay concrete. seo.md is exemplary: "A common question: 'How is your SEO?' Short answer: the same as anywhere else."
- **Honesty over hype.** Docs flag their own limits (`forms.md` has a "What's not implemented yet" section; `selfhosted.md` warns "'any OpenAI-compatible service' only works if..."). This is the opposite of AI-slop.
- **Anti-slop compliance is high.** No "In this guide, we will...", almost no promotional vocabulary. The only AI-vocab hit in the sampled set is one "robust" in `search.md:13`.
- **The Russian is native.** Across ~65k words: `осуществляется` 0, `предоставляется возможность` 0, `в случае если` 0, `данный` only 3. Answer-first openings are the norm. This reads as originally authored, not translated.

### Real problems (localized, not systemic)

1. **Positioning is fragmented across three surfaces.** See the landing verdict below. This is the single biggest issue and it is not a tone issue.

2. **Shipped editorial cruft (staleness).** Concrete rot that a reader sees:
   - `docs/en/user/live-editing.md:20-21` — an unrendered screenshot art-direction note left in the published body.
   - `docs/en/user/editor.md` — same pattern (`> 🖼️ Screenshot ...` placeholder).
   - `docs/en/user/advanced.md:51` and `:97` — two `<!-- TODO: verify with product -->` comments shipped in prose, under the frontmatter-patch and OAuth headings.
   - `docs/en/user/hosting.md:26` — pins `ghcr.io/trip2g/trip2g:0.2` while every other doc uses `:latest`.
   - `docs/en/user/publishing.md:68-69` — an empty `slug` example code block (dropped content).

3. **One weak file: `monetization.md`.** The integration sections are vague and product-centric — they describe nothing actionable.

4. **Duplicate/overlapping structure.**
   - `backup.md` (server SQLite DB) and `backups.md` (Obsidian vault) both carry the frontmatter title "Backups". Only `backups.md` is in `_sidebar.md`, so `backup.md` is orphaned from nav.
   - `advanced.md` re-documents SEO, backups, and CLI that already have dedicated files (`seo.md`, `backups.md`, `cli.md`) — the main content-divergence risk in the corpus. Its own TODO at `:51` already shows it is unsure of syntax that `frontmatter-patches.md` states confidently.

5. **RU terminology drift (one term only).** The Obsidian-storage concept appears as `хранилище` (133, the dominant native term), transliterated `волт` (`local-quickstart.md`, `okf.md`, `fly.md`), and English `vault` in prose (`agent_admin.md`, `mcp.md`). `vault` inside code/CLI is fine; `волт`/`vault` in running prose collides with `хранилище`. All other terms are consistent (`хаб` 73, `инстанс` 50, `заметка` 740).

6. **A couple of RU grammar/punctuation slips** (not translationese, just typos): `forms.md:36` "собирает свой вёрстку" (should be "свою вёрстку"); `forms.md:7` missing comma "ввод привязанный" (should be "ввод, привязанный").

### Before / after snippets

**1. AI-vocab — `search.md:13`**
> Before: "This is a standard academic method for hybrid search, robust to differences in scoring scales."
> After: "This is a standard method for hybrid search. It works even when the two searches score results on different scales."

**2. Product-centric, non-actionable intro — `advanced.md:7`**
> Before: "Advanced features for custom domains, SEO, backups, and command-line access."
> After: "Point your own domain at a base, sync from the command line, and tune how search engines see your site. Each section links to the full guide."

**3. Says-nothing filler — `monetization.md:33`**
> Before: "Boosty integration works similarly to Patreon: subscribers authenticate via Boosty and receive access to gated content based on their subscription tier."
> After: "Connect Boosty the same way as Patreon (admin panel → Integrations). A reader who logs in with Boosty gets access to notes whose `subscription_tier` matches their paid tier."

**4. Shipped cruft — `live-editing.md:20-21`**
> Before: a `> 🖼️ **Screenshot — assets/admin/live-editing-side-by-side.png**` block plus a paragraph of art-direction notes.
> After: either add the real screenshot, or cut the block. Do not ship the art-direction note.

**5. RU terminology drift — `okf.md:22`**
> Before: "## Почему волт trip2g уже является OKF-бандлом"
> After: "## Почему хранилище trip2g — это уже OKF-бандл"

---

## Landing verdict: "Markdown Operating System"?

**Recommendation: adopt as a secondary tagline for the technical/README audience; do NOT make it the primary hero for the docs home or the public landing.** The three surfaces should share one plain primary promise and use the OS metaphor as depth, not as the headline.

### The problem: three surfaces, three messages

| Surface | Current lead | Framing |
|---|---|---|
| `README.md` | "Markdown Operating System." | OS metaphor, for a technical reader |
| Live landing (`_layouts/mesh/hero.html`) | "Join the network. Your agents will find the rest." | Federation-first, network-first |
| Docs home (`docs/en/user/_index.md`) | "trip2g turns your Obsidian vault into a website in under a minute." | Plain job-to-be-done |

Someone who arrives wanting to "publish my Obsidian vault as a website" meets a different pitch on each page. Only the docs home speaks to that job. The live landing hero leads with **federation** — an advanced capability — and never plainly says "publish your vault as a website," which is the actual first-timer job. The README's OS metaphor is accurate and earned but lands for a developer evaluating architecture, not for the vault owner.

### Is the metaphor earned?

Yes, honestly. The README's OS table maps shipped primitives to OS concepts and tags each by reality (shipped / branch / planned). Filesystem (one note namespace), syscalls (MCP tools), network stack (federation), display server (templates), scheduler (cron webhooks), snapshots (`note_versions` + git mirror) are all shipped. The agent fleet and in-note executor are on `feat/agent-runtime`, and the README says so. The metaphor is not vaporware dressing.

But "earned" is not the same as "the right first sentence for a first-timer." The OS framing rewards a reader who already gets the analogy; it costs a reader who just wants their notes online. Lead with the job, reward the curious with the metaphor one screen down.

### Recommendation, concretely

- **Primary hero (docs home + landing):** the plain promise. "Publish your Obsidian vault as a website in under a minute."
- **Secondary tagline (README stays, landing gains a band):** "A Markdown Operating System — every note is a file, served to a human as a page and to an agent as an MCP call." Keep this as the second beat, not the first.
- **Realign the live landing hero.** "Join the network. Your agents will find the rest." is poetic but federation-first; it should move below the plain promise and the 3-step "how it works". Federation is the wow, not the door.

---

## Main pages: what to put on them

Grounded in what the user docs actually cover. Write both EN and RU; RU in natural Russian, not a translation of the EN.

### `docs/en/user/_index.md` (docs home)

**Hero line:** Publish your Obsidian vault as a website in under a minute.
**Subline:** Write in Obsidian, press Sync, your notes are live. The same hub also publishes to Telegram, gates paid content, and answers reader questions over MCP. A Markdown Operating System: one note, served to a human as a page and to an agent as a syscall.

**Primary path (feature prominently, above the capability grid):**
> New here? Follow **[[en/user/getting-started|Get your first note live]]** — instance → sign in → install the plugin → sync. About a minute.

**Capabilities to feature (6, each a one-line reader-benefit hook + link):**
1. **Publish from Obsidian** — your vault stays on your machine; sync pushes notes live, one-way or two-way. → `two-way-sync`
2. **Publish to Telegram** — write once in Obsidian, post to your channel on a schedule, links stay intact. → `telegram`
3. **Charge for content** — leave `free: true` off and a note is subscriber-only; take payment via crypto, Patreon, or Boosty. → `monetization`
4. **An AI that answers from your notes** — connect any MCP client and readers query your knowledge base directly. → `mcp`
5. **Make it look how you want** — compose pages from frontmatter widgets, or drop in your own Jet templates. → `templates`
6. **Run it yourself** — one Go process on SQLite; `docker compose up` on a laptop, a VM, or Fly. → `selfhosted`

**Section order:** hero + subline → primary getting-started path → 6-capability grid → "who uses it" (link `use-cases`) → footer nav. Surface: getting-started, the 6 above, use-cases. Bury (keep in sidebar, not on home): `bem`, `jet-debugging`, `yield_blocks`, `renderlayout`, `chartdata`, `read-replica`, `perfomance` — reference material, not first-visit content.

### `docs/ru/user/_index.md`

**Заголовок:** Опубликуйте хранилище Obsidian как сайт — за минуту.
**Подзаголовок:** Пишете в Obsidian, жмёте Sync — заметки уже онлайн. Тот же хаб публикует в Telegram, закрывает платные материалы за пейволом и отвечает на вопросы читателей по MCP. Markdown-операционка: одна заметка — и страница для человека, и системный вызов для агента.

**Главный путь (вынести над сеткой возможностей):**
> Впервые здесь? Пройдите **[[Начало работы|Запустите первую заметку]]** — инстанс → вход → плагин → синхронизация. Около минуты.

**Возможности (6, каждая — одна строка выгоды + ссылка):**
1. **Публикация из Obsidian** — хранилище остаётся у вас; синхронизация отправляет заметки на сайт, в одну или в обе стороны. → `two-way-sync`
2. **Публикация в Telegram** — пишете один раз в Obsidian, постите в канал по расписанию, ссылки не ломаются. → `telegram`
3. **Платный доступ** — уберите `free: true` — и заметка только для подписчиков; оплата через крипту, Patreon или Boosty. → `monetization`
4. **ИИ, который отвечает по вашим заметкам** — подключите любой MCP-клиент, и читатель спрашивает вашу базу знаний напрямую. → `mcp`
5. **Внешний вид под вас** — собирайте страницы из виджетов во фронтматтере или подключите свои Jet-шаблоны. → `templates`
6. **Свой сервер** — один Go-процесс на SQLite; `docker compose up` на ноутбуке, VPS или Fly. → `selfhosted`

**Порядок секций:** тот же, что в EN. Терминология: `хранилище` (не «волт»/«vault» в прозе), `хаб`, `инстанс`, `заметка`, `база знаний`, `пейвол`.

### Live landing (`_layouts/mesh/`) — non-blocking note

The mesh landing is a separate template surface (hero → how → cases → compat → privacy → roadmap → philo → matrix → try_now → ...). It is out of scope for a docs-only pass, but the positioning fix should reach it: reorder so the plain "publish your vault" promise leads and "join the network / federation" follows as the payoff. Flag for a separate frontend task; do not edit it in the docs batch.

---

## Humanization plan (prioritized)

The corpus does not need a line-by-line humanization pass. The work is targeted: positioning, staleness, one weak file, and a small RU terminology sweep. Sequence by reader impact.

### Batch 1 — Positioning (highest leverage). ~2-3h.
- Rewrite `docs/en/user/_index.md` and `docs/ru/user/_index.md` per the "Main pages" section above (plain hero + 6-capability grid + prominent getting-started path).
- Confirm the README's OS lead stays but reads as one voice with the docs home (plain promise reachable in one line).
- Leave the mesh landing template for a separate frontend task (note it in the PR).
- **Do both EN and RU in the same batch** so they ship parallel.

### Batch 2 — De-staleness (fast, high trust-impact). ~1-2h.
Per-file, mechanical:
- `live-editing.md:20-21` and `editor.md` — add the real screenshot or cut the placeholder block. Never ship the art-direction note.
- `advanced.md:51,:97` — resolve or delete both `<!-- TODO: verify with product -->` comments (verify against `frontmatter-patches.md` and `oauth.md`, which state the same facts confidently).
- `hosting.md:26` — change `:0.2` to `:latest` (or whatever the other docs use).
- `publishing.md:68-69` — fill or remove the empty `slug` code block.
- `search.md:13` — replace "robust to differences in scoring scales" (see before/after).

### Batch 3 — Fix the one weak file + de-duplicate. ~2h.
- `monetization.md` — rewrite the Patreon/Boosty integration sections to be actionable (where in the admin panel, what field maps to what tier). Match the concreteness of `telegram.md`.
- Resolve the `backup.md` / `backups.md` title collision: give them distinct frontmatter titles ("Back up the server database" vs "Back up your Obsidian vault") and add `backup.md` to `_sidebar.md`, or merge under one page with two sections.
- `advanced.md` — decide: make it a thin hub that links to `seo.md` / `backups.md` / `cli.md` (like RU already does), or keep it substantive and delete the duplication from the dedicated files. Recommend the hub pattern; RU already uses it (`advanced.md` is 17 lines of links there).

### Batch 4 — RU terminology + grammar sweep. ~1h.
- Normalize the storage term in RU **prose** to `хранилище`. Leave `vault` inside code/CLI/paths. Files: `local-quickstart.md`, `okf.md`, `fly.md`, `agent_admin.md`, `mcp.md`.
- Fix `forms.md:36` ("свой вёрстку" → "свою вёрстку") and `forms.md:7` (add the comma).
- Optional: break the 1,221-char wall-of-text paragraph in `perfomance.md:285`.

### Batch 5 — Parallelism (optional, lower priority). ~1-2h.
- The only true EN→RU hole is `sales-demo-api.md` (11 lines, low impact) — translate or intentionally skip.
- Inverted gap worth noting: RU is *more* complete than EN in several families (integrations: Confluence/WordPress/Quartz/Super.so/Obsidian Publish; `templates-advanced`, `templates-best-practices`, `jet.md`; 8 `Пример.*` use-case pages). If EN parity matters for onboarding/conversion, back-port the integration guides and use-case examples to EN. This is a content-creation task, not humanization — schedule separately.

**EN↔RU sync approach:** edit both languages in the same batch (never let one drift). For the `_index` pages and `monetization`, write RU natively from the intent, not as a translation of the reworked EN.

### Rough effort
Batches 1-4 (the real work) ≈ 6-8h total. Batch 5 is open-ended content work — treat as a separate initiative.

---

## Guardrails

**Do NOT change:**
- Technical accuracy: frontmatter field names, CLI flags, code samples, config keys, glob patterns, HTTP paths, table values. When a doc states a fact confidently (e.g. `frontmatter-patches.md`), do not "soften" it into hedged prose.
- The honest-limitation sections ("What's not implemented yet", "Common mistakes", the "not for" framings). They are a feature. Keep them.
- The status tagging in the README/OS table (shipped / branch / planned). Do not upgrade a `branch` claim to `shipped`.
- Working structure: the answer-first openings, mermaid diagrams, status tables, and numbered step blocks are already correct — do not restructure pages that already lead with the answer.
- RU `vault` inside code/CLI/paths — only the *prose* term normalizes to `хранилище`.

**Checks to run after edits:**
- `go generate ./onboarding-vault/` then `go run -tags dev ./cmd/server lint docs` — must stay clean for `en/user` and `ru/user` (baseline: only the three `demo/canvas_link_test.md` info-links). This catches broken cross-language links introduced by renamed pages or changed `lang_redirect`.
- Run the humanizer / anti-slop pass on any rewritten prose (no em/en dashes as habit, straight quotes, no delve/leverage/robust/seamless/comprehensive, no rule-of-three padding).
- For RU rewrites, apply the `clear-ru` / `_simple_text.md` checklist (answer-first, kill канцелярит, one thought per sentence).
- Verify every rewritten page still opens answer-first: a reader skimming only the first sentence of each page must still learn what they get.
- Spot-check that no batch introduced a new screenshot placeholder or TODO comment (grep the changed files for `🖼️` and `TODO`).
