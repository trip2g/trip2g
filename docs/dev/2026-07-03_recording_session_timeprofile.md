# Recording session time-profile — v2 onboarding demo (2026-07-03)

**TL;DR:** Of a 77-minute session (06:57:55–08:14:56, snapshot — agent was still mid-compositing at cut-off), only ~11 min (15%) was actual recording. ~28 min (36%) went to interactive rehearsal (blind-click → screenshot → look → correct), ~15 min to state resets between rehearsal and takes, ~9 min to codebase archaeology (dev sign-in code, DevMode, plugin internals). One retake (seg3a, Obsidian list-autocomplete ate the typed content). The pipeline's cost center is not ffmpeg or ElevenLabs — it's **unscripted UI state discovery and manual state reset**.

Transcript analyzed: `/home/alexes/.claude/projects/-home-alexes-projects2-trip2g/2f470def-1431-4ce5-a854-c2c9978e405d/subagents/agent-aec19c560d05f652e.jsonl` (735 events, 251 tool calls). The session is a **snapshot**: it ends at `Write post.py` — compositing, encode, and GIF had not run yet, so their cost is not in this profile.

## 1. Time profile

| Phase | Window | Wall clock | % | What happened |
|---|---|---:|---:|---|
| Dress rehearsal round 2 (full S1→live-flip proof) | 07:28–07:40 | 12.6 min | 16% | Restricted-Mode trust miss, "files deleted on server" dialog, live-flip finally proven |
| Between-segment vault prep (prep1–prep10) | 07:48–07:57 | 9.6 min | 12% | Rebuild vault from on-camera zip, plugin re-setup, click-by-click with screenshot checks |
| Rehearsal round 1 (Obsidian launch, login, coords) | 07:07–07:15 | 8.3 min | 11% | shot1–shot17; mis-clicks (search icon vs sign-in), pre-filled field, stray Firefox |
| Sync-on-save debugging | 07:15–07:22 | 7.1 min | 9% | BRAT installed plugin 0.5.0 without `autoSyncOnSave`; local 0.6.1 installed + Obsidian restart |
| seg3a bug-fix + retake prep | 08:00–08:06 | 6.3 min | 8% | Obsidian auto-list + wikilink popup ate Returns → vault3 content garbage; cleanup + record.mjs edits |
| Zero-state reset + docker DEV flap | 07:22–07:28 | 5.3 min | 7% | root-owned sqlite (`rm: Permission denied`), container `Exited (137)` after DEV=true recreation |
| Takes seg3a(retake)+seg3a2+seg3b + verify | 08:06–08:11 | 5.1 min | 7% | frame-extraction verify after each |
| Recon: spec + optB reuse + codebase research | 06:58–07:02 | 4.2 min | 5% | Found dev code is `111111` not `000000`; DevMode config hunt across ~15 greps |
| Endcard + record.mjs authoring | 07:40–07:44 | 3.9 min | 5% | 104s single Write of record.mjs |
| Takes seg1a + seg1b + verify | 07:44–07:47 | 3.5 min | 5% | Clean first takes |
| post.py compositing script write (cut-off) | 08:11–08:14 | 3.4 min | 4% | 205s single Write; session snapshot ends here |
| seg2 take + frame checks | 07:57–08:00 | 2.7 min | 4% | Clean take; 8 frame extracts to verify |
| memcli + trip2g boot, sign-in probing | 07:02–07:04 | 2.4 min | 3% | Boot fine; sign-in code hunted in prior-session tool-result files |
| Vault/Obsidian config prep | 07:05–07:07 | 1.4 min | 2% | |
| VO generation (ElevenLabs) + whisper check | 07:04–07:05 | 1.2 min | 2% | 2 API iterations, v3 accepted; **not** a time sink |
| **Total** | | **77 min** | 100% | |

**Grouped:** rehearsal/verification 28 min (36%) · state reset/re-prep 15 min (19%) · recording+verify 11.3 min (15%) · script authoring 7.3 min (9%) · recon/boot/VO 9.2 min (12%) · bug-fix/retake 6.3 min (8%).

**Longest single operations** (gap to next tool call): 215s stall after a prep click (model latency digesting a screenshot), 205s `Write post.py`, 104s `Write record.mjs`, 103s docker DEV-wrapper flap (`Exited (137)`), 76s seg2 record+frame-extract, 67s Obsidian/Chromium teardown, 66s vault3 content restore.

The see-click loop itself is the diffuse killer: **66 screenshot Reads** and **62 xdotool commands**; gaps preceding screenshot Reads alone sum to ~16.5 min, and each cycle (click → import → Read PNG → think) runs 10–25s.

## 2. Retakes / retries / loops

