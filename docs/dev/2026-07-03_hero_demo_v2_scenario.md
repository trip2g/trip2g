# Hero demo v2 — onboarding tour (final scenario)

**Target:** ~78s narrated MP4 (landing page / YouTube / Show HN "how it works") + a tight ~15s silent README GIF cut (the wikilink→live moment).
**Look:** Option B (macOS window frames — traffic lights, rounded corners, soft shadows) over the **slowed network background** (`autoproducer/assets/bg-network-slow5x.mp4`), **dimmed ~40-50% + 3px blur** so the product windows are the hero. 2K (2560×1440), h264 crf 17.
**Voice:** ElevenLabs eng `tvApEij8Hu2VscOrlzzV` (the owner's cloned voice, parameterized). Sincere, plain, no hype/marketing tone. **Use model `eleven_v3`** so the inline audio/delivery tags in the VO script below render (`[calm]`, `[pause]`, `[warm]`, `...`). Keep tags MINIMAL — a dev audience hates performative reads. **Fallback:** if the cloned voice isn't v3-enabled or v3 sounds worse, strip the tags and use `eleven_multilingual_v2` (the plain script still reads fine).
**Scope:** publishing only (vault → website via the plugin). The MCP/agent/token-economy angle is a SEPARATE video — do not include it here.
**Honesty:** real product only — real trip2g (memcli), real Obsidian + real trip2g-sync plugin auto-push, real browser live-update. No mockups; bg/frame/end-card are the only decoration.

## Beat sheet (narration + action)

| t (s) | Narration | On screen |
|---|---|---|
| 0–8 | "This is trip2g. It turns your Obsidian vault into a website. You host it yourself... laptop, or a server." | Brief terminal: start trip2g locally (memcli up / `docker compose up`), then a browser opens `localhost`. |
| 8–14 | "It's up. Open it in the browser and sign in as admin." | Login page → **DEV code `111111`** → admin view. (verified: 111111 — 000000 in local-quickstart.md is a doc bug) |
| 14–24 | "There's a starter vault you can download. It's a normal Obsidian vault, and the sync plugin's already inside." | Admin → **download the onboarding vault** (button → zip/folder). |
| 24–32 | "Open it in Obsidian. The plugin's right there, enabled. Nothing to install." | Cut past unzip; Obsidian open on the vault. Settings → Community plugins → **trip2g sync installed + enabled**. |
| 32–42 | "Now edit the home page and hit save. The plugin pushes it automatically, and the site updates right there." | Edit **`_index.md`** (a line/heading) → Ctrl+S → "Pushed 1 files" Notice → browser homepage updates live. |
| 42–56 | "New pages work the same way. Type a link to a page that doesn't exist, click it, and Obsidian creates the note. Write something, save... and it's on the site." | In `_index.md` type `[[About]]` → Ctrl+click → Obsidian creates `About.md` → type an about-me page → save → refresh browser → About link on homepage → click → the About page. **(this ~15s is the README GIF cut)** |
| 56–60 | "You can swap in your own template, too." | Short punch line — can float as an on-screen caption over the next beat's opening; keep it one breath. |
| 60–70 | "Keep going and it turns into a digital garden. Nav is a few lines in a header file. And this docs site runs on trip2g too." | **Sped-up montage, two parts:** (a) quick-author 2–3 digital-garden notes + edit `_header.md` nav; (b) then **fast-scroll through the real trip2g docs site** (`trip2g.com/en/user`, ~10–20 pages flying by) — dogfood, proving a real full site, not a toy. |
| 70–80 | "Publish with the plugin, push from the CLI, or connect your agent through the API. More at trip2g.com... and a star on GitHub helps." | Outro: montage settles → **CTA end-card** on the network bg — three lines: `Publish · Push · Connect`, then `github.com/trip2g/trip2g` + ⭐, `trip2g.com/en/user`, and **`connect your agent → trip2g.com/_system/mcp`**. |

## Digital-garden content for the montage (beat 56–68)
A believable personal knowledge garden, NOT lorem. Suggested ~10 notes (short, real-sounding titles + one line each), e.g.:
`Now`, `Reading list`, `Bookmarks`, `Recipes`, `Running log`, `Book notes/Deep Work`, `Projects`, `Til (today I learned)`, `Quotes`, `Contact`. Wikilink a few together so the graph feels alive. `_header.md` nav: Home · Garden · About · Now.

## Notes for the recorder
- **Pacing:** beats 0–5 (0–56s) real-time; beat 6 (10 pages) MUST be sped-up montage or it drags; outro real-time.
- **Cursor:** use `autoproducer/scripts/hero/smoothmove.py` for any eased cursor moves + a subtle click ripple; or keyboard-only where clean.
- **Terminal (opening only):** `kitty` (color emoji, clean), not xterm.
- **Background:** scale `bg-network-slow5x.mp4` to 2K, `boxblur=3` + brightness/opacity down ~40-50%, loop under the framed windows.
- **Attention direction (cursor alone is too weak):** in post, guide the eye with — (1) **smooth zoom/punch-in** to the action region per beat (ffmpeg zoompan / keyframed scale+crop to the (x,y,t) of what's happening; hold, then ease back) — this is the biggest win, Screen-Studio style; (2) a **soft glow following the cursor** + a subtle **click ripple**; (3) optional **spotlight dim** of the inactive window / a soft vignette around the active region; (4) a brief **callout** (glow box / underline / small arrow) on one or two key UI elements (e.g. "trip2g sync" in Settings). Use sparingly — zoom + cursor glow carry most of it.
- **Background music: NONE** (competes with the VO — dropped on purpose; clean VO reads better here).
- **Interaction SFX (recorder generates these):** generate 2 tiny sounds via the **ElevenLabs Sound Effects API** (`text-to-sound-effects`, key on file) and place them on the reward beats: a soft "publish/success" blip on the "Pushed 1 files" Notice (~32-42s), and a quiet "tick" on the browser live-update / the About link appearing (~40-56s). Prompts e.g. `"soft short UI success blip, gentle warm single note, subtle"` and `"very short soft tick, minimal UI confirmation, quiet"`. **Skip per-keystroke typing sounds** — fiddly and reads fake; only the confirmation moments. Barely audible, mixed well under the VO. (CC0 Pixabay/Freesound is a fallback if generation is off.)
- **README GIF:** cut the 42–56s wikilink→live segment, silent, ~15s, ≤5MB, 1000-1280px.
- **RU cut:** later (needs re-timed beats for RU VO; ru voice `87ifZMBSxa6PWYnKMWiH` on file).

## VO script (read this)

Model `eleven_v3`. Syntax: `[emotion]` delivery tags + `<break time="Xs" />` for precise pauses (use the breaks to land the VO on the visual beats). **Register = sincere, understated, dev-to-dev — NOT hype.** Tags are deliberately sparse (`[calm]` open, `[warm]` close only); do NOT add excited/screaming energy. If v3 misbehaves, strip both `[...]` and `<break/>` and use `eleven_multilingual_v2` with plain `...` pauses.

[0–8] [calm] This is trip2g. <break time="0.4s" /> It turns your Obsidian vault into a website. You host it yourself <break time="0.3s" /> laptop, or a server.
[8–14] It's up. Open it in the browser and sign in as admin.
[14–24] There's a starter vault you can download. It's a normal Obsidian vault, and the sync plugin's already inside.
[24–32] Open it in Obsidian. The plugin's right there, enabled. <break time="0.3s" /> Nothing to install.
[32–42] Now edit the home page and hit save. The plugin pushes it automatically <break time="0.5s" /> and the site updates right there.
[42–56] New pages work the same way. Type a link to a page that doesn't exist, click it, and Obsidian creates the note. Write something, save <break time="0.4s" /> and it's on the site.
[56–60] You can swap in your own template, too.
[60–70] Keep going and it turns into a digital garden. Nav is a few lines in a header file. And this docs site? <break time="0.3s" /> Runs on trip2g too.
[70–80] Publish with the plugin, push from the CLI, or connect your agent through the API. More at trip2g.com <break time="0.4s" /> [warm] and a star on GitHub helps.
