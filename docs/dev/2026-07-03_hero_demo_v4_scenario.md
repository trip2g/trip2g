# Hero demo v4 — YouTube-style, cloud vehicle + open-source/cloud messaging (final scenario)

**Supersedes** `2026-07-03_hero_demo_v3_scenario.md`. v4 = v3's proven look and customization beats, but the recording vehicle changes: **no terminal, no memcli, no localhost**. The demo opens on the trip2g.com landing, clicks **"Try free"** → hosted cloud → one-click **"Open as Admin"** → a real https subdomain. Pacing is YouTube-style: open straight on the landing (no cold-open teaser at 2-min length), short VO lines, kinetic captions. Target **~120s** narrated MP4 + the same ~15s silent README GIF cut (wikilink→live).

**The message:** trip2g is **open source** — run it on your own computer or a server — **or** use the **free cloud playground**: one click, no setup, no guarantees. Three ways in; the viewer picks.

## Look / pipeline (carried from v3, minus the terminal)

- 2K (2560×1440), h264. macOS window frames composited over `autoproducer/assets/bg-network2-2k-dim.mp4` (looped, dimmed, no blur). **Static windows — no zoom/push-in.** One-time intro entrance kept.
- **Browser CSS viewport ~1300px** (v3 fix — verify `document.documentElement.clientWidth >= 1290` over CDP, no `.site-header__hamburger` visible). `--force-device-scale-factor=1.1`, chromium window ≈1430 OS-px wide. Breakpoints: ≥1280 = 3-col desktop, both sidebars inline (`assets/defaulttemplate/src/index.scss:888`).
- v3 stage geometry stands: `obsidian` x=20 y=60 w=1068 h=1010; `chromium` x=1108 y=60 w=1430 h=1010 (20px gap). One fewer window than v3 — **no terminal window at all** (this also kills the terminal-window bug class).
- Voice: owner's clone, ElevenLabs eng `tvApEij8Hu2VscOrlzzV`, model `eleven_v3`, sparse tags. More energy than v3's understated read, still dev-to-dev — **not hype-bro**.
- Music: **none** (VO-forward). The two interaction SFX from v3 stay: success blip on push, soft tick on live-update. Kinetic captions (below) carry the "YouTube energy" instead of a music bed.
- Honesty: real product only, no mockups. Every beat is a shipped feature (customization beats verified against `internal/defaulttemplate/template.go` in v3).

## The cloud vehicle (verified flow)

1. Open **trip2g.com** landing briefly — the header CTA is literally **"Try free"** (`docs/_header.md` → `https://simplecloud.2pub.me`; also on the try-now and pricing sections). This is the on-camera path from landing to cloud.
2. `simplecloud.2pub.me` dashboard → click **"Open as Admin"** (HAT one-click).
3. Lands **already authenticated as admin** on the space's real **https** subdomain, on the empty-state "Welcome! … Download archive" onboarding page. No login typing, no DEV code, no localhost.
4. The cloud assigns a **random subdomain = adjective + writer's name** (e.g. `honesthesse.2pub.me` = honest + Hesse). We keep it and call it out — it's charming and it's honest.

**Honesty line (on camera + VO):** this is a **free cloud playground — early, best-effort, no guarantees**. Say it plainly. It buys trust and pre-empts "is this production?" comments.

### Retention math: why cloud beats the v3 opening

v3 spent 0–24s on terminal boot + login + DEV code — the weakest retention stretch and the biggest recording cost (see `2026-07-03_recording_session_timeprofile.md`: login rehearsal, dev-code archaeology, docker DEV flap). v4 replaces all of it with two clicks on real pages and lands the viewer on a live https domain by ~0:12.

## Target length: ~120s

- YouTube product demos hold best at 90–150s; v3 proved the content fits 115s.
- v4 removes ~16s of terminal/login but adds ~10s of landing→cloud + honesty beat and ~5s of a stronger three-ways-in hook. Net ≈120s.
- Retention hooks (the moments that stop the scroll / re-hook mid-video): **0:00 the "one click + open source" promise on the landing**, **0:32 first live push**, **0:44 wikilink page-from-nothing**, **1:16 the serif re-render** ("the whole look flipped from one line"). The beat sheet paces so none of these is more than ~25s from the previous one.

## Beat sheet (~120s)

