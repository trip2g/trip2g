# v1 onboarding demo: scenario, narration, and rig (Option A — picom live compositing)

**TL;DR.** The v1 demo shows plain onboarding in ~20 seconds of real 2K screen capture: create your first page in Obsidian → the **trip2g sync plugin** pushes it automatically on save → the signed-in browser (parked with the real `live_follow` feature) jumps to the freshly published page by itself. No terminal, no CLI — the plugin and the product do everything on camera. Narrated MP4 `docs/demo-video/v1-optA-live.mp4` (23.6 s, 2560×1440, ends on a GitHub CTA card) + silent README GIF `docs/demo-video/v1-optA-live.gif` (13 s, 1100 px, 0.18 MB). Reproducer: `autoproducer/scripts/hero-v1/run-all.sh`. Every product pixel is real (honesty rule from the launch kit); the only non-product visuals are the wallpaper and the closing CTA card.

## The v1 story (plain onboarding)

Get a vault → it's already a site → create your **first page** (title + one line) → save → it's live. Richer beats (MCP, agents, two-way sync) are for later cuts.

## Rig (all real, Xvfb `:99`, 2560×1440, dwm + picom)

| Piece | What it is |
|---|---|
| Server | **memcli** instance `demo-v1`: `node cli/memcli/dist/memcli.js up --folder autoproducer/demo-vault --name demo-v1 --port 24181 --image trip2g:local --no-hub --no-seed` (docker container `trip2g-memory-demo-v1`, web `http://localhost:24181`). The memcli CLI sync watcher is killed — the **Obsidian plugin** does the syncing. |
| Vault | `/home/alexes/projects2/autoproducer/demo-vault/` — minimal, isolated: `_index.md` (home), `_header.md` (mounts the site chrome incl. live features), `_config/publish-everything.md` (a real `type: frontmatter-patch` note with `include: ["**"]` + `{ free: true }` so every note is public). |
| Left window | **Obsidian 1.12.7** (real AppImage, isolated `XDG_CONFIG_HOME`, dark theme, scale 1.5) with the demo vault and the real **trip2g sync plugin 0.6.1** enabled, `autoSyncOnSave: true`. Core `sync`/`publish` plugins disabled (kills the red "Uninitialized" badge). |
| Right window | **Chromium** (scale 1.75) signed in once via memcli's HAT login URL, parked on `http://localhost:24181/?#!live_follow=1` — the product's real live-follow ("cinema mode"). |
| Look | Option A, live compositing: floating windows with 48 px padding, **picom** (xrender; rounded corners 14 px + soft shadows), **feh** gradient wallpaper (`#1b2733→#0d1117`, ImageMagick-generated), dwm bar hidden (`alt+b`), cursor parked on the wallpaper. picom + feh run from `autoproducer/vendor/{picom-root,feh-root}` (extracted .debs — no sudo on this box). |

## Why the browser is signed in (honesty note)

`noteChanges` (the SSE subscription behind live-reload/live-follow) rejects anonymous subscribers by design (`internal/graph/note_changes_acl.go`). The demo browser is the **owner's** browser, signed in via memcli's HAT link — exactly what `memcli open` produces for a real user. Anonymous visitors see the published page; the live-follow beat is an owner-session feature.

## Narration (ElevenLabs `eleven_multilingual_v2`, voice **parameterized**; 20.16 s)

> My site runs on trip2g. And it is just an Obsidian vault. Let me create the first page: a title, a line of text. Save. The plugin syncs it automatically — and the browser is already on the new page. Live on my own server, seconds after typing. That's trip2g. Your notes, published. It's open source — try it, and if you like it, star it on GitHub.

- English render (this cut): voice `tvApEij8Hu2VscOrlzzV`.
- Voice swap is ONE command: `VOICE_ID=<id> node autoproducer/scripts/hero-v1/vo.mjs .` then rerun `run-all.sh` (beats are timed against the new `vo-align.json` word timestamps — re-check the beat table if the pacing shifts).
- Russian voice on file for a possible RU cut: `87ifZMBSxa6PWYnKMWiH` (not rendered in v1).