- **1 true retake:** seg3a — Obsidian's list auto-continuation + wikilink autocomplete popup swallowed Return keys, corrupting typed note content; content cleaned, record.mjs patched, re-rolled (~6 min lost). seg3a2 was a planned extra segment, not a retake.
- **4 Obsidian restarts:** initial launch, plugin 0.5.0→0.6.1 upgrade, zero-state reset, mid-rehearsal pkill for vault2.
- **3 docker container cycles:** initial boot, DEV=true recreation, recovery from `Exited (137)`.
- **2 stray-window hunts:** unexpected Chromium window (07:12) and a Firefox spawned by an off-script `xdg-open` (07:13) — PID forensics + targeted kill, ~2.5 min.
- **~6 coordinate mis-clicks** corrected via screenshot loop (search toggle instead of sign-in icon, Restricted Mode instead of "Trust author", pre-filled email field needing ctrl+a, etc.).
- **2 full environment resets** (rehearsal → zero state → dress rehearsal → zero state → takes). Each reset is click-by-click, ~5–10 min.

Total loop/rework loss: **~15–18 min (~20% of session)**.

## 3. Top difficulties (ranked by time cost)

1. **Unscripted UI state discovery (~28 min).** Every button was found by blind-click + `import -window root` + Read PNG. Representative: `07:08:40 "I hit the search toggle instead. The sign-in icon is the leftmost one."` Coordinates from the v1 run were not reusable/recorded, so both rehearsal rounds re-derived them.
2. **Environment state not resettable by script (~15 min).** Zero-state restore fought root-owned docker volumes (`rm: cannot remove '.../local.sqlite3': Permission denied` → alpine-container rm workaround), a DEV=true container recreation that OOM/exited `(137)`, and manual vault re-download/re-setup between segments.
3. **Plugin/app behavior surprises (~13 min).** (a) BRAT delivered plugin 0.5.0 without `autoSyncOnSave` — the demo's core feature — vs local build 0.6.1; (b) "Trust author" click actually landed on Restricted Mode, silently disabling the plugin ("No push attempt at all — plugin may not have loaded"); (c) first sync of a downloaded vault raises a "Files deleted on server → Keep locally" dialog nobody planned for.
4. **Obsidian typing semantics (~6 min + the only retake).** Auto-continued lists and the `[[` autocomplete popup eat Return keystrokes typed via xdotool.
5. **Codebase archaeology mid-recording (~5 min).** Dev sign-in code (`111111` vs assumed `000000`), DevMode env plumbing, title-extraction rules — all hunted with grep during the session instead of being in the scenario/harness docs.

## 4. Speed-up levers

1. **Persist a coordinate/state map per stage layout.** Both rehearsal rounds (20+ min) rediscovered the same buttons at 2560×1440. Save a `coords.json` (sign-in icon, trust button, community-plugins toggle, ribbon sync, hamburger→live-reload, download button) next to record.mjs and validate it with one screenshot diff at boot instead of 66 look-loops. This attacks the single biggest cost.
2. **Script the reset, not just the take.** Make `reset-stage.sh` do the full zero-state: `docker run alpine rm` for root-owned data (already proven in-session), container recreate **with DEV=true baked into the memcli wrapper from the start**, vault wipe/restore, Obsidian relaunch with pre-trusted vault + pre-enabled plugin (`community-plugins.json` + `data.json` written before launch — skips the Trust/Restricted-Mode dialog class entirely, which cost ~5 min and one silent failure).
3. **Pre-bake the plugin and dialogs.** Install the local 0.6.1 build into the vault template (never BRAT), and pre-answer the first-sync "files deleted on server" dialog off-camera in the harness (the session did this manually at 07:36 — encode it).
4. **Kill the typing-retake class:** paste note content via `xdotool key ctrl+v` from clipboard (or write the .md file on disk and let Obsidian pick it up, which the session already used at 07:40 for `_index.md`) instead of `type`-ing markdown that triggers list-continuation/wikilink popups. This removes the only retake.
5. **Split content from capture.** The 205s `post.py` and 104s `record.mjs` Writes plus VO are text-only artifacts — keep them as committed, parameterized scripts in `scripts/hero/v2/` so a VO/endcard/timing tweak never re-triggers rehearsal or takes. VO is already cheap (~1 min) and cached per-beat; keep per-beat mp3s so one line change re-renders one beat.
6. **Guard against off-script windows:** the stray Firefox came from an `xdg-open` on :98. Set `BROWSER=/bin/true`/neutralize xdg-open in the stage env, and have record.mjs assert the exact window list before rolling — a black-frame/extra-window check is cheaper than PID forensics mid-session.

**Caveat:** compositing/encode/GIF cost is unknown — the snapshot ends before they ran. Profile that tail in the next session before optimizing it.