| t (s) | Beat | Narration promise | On screen |
|---|---|---|---|
| 0–12 | Open on the landing (hook) | "Your Obsidian vault becomes a real website — and it's open source. Run it on your computer, a server… or the free cloud." | trip2g.com landing in frame from t0 (NO cold-open teaser — dropped for a 2-min cut). Caption: **"open source · self-host · or one click"**; cursor to **"Try free"**, click. Caption: **"free cloud playground — no guarantees"** |
| 12–20 | Open as Admin | one click, you're the admin of a real site | simplecloud dashboard → **"Open as Admin"** → lands on `https://<adjective><writer>.2pub.me` empty-state Welcome page. Callout on the URL bar: **"random subdomain: adjective + a writer"**. Brief labels/arrows on the three admin header icons: **person = your account · gear = admin panel · magnifier = search** (≤2s, shares the beat) |
| 20–28 | Download starter vault | the plugin's already inside | click "Download archive" on the Welcome page; unzip flash-cut; Obsidian window enters (one-time entrance) |
| 28–38 | Edit `_index`, save → **live** | save in Obsidian, the site updates itself | edit `_index.md` → Ctrl+S → "Pushed 1 files" (success blip) → homepage updates live (tick). Caption: **"live"** |
| 38–52 | **Wikilink → new page (GIF cut)** | pages from links, nothing else — plus the publish flag | type `[[About]]` → Ctrl+click → `About.md` created → write + add `free: true` in frontmatter → save → About appears on the site → open it. Caption: **"private by default — `free: true` publishes"**. This 14s window is the silent README GIF |
| 52–62 | `_header.md` → nav | your navigation is a note — the **standard template** renders it | edit `_header.md` (markdown list of links), save → nav bar appears site-wide |
| 62–74 | Sidebars | one line of frontmatter — the **standard template** does the rest, no coding | open `Deep Work.md`, add `left_sidebar`/`right_sidebar` frontmatter, save → **both sidebars inline** (TOC left, links right — the 1300px viewport payoff) |
| 74–90 | Custom Jet layout (AST + cross-note) | beyond the standard template, the whole look is yours — and the template can read your note's structure and pull in other notes. Caption: **"your template"** | apply `layout: showcase/magazine` to a note that has a few `##` sections (pre-staged) → save → a **magazine layout** appears: hero + **numbered feature rows built from the note's own sections** (`PartialRenderer().Sections(2)`) + a **cross-note CTA banner** (pulled from `_snippet-cta` via `nvs.ByWikilink`). Then **edit `_snippet-cta.md`** → the banner updates live — the template pulling another note, on camera. (retention hook: the full-layout flip + the live cross-note) |
| 88–102 | Digital garden + real proof | keep going → a garden; the docs run on trip2g | sped-up montage: 2–3 garden notes; fast-scroll trip2g.com/en/user (~10 pages) |
| 102–120 | Outro CTA — both doors | open source: star it, self-host it. Or the playground: one click. Agents: MCP | end-card on network bg: `Publish · Push · Connect` / **⭐ github.com/trip2g/trip2g** / **Try free → simplecloud.2pub.me** / `connect your agent → trip2g.com/_system/mcp` |

Kinetic captions total five: "open source · self-host · or one click", "free cloud playground — no guarantees", "live", "private by default — `free: true` publishes", "your template". Short, lower-third, ≤2s each. No more — tasteful, not TikTok.

**Customization arc (owner framing):** nav, sidebars, TOC, backlinks are all **standard-template features** — frontmatter-driven, zero coding, out of the box. The custom Jet layout beat is the deliberate contrast: the escape hatch when you want to go beyond what the standard template gives you. The VO carries this framing with one clause per beat (52–62, 62–74, 74–88).

## Concrete on-camera file contents (reused from v3, pre-stage in starter-pack where noted)

### `_header.md` (edited on camera, beat 52–62)
```markdown
---
title: Navigation
free: true
---

- [Home](/)
- [About](/about)
- [Deep Work](/deep-work)
```
Renders to a top `<header>` nav. `template.go` HeaderRef() auto-detects `_header`. Plain list, no logo — keeps the beat short.

### `About.md` (created via wikilink on camera, beat 38–52)
```markdown
---
free: true
---

# About

I write about focus and deep work. This site is my digital garden.
```
Pages are **hidden by default** — a page is public only with `free: true` in frontmatter (109 of 141 `docs/demo` notes set it; e.g. `docs/demo/article-with-toc.md`). Without it, About would only show because the admin sees hidden pages — a visitor would get nothing. The flag is taught on camera in this beat.

