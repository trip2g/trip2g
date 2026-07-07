# Hero demo: scenario, narration, and shot-list

**TL;DR.** The README hero shows the two-audience loop in ~30 seconds of real screen capture: type a line into the Obsidian vault → `trip2g-sync` pushes it → the browser refresh shows it live on the site → an MCP `search` in a terminal returns the same fresh section to an agent. Narrated MP4 (`docs/demo-video/hero-narrated.mp4`, 30.5s, 1.35 MB) for the landing page/social; silent loop GIF (`docs/demo-video/hero.gif`, 20s, 1000px, 0.73 MB) for the README. Reproducer: `autoproducer/scripts/hero/run-all.sh`. Every pixel is the real product — no mockups (launch kit §4.2 honesty rule).

## The one-loop story

One markdown vault, served two ways. A human edits in Obsidian; the page is live on the website in about a second; an AI agent reads the same note over MCP. Grounded in `2026-07-02_launch_kit.md` §4.2 storyboard.

## Screen setup (all real, on Xvfb `:99`, 1920×1080, dwm tiling)

| Window | What it is | Honesty note |
|---|---|---|
| Left (master, 60%) | **Obsidian 1.12.7** (the real AppImage, second isolated instance, `XDG_CONFIG_HOME` scratch) with the real `docs/` vault open, note `demo/hello-agents.md` | Restricted Mode, so the vault's sync plugin doesn't double-run next to the user's live instance; sync is done by the product's own `obsidian-sync` CLI |
| Top right | **Chromium** at `http://localhost:8081/demo/hello_agents` — the local `make air` trip2g instance rendering the synced vault | real server, real page, real refresh |
| Bottom right | **xterm** with two thin wrappers on PATH: `trip2g-sync` (runs the real `obsidian-sync/trip2g-sync.mjs` push, output shown verbatim, debug line filtered) and `trip2g-mcp` (curl JSON-RPC to the real `/_system/mcp` endpoint, prints the result text verbatim) | wrappers only shorten the command line; nothing on screen is synthesized |

## Narration (ElevenLabs, `eleven_multilingual_v2`, voice from instagen `.env`; 27.66s)

> One Obsidian vault, published two ways. Watch: I type a line into a note, save, and the sync CLI pushes it to my self-hosted trip2g server. One second later, it is live on the website. Same text, real page. And AI agents read the same source over MCP: search returns the exact section, a few hundred tokens, not the whole vault. One vault. A website for humans, context for agents. trip2g.

## Beat sheet (times = seconds on the VO clock; the audio is the clock)

| t (s) | VO says | On screen (exact commands) |
|---|---|---|
| 0.0–3.0 | "One Obsidian vault, published two ways. Watch:" | static tiled scene, Obsidian focused, caret at end of note |
| 3.0–5.8 | "I type a line into a note" | `xdotool type` (38 ms/char): newline, `## Fresh from the vault`, newline, `This line reached the web in about a second.` |
| 5.9 | "save" (word at 4.50; action lands ≤1.5s after) | `xdotool key ctrl+s` |
| 6.3–9.5 | "the sync CLI pushes it to my self-hosted trip2g server" ("pushes" at 6.72) | focus terminal; type `trip2g-sync` ⏎ at 7.2 → real push output: `Executing sync… / ✅ Pushed 1 notes` |
| 9.8–13.9 | "One second later, it is live on the website. Same text, real page." (at 9.96) | focus browser; `F5` at 10.3; `Down`×3 at 11.4 reveals **Fresh from the vault** on the live page |
| 14.0–21.5 | "AI agents read the same source over MCP: search returns the exact section…" ("search returns" at 17.91) | focus terminal; type `trip2g-mcp search 'fresh from the vault'` ⏎ at 15.9 → real MCP result: title, `demo/hello-agents.md`, URL, snippet containing the fresh line, `match_id` |
| 22.4–27.7 | "One vault. A website for humans, context for agents. trip2g." | focus browser on the fresh section; hold |
| 27.7–29.3 | (silence) | tail hold, then cut |

Word timings come from the ElevenLabs `with-timestamps` alignment (`autoproducer/scripts/hero/vo-align.json`); the driver (`record.mjs`) schedules each beat against wall-clock t0 written to `t0.txt`, and the montage computes the exact audio offset from the recording's wallclock start (`ffprobe start_time`), so VO-to-action sync is deterministic.

## Cuts

- **Narrated MP4** (primary, landing/social): full take + VO, 1920×1080, 30.5s, 1.35 MB. `docs/demo-video/hero-narrated.mp4`
- **README loop GIF** (silent): VO-seconds ≈2.3→22.7 — typing → sync → live refresh → MCP result — 20s, 15 fps, 1000px, 0.73 MB (≤5 MB budget). `docs/demo-video/hero.gif`
- Raw capture: `docs/demo-video/hero-raw.mkv`; VO track: `docs/demo-video/hero-vo.mp3`.

## Reproduce

```bash
# once: local instance on :8081 (make air), dev minio up, VO generated (vo.mjs)
TRIP2G_API_KEY=<key> /home/alexes/projects2/autoproducer/scripts/hero/run-all.sh /tmp/hero-out
```

`run-all.sh` = stage up (Xvfb+dwm via autoproducer) → reset note + push → launch/arrange the three real apps → timed take (`record.mjs`) → mux VO → cut GIF. First Obsidian launch shows the vault-trust dialog once: pick "Browse vault in Restricted Mode".

## Known blemishes (accepted)

- dwm top bar (tags + `dwm-6.2`) is visible — honest dev-rig framing, not product chrome.
- The demo note `docs/demo/hello-agents.md` stays in the vault (base content, `free: true`); the take appends the "Fresh from the vault" section and the reproducer resets it.
- Obsidian shows the Properties block (frontmatter) — authentic, kept.