## Beat sheet (times = VO seconds; the audio is the clock)

| t (s) | VO says | On screen |
|---|---|---|
| 0–4.3 | "My site runs on trip2g. And it is just an Obsidian vault." | static scene: Obsidian (empty New tab) left, site home right |
| 4.6 | "create the first page" | `Ctrl+N` |
| 5.85 | "a title" | type `My first page` |
| 6.7 | "a line of text" | `Enter`, type `This page went live the moment I saved it.` |
| 7.95 | "Save." | `Ctrl+S` |
| ~10.5–11.5 | "the plugin syncs it automatically — and the browser is already on the new page" | **product acts, not the script**: plugin auto-push fires (debounce 2.5 s), Notice "Pushed 1 files to server", live_follow navigates the browser to `/my_first_page` |
| 12.2–16.9 | "Live on my own server… Your notes, published." | hold on the live page |
| 16.9–20.2 | CTA line | cross-fade to the end card (`endcard.png`: repo URL + star invite) |

## Cuts

- **Narrated MP4** (landing/social): `docs/demo-video/v1-optA-live.mp4` — 2560×1440, 23.6 s, crf 17 `veryslow`, AAC 160k, CTA end card.
- **README GIF** (silent): `docs/demo-video/v1-optA-live.gif` — VO ≈2→15 s (typing → save → push Notice → browser jumps), 12 fps, 1100 px, 0.18 MB (≤5 MB budget; ffmpeg palette — gifski is not apt-installable on this box).
- Raw 2K capture: `docs/demo-video/v1-optA-live-raw.mkv`; VO: `docs/demo-video/v1-vo.mp3`.

## Reproduce

```bash
/home/alexes/projects2/autoproducer/scripts/hero-v1/run-all.sh /tmp/v1-out
```

One-time setup is documented in the script header (memcli up, plugin install into the vault, vendor picom/feh, Obsidian trust click, HAT sign-in, VO generation). Per-take the script: hides + content-scrambles the demo note server-side → deletes it locally → restarts Obsidian (`pkill -f mount_obsidi` — AppImage processes don't match the launcher path) → clears the plugin's localStorage sync-state via CDP (`obs-reset.mjs`) → re-arranges/verifies floating windows → CDP-navigates the browser (`cdp-nav.mjs`) → starts picom → records (`record.mjs`, VO-clocked beats) → encodes MP4+GIF.

## Hard-won gotchas (for the next take)

- **Electron windows mapped under running picom-xrender render black** on this Xvfb — start picom *after* the apps.
- **Obsidian is single-instance per config dir**: "relaunching" while the old process lives just refocuses it; kill by `mount_obsidi` (AppImage mount path), not the `/usr/local/bin/obsidian` launcher path.
- **Plugin sync-state lives in Obsidian's localStorage**, not `data.json`; wiping the whole `Local Storage` leveldb also wipes vault trust (trust dialog returns). Clear just the `sync-state`/`published-urls` keys via CDP and cycle the plugin.
- **Pushing content identical to the (hidden) server version is a no-op** — the per-take reset must scramble the server-side note content, or the retake never publishes.
- **A session-restored Chromium page carries a dead SSE subscription** — always force a fresh load (CDP `Page.navigate`), or live_follow silently never fires.
- `xdotool search` returns stale/hidden window ids — intersect with dwm's client list (`dwm_get_state`) before focusing; verify focus via `getactivewindow` before typing a single key.
- ImageMagick's `display -window root` sets a root pixmap picom can't bind (black band) — use `feh --bg-fill`.

## Known blemishes (accepted)

- The site renders in its default light theme next to dark Obsidian — reads as "editor vs published site".
- Obsidian shows its own window chrome/tabs — authentic.
- The click-ripple/smoothmove guidance is moot for this take: it is keyboard-only, the cursor sits parked on the wallpaper (parked via `smoothmove.py` before the recording starts).
