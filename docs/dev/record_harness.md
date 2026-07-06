# Record harness — local recording target for the demo-video rig

**TL;DR:** `node scripts/record-harness/seed.mjs` brings up
`docker-compose.record.yml` and seeds it — a deterministic local clone of the
demo scenario's two public targets (trip2g.com landing + a cloud space).
Point the rig at it with `V4_CONFIG=v4-config.record.json` (autoproducer
`scripts/hero/v4/`). One canonical harness per video: stand it up first, then
record — no more takes lost to prod drift (the header CTA that vanished, the
stale cloud image).

## What it stands up

| Service | URL | Contents |
|---------|-----|----------|
| landing | http://localhost:21080 | repo `docs/` vault (the same content trip2g.com CI re-seeds from, same excludes as `scripts/sync-docs-prod.sh`) + seed-time overrides below |
| dashboard stand-in | http://localhost:21080/simplecloud | a note mimicking the simplecloud space card, with the **Open as Admin →** link to the space |
| space | http://localhost:21090 | EMPTY — the onboarding "Welcome! / Download archive" state the demo opens on |
| minio | localhost:29200 (console :29201) | asset store for both instances (tmpfs, clean each recreate) |

Build pin: images are built from the repo checkout (`Dockerfile`), so the
harness always runs exactly the branch you're on — current main includes the
render fix (fb2a9efe) that the stale prod image was missing (magazine layout /
`Header()` / `Styles()` helpers work).

## Seeded credentials (local-only, not secrets)

- Sign-in on both instances: `hello@example.com` + dev code `111111` (`DEV=true`).
- Space api key (deterministic, same value every seed):
  `recordharness0recordharness0recordharness0recordharness0record0`
  — inserted straight into the DB by seed.mjs; the rig profile
  (`v4-config.record.json`) carries the same value for preflight/stage/gate.
- Landing seeder key: created fresh per seed run (only seed.mjs uses it).

## Seed-time overrides (where the reference landing differs from public)

The overrides are applied by `seed.mjs` as pushes on top of the docs vault —
the repo `docs/` files stay untouched, nothing is published:

1. **Header CTA `try free →` → `/en/user/cloud`** — inserted into
   `_layouts/mesh/bar.html` (EN header, next to `start →`). This is the exact
   element segLand clicks. It existed on prod (deployed ad-hoc 2026-07-03) and
   was clobbered by the CI re-seed because it was never in the repo. If you
   want it back on the public landing, land it in `docs/_layouts/mesh/bar.html`
   properly.
2. **`en/user/cloud.md`** — `https://simplecloud.2pub.me` links rewritten to
   `http://localhost:21080/simplecloud` (path contains `simplecloud` on
   purpose: record.mjs asserts `href.includes('simplecloud')`).
3. **`simplecloud/_index.md`** — dashboard stand-in
   (`scripts/record-harness/overrides/`). The real control plane is a separate
   repo; this page only reproduces the space card + "Open as Admin" link the
   scenario needs. The video's opening beat reads the same; the login form and
   multi-space list of the real dashboard are not reproduced.

Intentional differences from the public targets, in one list: the try-free CTA
(missing on prod today), the dashboard (stand-in note, no auth handoff), and
the RU header (no try-free link added there).

## Usage

```bash
# stand up + seed + self-verify (idempotent, re-run any time)
node scripts/record-harness/seed.mjs

# seed only, stack already running
SKIP_UP=1 node scripts/record-harness/seed.mjs

# tear down (fresh DBs next time: also rm -rf tmp/record-data)
docker compose -f docker-compose.record.yml down
```

Point the rig at it (in `autoproducer/scripts/hero/v4/`):

```bash
V4_CONFIG=v4-config.record.json ./preflight.sh   # health gate vs the local target
V4_CONFIG=v4-config.record.json node record.mjs ...  # record.mjs honors it too
```

One-time, in the recording browser profile: sign in as admin on
http://localhost:21090 (dev code `111111`) so the space Welcome page shows the
"Download archive" button — this mirrors the operational simplecloud login
that was never on camera.

## Known gaps

- `reset-stage.sh`, `stage-notes.sh` and `run-all.sh` still hardcode
  `https://$SPACE` and read `v4-config.json` directly — a full local
  `gate.sh`/`run-all.sh` wave needs the same `V4_CONFIG` + scheme treatment
  (preflight.sh and record.mjs already have it). Also `record.mjs` matches CDP
  pages by URL substring (`SITE`, `'trip2g.com'`); with both local targets on
  `localhost` those match strings need a look before a full local wave.
- A local golden-vault (make-golden.sh from the LOCAL onboarding zip) is needed
  before actually recording Obsidian beats against this harness — the current
  golden-vault carries the cloud space's apiUrl/key.
- "Open as Admin" is a plain link (no HAT auth handoff): the admin session on
  the space comes from the one-time browser sign-in above.
- Product finding (upstream, not harness-specific): the onboarding zip's
  directory entries carry no trailing slash, so plain `unzip` extraction fails
  with "exists but is not directory". `unzip -p <zip> <entry>` and `bsdtar -xf`
  work. Affects `make-golden.sh` (which unzips) — and possibly real users.
