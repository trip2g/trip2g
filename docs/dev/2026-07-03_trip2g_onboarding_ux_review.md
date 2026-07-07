# First-run onboarding UX review — prioritized backlog

**TL;DR.** A new user can get from zero to a published page, but the path fights them at four points: the README quickstart is a dead end (`docker compose up` starts no trip2g), the README's own "Getting started" link 404s on the live site, the docs describe a manual BRAT+API-key setup while the product ships a one-click preconfigured vault the docs never mention, and every sync failure after setup is silent (console-only). Fixing those four removes most first-user pain.

**Method.** Walked the real path on an isolated memcli instance (`--name uxreview`, port 24981): boot → anonymous empty state → HAT login → vault download → plugin data.json → push a note → anon/admin visibility. Mined the demo-recorder reproducers and gotchas (`docs/dev/2026-07-03_hero_demo_v1_scenario.md`, `_v1_optB_`, `_v2_`; `autoproducer/scripts/hero*`). Cross-checked every claim against code (file:line below) and the live `trip2g.com`.

## Biggest wins (do these first)

1. **Fix the README quickstart dead end.** `README.md` says `git clone … && docker compose up` under "Run your own hub" — but the root `docker-compose.yaml` starts only MinIO + a 2.3 GB embedding server + a data mock, **no trip2g server**. The first thing a motivated dev tries, fails. Ship a compose file that boots the hub (the `docs/en/user/selfhosted.md` compose works), or point the README at it.
2. **Fix broken README links.** `https://trip2g.com/en/user/getting-started` → **404** (live slug is `/en/user/getting_started`); `…/two-way-sync` → 404 (live: `two_way_sync`). The single most-clicked onboarding link is dead. Verified 2026-07-03 with curl against all README `trip2g.com` links; all others 200.
3. **Document the starter-vault path in Getting started.** The product's real first-run flow (empty instance → "Download archive" → open in Obsidian → plugin already configured → save → live) is not mentioned anywhere in `docs/en/user/getting-started.md`, which instead walks users through BRAT + manual API key + manual plugin config — 4 steps the vault makes unnecessary. Only `selfhosted.md:560` hints at it. Lead Getting started with the vault path; keep BRAT as the "existing vault" path.
4. **Make background sync failures visible.** After setup, `autoSyncOnSave` push errors go to `console.warn` only (`obsidian-sync/src/main.ts:294`), live-pull errors too (`main.ts:715`), background change-polling swallows errors silently (`main.ts:601`). A user whose API key expired or server went down keeps "saving" into the void with zero feedback. Surface a Notice + red ribbon state on repeated auto-push failure.
5. **One dev sign-in code, everywhere.** The code is `111111` only (`cmd/server/auth.go:199,250`), but `docs/en/user/local-quickstart.md:66,141` claims "`000000` also works" (it does not), and the demo scenario had to leave a "*verify the dev code*" note (`2026-07-03_hero_demo_v2_scenario.md` beat 8–14). Fix the doc; even the team can't tell which code is real.

---

## CLI

### memcli

| # | Finding | Why confusing | Sev | Fix |
|---|---|---|---|---|
| C1 | Every `up` prints `NOTE: image must include feat/filestorage local-storage backend.` (`cli/memcli/src/cli.ts:1857`, also in `--help` NOTES) | Internal branch jargon on the happy path; a new user can't act on it and wonders if something is wrong | MED | Drop the note (the default image includes it), or turn it into a post-failure diagnostic |
| C2 | Happy path otherwise excellent: `up` boots, mints a key, writes plugin config, prints `memory live — web: http://localhost:24981` (verified live). Port-busy error is actionable (`cli.ts:1847`) | — | — | Keep; this is the best onboarding surface in the product — consider promoting `memcli up` + `memcli open` as *the* local try-it path in README |
| C3 | `memcli` exists only as `node cli/memcli/dist/memcli.js` inside a repo checkout; docs (`docs/en/user/memcli.md:15`) reflect that | "Install" = clone a whole monorepo; long invocation deters copy-paste | LOW | Publish to npm (`npx trip2g-memcli up`) or a curl-install like kanban's |

### trip2g-sync CLI (`obsidian-sync/dist/trip2g-sync.mjs`)

