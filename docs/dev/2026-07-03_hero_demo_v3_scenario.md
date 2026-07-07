# Hero demo v3 — onboarding + customization tour (final scenario)

**Supersedes** `2026-07-03_hero_demo_v2_scenario.md`. v3 = v2 look/pipeline + three new customization beats (`_header.md` → nav, sidebar, custom layout) + a **browser-viewport fix**. Target ~115s narrated MP4 + the same ~15s silent README GIF cut (wikilink→live).

**Look:** unchanged from v2 — Option B (macOS window frames) over `autoproducer/assets/bg-network2-2k-dim.mp4` (no blur, looped, dimmed). 2K (2560×1440), h264, **static windows (no zoom/push-in — v2fix)**, one-time intro entrance kept.
**Voice:** ElevenLabs eng `tvApEij8Hu2VscOrlzzV`, model `eleven_v3`, sparse `[calm]`/`[warm]` tags. Sincere dev-to-dev, no hype.
**Honesty:** real product only. No mockups. All customization beats use shipped features (verified against `internal/defaulttemplate/template.go`).

## CRITICAL: browser viewport fix (root cause of the "mobile version" bug)

The v2 browser was launched `--force-device-scale-factor=1.4` at window width 1120 → CSS viewport ≈ **800px** → tripped the template's `max-width: 1023px` mobile breakpoint (hamburgers, collapsed nav). **The default template breakpoints (`assets/defaulttemplate/src/index.scss:888`):**
- **≥1280px viewport** → full 3-column desktop layout, **both sidebars inline, zero hamburgers**.
- **1024–1279px** → left sidebar inline, right sidebar becomes a drawer (a right hamburger appears).
- **<1024px** → both sidebars drawers, both hamburgers (the v2 bug).

**v3 target: browser CSS viewport ~1300px** (`document.documentElement.clientWidth ≥ 1290`, safely over the 1280 line so the sidebar beat shows both sidebars inline). Achieve by lowering DSF + widening the window:
- Recommend `--force-device-scale-factor=1.1` with chromium window width ≈ **1430 OS-px** (1300 × 1.1). **Verify over CDP** that `document.documentElement.clientWidth >= 1290` on the homepage and that **no `.site-header__hamburger` is visible**; nudge window width if the scrollbar shifts it.
- Rebalance the stage so the wider browser fits without overlapping Obsidian, and **reduce the inter-window padding** (user ask). Proposed `coords.json` geometry (validate + screenshot after reset):
  - `obsidian`: x=20, y=60, w=1068, h=1010 (ends 1088)
  - `chromium`: x=1108, y=60, w=1430, h=1010 (ends 2538) — 20px gap
- In `post.py`, tighten the macOS-frame gap/padding to match the new geometry so the two framed windows sit closer and the browser reads wider.

**Consequence:** every browser beat looks different at 1300 vs 800 (desktop nav, wider content) → all browser footage is re-recorded. Obsidian footage geometry also shifts (narrower) → re-record Obsidian beats too. This is a near-full re-record; apply the speed levers below.

## Beat sheet (~115s)

| t (s) | Narration beat | On screen |
|---|---|---|
| 0–8 | intro | terminal `memcli up` → browser opens `localhost:PORT` (desktop layout now) |
| 8–14 | sign in admin | login → DEV code `111111` → admin |
| 14–24 | download starter vault | admin → download onboarding vault (plugin already inside) |
| 24–32 | open in Obsidian | Obsidian on the vault, trip2g sync plugin enabled |
| 32–42 | edit `_index`, save → live | edit `_index.md` → Ctrl+S → "Pushed 1 files" → homepage updates live |
| 42–56 | wikilink → new page (**GIF cut**) | type `[[About]]` → Ctrl+click → create `About.md` → write → save → About on homepage → open it |
| **56–66** | **`_header.md` → nav** | open/edit **`_header.md`** (a markdown list of links), save → a **nav bar appears at the top** of every page |
| **66–78** | **sidebar** | open a longer note (**`Deep Work.md`**, has `##` headings), add frontmatter `left_sidebar: [TOC, inlinks]` + `right_sidebar: [outlinks]`, save → **both sidebars appear inline** (TOC left, links right) |
| **78–90** | **custom layout** | show pre-staged **`_layouts/custom/index.html`** (a tiny Jet wrapper), add `layout: /custom/index` to a note, save → note **re-renders in the custom look** (serif, narrow column) |
| 90–102 | digital garden + docs | sped-up montage: quick-author 2–3 garden notes; then fast-scroll the real trip2g docs site (`trip2g.com/en/user`, ~10 pages) |
| 102–115 | outro CTA | end-card on network bg: `Publish · Push · Connect`, `github.com/trip2g/trip2g` + ⭐, `trip2g.com/en/user`, `connect your agent → trip2g.com/_system/mcp` |