### `Deep Work.md` (PRE-STAGE body; add frontmatter on camera, beat 62–74)
Body pre-staged (**≥5 headings on purpose** — the TOC auto-shows in `auto` mode only at ≥5 headings / ≥2min read per `internal/model/note.go:530`; a longer note means NO `toc:` key is needed and the `left_sidebar: [TOC]` widget fills):
```markdown
# Deep Work

## What it is
Focused, undistracted work on a cognitively demanding task.

## Why it matters
It compounds. The ability to focus is getting rarer and more valuable.

## How I practice it
Time-blocked mornings, phone in another room. See [[About]].

## What breaks it
Notifications, open tabs, and meetings that could have been a note.

## What it gives back
Fewer hours, better output — and the rare feeling of actually finishing.
```
On camera, add to the top:
```yaml
---
title: "Deep Work"
free: true
left_sidebar:
  - TOC
  - inlinks
right_sidebar:
  - outlinks
content:
  - selfcontent
---
```
Result at ≥1280 viewport: TOC + backlinks left, outlinks right, both inline. Frontmatter shape matches `docs/demo` (`article-with-toc.md`, `full-layout.md`, `with_right_sidebar.md`) — `content: [selfcontent]` included. Note: `toc:` IS a real key (`toc: show`/`auto`, see `docs/demo/toc_test.md`, `complex_content.md`) — we deliberately DON'T set it here and instead give the note ≥5 headings so `auto` mode fills the TOC on its own (owner's call, keeps the frontmatter minimal on camera).