| # | Finding | Why confusing | Sev | Fix |
|---|---|---|---|---|
| C4 | `docs/en/user/cli.md` has drifted from the binary: pins release **0.3.5** (`cli.md:26,139`) while repo manifest is 0.6.1 and GitHub latest release is 0.5.0; documents `-f` short flag and default endpoint `http://localhost:8081/graphql` (`cli.md:66-67`) while the real help says positional `<folder>`, `-u` default `…/_system/graphql`, and `/graphql` is deprecated (`docs/dev/search_next_steps.md:25`); omits `--watch`/`--include`/`--exclude`/`--state-file` entirely | Users following the doc download an old binary and use a deprecated endpoint | HIGH | Regenerate `cli.md` from the current `--help`; link "latest" release instead of a pinned tag |
| C5 | Help usage line says `npx ts-node src/sync/cli/cmd.ts [options] <folder>` — a dev-repo invocation, and mixed positional/flag styles (error says `--folder is required` while examples use positional; both actually work) | First contact with the tool is its help text; it currently describes running from source | LOW | Usage line `trip2g-sync [options] <folder>`; make the required-arg error mention both forms |
| C6 | Fatal errors dump raw stack traces: `❌ Fatal error: Error: ENOENT … at Module.readdirSync (node:fs:1570:26) …` (verified: nonexistent folder). Exit codes are correct (1) | Stack trace reads as a crash, not "folder not found" | MED | Catch known errors (missing folder, 401 bad key, ECONNREFUSED) → one-line actionable message; keep stack behind `--verbose` |

## Docs

| # | Finding | Why confusing | Sev | Fix |
|---|---|---|---|---|
| D1 | README quickstart `docker compose up` starts no server (root `docker-compose.yaml`: minio + embedding + data_mock only) | Advertised 30-second start fails; user has no fallback except digging into docs | **HIGH** | See Biggest win 1 |
| D2 | README links 404: `en/user/getting-started`, `en/user/two-way-sync` (live slugs use underscores) | Dead getting-started link right after a failed quickstart = bounce | **HIGH** | Fix links; add a link-check to CI (a landing-link audit doc already exists: `docs/dev/2026-07-02_landing_link_audit.md`) |
| D3 | `getting-started.md` never mentions the onboarding-vault download; teaches the long path (BRAT → API key → paste URL+key → Test All Connections) | New users do 6 manual steps the product automates; docs and product tell different stories | **HIGH** | Restructure: "Path A — starter vault (recommended, 2 steps)" / "Path B — into your existing vault (BRAT)" |
| D4 | `getting-started.md:45-53` instructs enabling **Show Draft Versions** as a first step, but its default is already `true` (`internal/configregistry/registry.go:111-114`) | First admin-panel task the doc gives is a no-op; erodes trust in the doc | MED | Remove the step, or reframe: "drafts are visible by default; turn OFF to use releases" |
| D5 | `local-quickstart.md:66,141` claims dev code "`000000` also works" — false (`cmd/server/auth.go:250` accepts only `111111`) | User pastes 000000, gets rejected, doubts the whole setup | HIGH | Correct to 111111 only |
| D6 | `getting-started.md:120`: "Open the site in an incognito window and the page is not accessible." Reality: a non-free note returns **200** with the title rendered and a JS sign-in wall; body hidden (verified on isolated instance) | User checking "is it private?" sees the page half-exists and can't tell if privacy works | LOW | Reword: "visitors see the title and a sign-in wall, not the content" |
| D7 | `hosting.md:26` references image `ghcr.io/trip2g/trip2g:0.2` while everything else uses `:latest` | Version pin drift | LOW | Use `:latest` or a documented release policy |
| D8 | Sign-in with no/broken SMTP: `requestEmailSignInCode` enqueues and returns success (`internal/case/requestemailsignin/resolve.go:114-119`); with no SMTP the code is only **logged** (`email.go:14-17`, `sendsignincode/resolve.go:24`). `selfhosted.md:171` documents the journalctl fallback for bare-metal, but the Compose section and getting-started don't | The #1 self-host failure mode ("code never arrives") has its answer buried in one variant of one doc | MED | Put "code didn't arrive? check `docker compose logs trip2g` — the code is printed there" into getting-started's Sign in section and the Compose section |