## Concrete on-camera file contents (pre-stage in starter-pack where noted)

### `_header.md` (edited on camera, beat 56–66)
```markdown
---
title: Navigation
free: true
---

- [Home](/)
- [About](/about)
- [Deep Work](/deep-work)
```
Renders to a top `<header>` nav (list → links). `template.go` HeaderRef() auto-detects `_header`. Keep it a plain list (no logo to keep the beat short).

### `Deep Work.md` (PRE-STAGE the body in starter-pack; add the sidebar frontmatter on camera, beat 66–78)
Body pre-staged (a few short sections so TOC has entries):
```markdown
# Deep Work

## What it is
Focused, undistracted work on a cognitively demanding task.

## Why it matters
It compounds. The ability to focus is getting rarer and more valuable.

## How I practice it
Time-blocked mornings, phone in another room. See [[About]].
```
On camera, add to the top:
```yaml
---
title: Deep Work
left_sidebar:
  - TOC
  - inlinks
right_sidebar:
  - outlinks
---
```
Result at ≥1280 viewport: left column shows the TOC (What it is / Why it matters / How I practice it) + backlinks; right column shows outlinks. This is why the viewport must be ≥1280.

### `_layouts/custom/index.html` (PRE-STAGE in starter-pack; shown + wired on camera, beat 78–90)
```jet
{{ defaultTemplate.Styles() }}
<main style="max-width:640px;margin:3rem auto;padding:0 1rem;font-family:Georgia,'Times New Roman',serif;font-size:20px;line-height:1.7">
  {{ defaultTemplate.Header() }}
  <h1>{{ note.Title }}</h1>
  {{ note.HTMLString() | unsafe }}
  {{ defaultTemplate.Footer() }}
</main>
```
On camera: briefly show this file, then add `layout: /custom/index` to a note's frontmatter (reuse `About.md`), save → About re-renders serif + narrow. Punchy visible change.

## VO script (model eleven_v3, sparse tags)

[0–8] [calm] This is trip2g. <break time="0.4s" /> It turns your Obsidian vault into a website. You host it yourself <break time="0.3s" /> laptop, or a server.
[8–14] It's up. Open it in the browser and sign in as admin.
[14–24] There's a starter vault you can download. It's a normal Obsidian vault, and the sync plugin's already inside.
[24–32] Open it in Obsidian. The plugin's right there, enabled. <break time="0.3s" /> Nothing to install.
[32–42] Edit the home page and hit save. The plugin pushes it automatically <break time="0.5s" /> and the site updates right there.
[42–56] New pages work the same way. Link to a page that doesn't exist, click it, Obsidian creates the note. Write, save <break time="0.4s" /> and it's on the site.
[56–66] Your navigation is just a note. Edit `_header`, list your links, save <break time="0.3s" /> there's your header.
[66–78] Want a sidebar? One line of frontmatter — a table of contents, backlinks <break time="0.3s" /> and the page grows one.
[78–90] And the whole look is yours. Drop an HTML layout in the vault, point a note at it <break time="0.3s" /> and it renders your way.
[90–102] Keep going and it turns into a digital garden. And this docs site? <break time="0.3s" /> Runs on trip2g too.
[102–115] Publish with the plugin, push from the CLI, or connect your agent through the API. More at trip2g.com <break time="0.4s" /> [warm] and a star on GitHub helps.

## Speed levers for this re-record (from `2026-07-03_recording_session_timeprofile.md`)

The scripted reset (`reset-stage.sh` + `starter-pack/`) and `coords.json` already exist — USE them, don't re-derive. Specific to this run:
1. **Update `coords.json` for the new geometry FIRST**, validate with one screenshot, then record — don't blind-click.
2. **Pre-stage the new files** (`Deep Work.md` body, `_layouts/custom/index.html`, and a garden note or two) into `starter-pack/` so they're present after reset; only the *edits* (header links, sidebar frontmatter, `layout:` line) happen on camera.
3. **Write edits to disk, don't `type` markdown** — avoids the Obsidian list-autocomplete / `[[`-popup retake class. For frontmatter, writing the file + letting the plugin push is cleanest; use `xdotool` only for the visible cursor/save moment.
4. **Only re-record what changed**: keep `run-all.sh`'s per-segment structure; add segments for header/sidebar/template; re-shoot the browser+obsidian segments at the new geometry. Reuse VO per-beat mp3s (only new beats need generation).
5. Keep content (VO, endcard, post.py) as committed scripts so a timing tweak doesn't re-trigger takes.
