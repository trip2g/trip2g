# v1 onboarding demo, Option B (post-compositing): scenario and shot-list

**TL;DR.** The v1 onboarding demo in ~27 seconds of real screen capture: a fresh memcli-served vault open in the real Obsidian → the user creates their *first page* (title + one line) → the real `trip2g sync` plugin auto-pushes on save ("Pushed 1 files to server" toast) → the page is live in a real Chromium on the local instance → CTA end card (repo + star). Option B = the macOS-window look (title bar with traffic lights, rounded corners, drop shadow, wallpaper) is added **in post** over untouched window crops; the sibling Option A take does it live. Artifacts: `docs/demo-video/v1-optB-post.mp4` (2560×1440, 29 s, 1.9 MB) + `v1-optB-post.gif` (13.6 s silent loop, 1080 px, 0.5 MB). Reproducer: `autoproducer/scripts/hero/optB/run-all-optB.sh`. Every product pixel is real — no mockups; the frame/background are declared chrome.

## The story (one save, and it's on your website)

A person gets a trip2g-backed vault, opens it in Obsidian, writes their first page, hits save — and the page is live on their own website a second later. No terminal on screen: the sync is the product's own Obsidian plugin (`autoSyncOnSave`), the browser shows the page on the real local instance.

## Screen setup (all real, Xvfb `:98`, 2560×1440, no WM)

| Window | What it is | Honesty note |
|---|---|---|
| Left (1260×1010 @ 40,60) | **Obsidian 1.12.7** (real AppImage, isolated `XDG_CONFIG_HOME`), vault served by memcli instance `demoB`; **trip2g sync plugin v0.6.1 enabled** (community plugins on, `autoSyncOnSave: true`) | the push on screen is the plugin's own debounced auto-sync (2.5 s after the last edit); the toast is the plugin's real Notice |
| Right (1120×1010 @ 1360,60) | **Chromium** on `http://localhost:24281` — the memcli-booted trip2g server (docker, `trip2g-memory-demoB`) rendering the vault | logged in once as the owner via a one-time `?hat=` URL; page load and URL typing are real |

No window manager: Option B captures the full screen once and crops each window in post, so windows just sit at fixed rects on a black root. The pointer is parked outside both windows — takes are keyboard-driven, so the crops never show a stray cursor (and there are no janky pointer teleports on camera).

## Narration (ElevenLabs `eleven_multilingual_v2`, voice `tvApEij8Hu2VscOrlzzV`, 24.71 s)

> A plain Obsidian vault on the left. On the right — my own website, served by self-hosted trip2g. I create my first page: a title, and one line of text. I hit save. The sync plugin pushes it to the server automatically. Now open the page — and it's live. Write in Obsidian, save, and it's on your website. Try it yourself — the repo is trip2g on GitHub. And if you like it, star it.

## Beat sheet (times = seconds on the VO clock; the audio is the clock)

| t (s) | VO says | On screen |
|---|---|---|
| 0.0–6.6 | "A plain Obsidian vault… my own website, served by self-hosted trip2g." | static two-window scene, Obsidian focused (empty New tab), browser on the vault Index |
| 6.5 | "I create my first page:" | `ctrl+n` (then a generous 1.4 s pause — a cold Obsidian drops the first keystrokes of the inline title otherwise) |
| 7.9–9.0 | "a title" | type `My first live page`, `Return` |
| 9.3–11.3 | "and one line of text" | type `Hello world! This page went from Obsidian to my own website with one save.` |
| 11.6 | "I hit save." | `ctrl+s` |
| ~13.8 | "…pushes it to the server automatically." | the plugin's debounce fires: **"Pushed 1 files to server"** toast |
| 13.9–15.5 | "Now open the page" | focus browser, `ctrl+l`, type `localhost:24281/my_first_live_page`, `Return` |
| ~15.9 | "and it's live." | the page renders: same title, same line, real server |
| 15.6–21 | "Write in Obsidian, save, and it's on your website." | hold on the two windows |
| 21.3→ | "Try it yourself — the repo is trip2g on GitHub…" | **end card** fades in (0.7 s): trip2g / github.com/trip2g/trip2g / "Try it yourself — ⭐ star it on GitHub" |
| 24.7–26.5 | (silence) | end card holds, cut |

Word timings come from the ElevenLabs `with-timestamps` alignment (`optB/vo-align.json`); `record.mjs` schedules beats against wall-clock `t0`, and `composite.sh` computes the exact audio offset from the recording's `ffprobe start_time` — same deterministic sync approach as the first hero cut.

## Option B post pipeline (`composite.sh`)

1. One 2560×1440 x11grab take (`raw.mkv`, crf 16) of the bare windows on the black root.
2. ImageMagick builds a static plate: fal.ai wallpaper (decorative only) + per-window soft shadow (blurred rounded rect) + white rounded frame with a grey 44 px title bar and red/yellow/green traffic lights (SrcAtop keeps the strip inside the rounding).
3. ffmpeg crops each window from the take, rounds its bottom corners (`alphamerge`), and overlays both crops onto the plate — product pixels untouched, only re-framed.
4. End card (`endcard.html` → headless-Chromium PNG, same wallpaper) fades in at VO 21.3 s.
5. VO muxed with the computed offset; master `libx264 crf 17 veryslow`, 2560×1440.
6. Silent README loop GIF: VO 5.4→19.0 s (create → save → push toast → live page), 14 fps, 1080 px, palette dither — 0.5 MB (≤5 MB budget).

## Cuts

- **Narrated MP4** (landing/social): `docs/demo-video/v1-optB-post.mp4` — 2560×1440, ~29 s, 1.9 MB, VO + CTA end card.
- **README loop GIF** (silent, no CTA): `docs/demo-video/v1-optB-post.gif` — 13.6 s core loop, 1080 px, 0.5 MB.

## Reproduce

```bash
/home/alexes/projects2/autoproducer/scripts/hero/optB/run-all-optB.sh /path/to/outdir
```

One-time interactions on a fresh rig: Obsidian's vault-trust dialog (**Trust author and enable plugins** — the plugin must run) and a history clear in Chromium after the `?hat=` login (keep cookies) so the one-time token never shows up in omnibox suggestions on camera.

## Known blemishes (accepted)

- Obsidian shows its Linux window controls (−□×) inside the frame next to the added macOS traffic lights — honest capture of the real app, kept.
- The plugin's red "sync warnings" ribbon badge is visible in Obsidian's status bar — real UI, kept.
- The browser is logged in as the instance owner; the page itself is also publicly reachable (curl 200 without cookies).
- A cold-started Obsidian (<1 min uptime) drops the first keystrokes after `ctrl+n` — cost takes 1 and 3; warm the app before recording.