## Plugin / sync setup

| # | Finding | Why confusing | Sev | Fix |
|---|---|---|---|---|
| P1 | Auto-push/auto-pull/background errors are console-only (`obsidian-sync/src/main.ts:294,601,715`); red ribbon badge fires only on content conflicts, not connection failures | Silent data-not-syncing; the worst failure UX in the funnel | **HIGH** | Notice + persistent ribbon error state after N consecutive failures; "last successful push HH:MM" in settings |
| P2 | Settings empty state (`main.ts:1401-1407`) links `https://trip2g.com/docs/onboarding` — which serves a **Russian** page ("Начало работы") to every user | English user's first in-plugin doc link lands on RU content | HIGH | Link `…/en/user/getting_started` (or lang-negotiated) |
| P3 | `autoSyncOnSave` defaults to **false** (`src/types.ts:29-34`) and getting-started never mentions it; the "magic" moment in all demos (save → live) is off by default and undocumented | Users manually click sync forever, missing the product's headline behavior | MED | Mention the toggle in getting-started; consider default-on for vault-download path (ship it in the vault's `data.json` — currently `skipPushConfirmation: false`, no autoSync, verified in downloaded zip) |
| P4 | Sync state lives in Obsidian localStorage (`main.ts:31-32` `sync-state`/`published-urls`), not in the vault; copy vault to a new machine without `.obsidian` internals → state lost → first sync triggers migration modal (`main.ts:945-949`). Reset exists only as a settings button (`main.ts:1478-1485`) | Invisible state with surprising cross-machine behavior; demo rig needed CDP surgery to clear it (`hero_demo_v1_scenario.md` gotchas) | MED | Document the behavior + the Reset button; consider persisting state in the plugin data dir instead |
| P5 | No-op save gives no feedback on auto-sync (Notice only when `pushed > 0`, `main.ts:339`); manual sync says "All files are up to date" (`main.ts:1010-1012`) | After editing-and-reverting, user saves and sees nothing — is sync broken? (also cost the demo recorder retakes) | LOW | Optional subtle status-bar tick "up to date" on auto-sync no-op |
| P6 | Plugin not in the official Obsidian catalog; BRAT-only (`obsidian-sync/README.md:18`) and BRAT "Latest" fetches GitHub release **0.5.0** while repo is at 0.6.1 | Extra installer plugin, plus users get a stale build | MED | Cut a 0.6.x release; pursue the official catalog listing |
| P7 | Failure text on Test All Connections is good (per-dir red error, `main.ts:1501-1508`; summary "X successful, Y failed", `main.ts:1446`) — but bad-URL errors are raw fetch messages | Raw "Failed to fetch" doesn't say "check the URL / your key" | LOW | Map common errors (401 → "API key rejected", ENOTFOUND → "check the site URL") |

## Admin UI / first-run