### The magazine layout (PRE-STAGE; shown + wired on camera, beat 74–90)
Use the **live-verified** showcase layout `docs/demo/_layouts/showcase/magazine.html` (authored + rendered on a HEAD instance by tmpldemo; full-page screenshot confirmed). It reads the note's structure and pulls in another note — the two capabilities the owner wants shown. The core of it (real Jet, no invented API):
```jet
{{ defaultTemplate.Styles() }}   {{-* the design-system CSS: --pico-* vars, dark mode free *-}}
...
{{ defaultTemplate.Header() }}
<header class="mg-hero">
  <h1>{{ note.Title() }}</h1>
  <div class="mg-hero__lead">{{ note.PartialRenderer().Introduce().ContentHTML }}</div>
</header>
<main class="mg-rows">
  {{ range i, sec := note.PartialRenderer().Sections(2) }}   {{-* AST: one feature row per H2 section *-}}
  <article class="mg-row"><div class="mg-row__num">{{ i+1 }}</div><h2>{{ sec.TitleHTML }}</h2>
    <div class="mg-row__body">{{ sec.ContentHTML }}</div></article>
  {{ end }}
</main>
{{ cta := nvs.ByWikilink("_snippet-cta") }}   {{-* cross-note: pull another note *-}}
{{ if cta }}<aside class="mg-cta">{{ cta.HTMLString() }}</aside>{{ end }}
```
On camera: apply `layout: showcase/magazine` to a note that has a few `##` sections (pre-stage that note + `_snippet-cta.md` into the golden vault) → save → hero + numbered feature rows (from the note's own sections) + the CTA banner appear. Then **edit `_snippet-cta.md`** → the banner updates live (the template pulling another note, on screen).

Verified facts (from tmpldemo's live renders): custom `_layouts/*.html` are **Jet** (CloudyKit); loops are `{{ range i, sec := ... }}` (single-var range yields the INDEX; `_` as a range var PANICS the server — a real bug, filed below); layouts compile with `jet.WithSafeWriter(nil)` so raw HTML needs **no `| unsafe` pipe** (it's a harmless no-op if present); `defaultTemplate.Styles()/Header()/Footer()/UserSpaceScripts()` **DO exist** (`internal/case/rendernotepage/userspace.go:185` + `endpoint.go:410`) — the earlier "they don't exist" note was wrong; `Styles()` links the Pico-based design system so `--pico-*` CSS vars are available. `layout: showcase/magazine` → `_layouts/showcase/magazine.html` (no leading slash, no `.html`). Companion demo layouts + a cookbook live at `docs/demo/_layouts/showcase/` + `docs/demo/layout-functions.md`.

## VO script (eleven_v3, sparse tags — punchy, short lines, owner's voice)

[0–4] [warm] Hi guys. Let me show you trip2g. <break time="0.3s" /> Your Obsidian vault — turned into a real website. And it's open source.

[4–12] You can run it on your own computer. Or a server — the code's on GitHub. <break time="0.4s" /> Or skip all that <break time="0.2s" /> and use our free cloud playground. It's early, no guarantees — but it's one click.

[12–20] One click. <break time="0.3s" /> I'm the admin of a real site, on a real domain. [warm] The cloud names it after a writer — I got Hesse today. <break time="0.3s" /> Up top: your account, the admin panel, and search.

[20–28] It hands you a starter vault. A normal Obsidian vault <break time="0.3s" /> with the sync plugin already inside.

[28–38] Open it. Edit the home page. Hit save. <break time="0.4s" /> That's it — the site just updated. Live.

[38–52] New pages? Just link them. Click the link, Obsidian makes the note. Write — <break time="0.2s" /> and to publish it, one line: free, true. Pages stay private until you say so. <break time="0.3s" /> Save, and it's on the site.

[52–62] Navigation is a note too. List your links in `_header`, save <break time="0.3s" /> and the built-in template turns it into your menu.

[62–74] Sidebars? One line of frontmatter. Table of contents, backlinks <break time="0.3s" /> the standard template renders it all. No coding.

[74–88] And when the standard template isn't enough <break time="0.2s" /> the look is yours. Drop an HTML layout into the vault, point a page at it <break time="0.4s" /> and it renders your way. Completely.

[88–102] Keep writing and it grows into a digital garden. This docs site you're scrolling? <break time="0.3s" /> Runs on trip2g.

[102–120] So — three ways in. Star it on GitHub and run it yourself. <break time="0.3s" /> Or hit try-free and play in the cloud. <break time="0.4s" /> And if you've got an AI agent — it can connect too, over MCP. [warm] Link's on screen.

Delivery note: faster read than v3, sentences land on cuts. Beats 38–52 now carry the free-flag teach, so the VO fills more of the GIF-cut window — the footage still breathes on the final "Save, and it's on the site" landing. The silent README GIF is cut from the same footage without VO, so the extra line costs the video nothing.

## Speed levers for this recording

Cloud removes the three biggest costs from the time-profile: no terminal/memcli boot, no login/DEV-code beat, no reset harness fighting root-owned docker volumes — and no terminal window means the terminal-window bug class is gone entirely.

1. **State reset between takes:** either record on a **fresh space** (empty state costs one "Open as Admin" click), or clean the current one with a `hideNotes(paths)` mutation via api-key (`internal/graph/schema.graphqls:2101`) — scriptable, no dashboard clicking. Fresh space is simpler; hideNotes is for mid-session partial resets.
2. **`coords.json` first.** Only two windows now (Obsidian + Chromium at the v3 geometry) plus the landing/dashboard clicks. Map the landing "Try free" CTA, dashboard "Open as Admin", and Welcome-page download button; validate with one screenshot before rolling.
3. **Pre-stage files, don't type them** — v3 rule stands: `Deep Work.md` body, `_layouts/custom/page.html`, garden notes go into the starter-pack; only the visible edits (header links, About's `free: true`, sidebar frontmatter, `layout:` line) happen on camera, written to disk with an xdotool cursor/save moment. Kills the Obsidian list-autocomplete retake class.
4. **Per-segment structure stands** (`run-all.sh`): new segments are landing→cloud (beats 0–20); everything from beat 20 on reuses v3 segment logic at the same geometry. Reuse v3's per-beat mp3s only where lines are unchanged (few are — most VO re-renders, it's cheap, ~1 min).
5. **Network honesty:** the cloud is best-effort — do one dry-run push to confirm sync latency on the hosted box before takes, so a slow "Pushed 1 files" doesn't eat a take. If latency is visible, cut around it (save → cut → site updated) rather than faking it.
6. Keep VO/endcard/post.py as committed parameterized scripts (v3 lever 5) so timing tweaks don't re-trigger takes.

## Open choices (flag for the owner)

1. **Random subdomain on camera: keep it.** Recommendation — the adjective+writer name is memorable and supports the honesty framing ("playground"). Alternative: pre-provision a branded space (e.g. `demo.2pub.me`) if the random draw lands an awkward name; decide after seeing the actual draw on recording day.
2. **Landing beat length:** 4–12s assumes the landing reads instantly at 1300px viewport. If the mesh landing needs a scroll to reach "Try free", cut to the header CTA directly — don't spend seconds scrolling.
3. **Landing "try free" → login-page gap:** the link audit (`2026-07-02_landing_link_audit.md`) notes simplecloud lands on a bare login form. On camera we'll be pre-authenticated, so it won't show — but a first-time viewer following the CTA hits that form. Worth fixing before the video ships traffic at it.