| # | Finding | Why confusing | Sev | Fix |
|---|---|---|---|---|
| A1 | Anonymous empty state says "Log in as administrator to get started" with **no login link** in that block (`internal/case/rendernotepage/view.html:288-295`); login lives in the JS-mounted `$trip2g_user_space` widget top-right | Fresh owner opening their new instance must hunt for how to log in at the exact moment of first contact | HIGH | Make "Log in" in that sentence a link/button that opens the auth widget |
| A2 | Onboarding page copy is hardcoded RU-first bilingual (both languages stacked, `view.html:253-295`; sidebar item hardcoded "Главная", `view.html:236`), bypassing the i18n system (`langs/en.toml` exists) | First screen looks half-translated/unfinished | MED | Localize via `ctx.T()` like the rest of the template |
| A3 | Onboarding page ends at "Download archive" — nothing about the next steps (open in Obsidian → **Trust author and enable plugins** → press sync). The trust dialog is a known stumble (demo rig one-time setup, `hero_demo_v1_optB_scenario.md:60`) | User downloads a zip, opens it, and hits an Obsidian security dialog no one warned them about; if they decline, the plugin never runs and nothing syncs | HIGH | Add 3 numbered steps under the button, explicitly including the trust dialog and the sync button |
| A4 | Dev-mode login pre-fills the code field with `111111` (`assets/ui/auth/auth.view.ts:245`) with no hint text; dev mode also self-activates on any `localhost` hostname (`assets/ui/settings/settings.ts:3-5`) | Pre-filled value with no label reads as a bug; and on a non-DEV localhost server the pre-filled code is wrong | MED | Show "(dev instance — code is 111111)" caption; gate the prefill on server-reported `is_dev_mode`, not hostname |
| A5 | Each visit to `/_system/onboarding-vault` mints a new API key "Onboarding vault" (`downloadonboardingvault/resolve.go:107-124`) | Curious admin clicks 3× → 3 live admin keys, no cleanup, no warning | LOW | Reuse/revoke prior onboarding keys, or name them with a timestamp |
| A6 | `?enable_admin_graphql` download option exists only as an undocumented query param (`downloadonboardingvault/endpoint.go:35-37`); UI toggle is a button buried in Integrations → API Keys → key (`mcpadmintools.view.ts:22`) | Agent-oriented users can't discover the capability | LOW | Checkbox on the onboarding page ("include admin access for agents") |
| A7 | API Keys live under **Integrations**, site config under **System** with raw `config_id` rows (`show_draft_versions`, no friendly labels) (`assets/ui/admin/-/node.view.tree:4469`; `admin/menu/system/system.view.tree:9`) | Getting-started sends users to "Settings → API Keys"-style paths that don't match the nav; config table is developer-facing | MED | Friendly config labels + make doc paths match nav ("Integrations → API Keys") |
| A8 | Turning `show_draft_versions` **off** makes every future sync invisible until a release, with no UI warning (`rendernotepage/resolve.go:208-220`) | "I synced and my page vanished" support trap | LOW | Confirmation dialog explaining the release workflow when toggling off |
| A9 | HAT login is clean server-side: token consumed, `?hat=` stripped via 302 (`cmd/server/server.go:71-83`, verified: `302 Location: /` + cookie). But the original `?hat=` URL remains in browser history/omnibox suggestions (demo had to clear history, `hero_demo_v1_optB_scenario.md:60`) | Short-TTL mitigates it, but a signed login token in shareable history is a footgun | LOW | Note in `memcli open` output that the link is one-time/short-TTL; consider POST-based handoff page |

## Errors (cross-cutting)

| # | Finding | Sev | Fix |
|---|---|---|---|
| E1 | Email send failures are invisible end-to-end: GraphQL returns success before the job runs (`requestemailsignin/resolve.go:114-119`), SMTP-missing is a log-warn no-op (`email.go:14-17`) | MED | Admin-visible mail health (banner in admin panel: "email is not configured — sign-in codes appear in server logs") |
| E2 | Broken custom layout falls back silently to the default layout with HTTP 200 (`local-quickstart.md:143` documents it as a gotcha) | LOW | Admin-only inline warning banner on fallback-rendered pages |
| E3 | Missing S3 object for a header logo generates a presigned URL without existence check → silent broken `<img>` (`internal/miniostorage/storage.go:182-207`) | LOW | Skip the logo when the object is missing; log once |

## Automation-only quirks (not user-facing — don't spend product effort)

From the demo-recorder gotchas (`2026-07-03_hero_demo_v1_scenario.md:58-66`, `_optB_:60-67`): cold-start Obsidian dropping first keystrokes after `Ctrl+N`; stale `xdotool` window ids vs dwm client list; picom/Electron black-window ordering; `feh` vs ImageMagick root pixmap; session-restored Chromium carrying a dead SSE subscription (real-ish, but only bites parked long-lived tabs; live_follow is an owner/demo feature). The per-take server-side content scramble exists only because of the (real, listed above as P5) no-op push behavior; the localStorage CDP surgery exists because of (real, P4) sync-state placement.

## Verification notes

- Isolated instance `trip2g-memory-uxreview` (port 24981, `trip2g:local`) was booted and torn down; the recording rig on `:99` and its `demo-v1`/`demoV2` instances were not touched.
- Live-site link checks ran against `trip2g.com` on 2026-07-03.
- Not tested for real: the BRAT install flow inside a GUI Obsidian and the admin panel's visual navigation (analyzed from `assets/ui` source + agent reports instead); the resend email path (analyzed from code).
