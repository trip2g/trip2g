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
| caddy TLS front | 127.0.0.1:443 | serves the REAL demo domains with `tls internal`: `trip2g.com` → landing, `simplecloud.2pub.me` → landing (root rewritten to `/simplecloud`), `honesthesse.2pub.me` → space, `assets.trip2g.com` → minio (upstream Host restored to the signed `minio:29200`) |

## Real domains in the recording browser (F1)

The video must show real https domains — never localhost. The caddy service
(`scripts/record-harness/Caddyfile`) terminates TLS with its local CA:

1. Root CA: `docker cp trip2g-record-caddy:/data/caddy/pki/authorities/local/root.crt .`
   then `certutil -A -d sql:$HOME/snap/chromium/current/.pki/nssdb -n trip2g-record-caddy-local -t "C,," -i root.crt`
   (no sudo; certutil extractable from libnss3-tools via `apt-get download` + `dpkg -x`).
2. Recording chromium runs with
   `--host-resolver-rules="MAP trip2g.com 127.0.0.1, MAP simplecloud.2pub.me 127.0.0.1, MAP honesthesse.2pub.me 127.0.0.1, MAP assets.trip2g.com 127.0.0.1"`
   (reset-stage.sh sets this). No `/etc/hosts` edit needed for browser-only scenes.
3. `PUBLIC_URL` is the real domain per instance; `MINIO_PUBLIC_URL=https://assets.trip2g.com`
   keeps presigned asset URLs browser-reachable and https (mixed-content upgrade
   would break plain-http images).

Obsidian scenes (2+) additionally need host resolution for the sync plugin
(`/etc/hosts` or equivalent) + `NODE_EXTRA_CA_CERTS=<root.crt>` — not wired yet.

Build pin: images are built from the repo checkout (`Dockerfile`), so the
harness always runs exactly the branch you're on — current main includes the
render fix (fb2a9efe) that the stale prod image was missing (magazine layout /
`Header()` / `Styles()` helpers work).

## Seeded credentials (local-only, not secrets)

- Sign-in on both instances: `maya@2pub.me` + dev code `111111` (`DEV=true`).
  Demo identity (F2 scrub): the email domain must resolve in DNS
  (`is.Email` checks it) and no dev email may render in frame.
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
2. **`en/user/cloud.md`** — the real `simplecloud.2pub.me` domain stays in the
   page (the caddy front serves it locally); only the "Open the cloud" CTA is
   pointed at `/_system/admin` so the on-camera click lands on the real
   sign-in form (record.mjs asserts `href.includes('simplecloud')`).
3. **`simplecloud/_index.md`** — dashboard stand-in
   (`scripts/record-harness/overrides/`). The real control plane is a separate
   repo; this page only reproduces the space card + "Open as Admin" link the
   scenario needs.
4. **dev-email scrub** — `alexes.dev@gmail.com` replaced with `hello@trip2g.com`
   in `_layouts/mesh/{foot,try_now,pricing}.html`, `_footer.md`,
   `ru/_footer.md`, `en/user/hosting.md`, `ru/user/hosting.md` (seed-time
   override pushes; the repo files stay untouched — the address is intentional
   on the public site).

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

- "Open as Admin" is a plain link (no HAT auth handoff): the admin session on
  the space comes from the one-time browser sign-in above. Scriptable
  alternative: `signInByEmail` over GraphQL, then CDP `Storage.setCookies`
  with `trip2g_record_space=<token>` on the demo browser.
- `/_system/onboarding-vault` does NOT accept `X-Api-Key` — it requires an
  admin session (cookie or `Authorization: Bearer <session token>`). preflight
  check 4 therefore always WARNs here; make-golden downloads need the Bearer.

Closed 2026-07-06 (full local wave is GREEN — see the rig changes in
autoproducer `scripts/hero/v4/`):

- `gate.sh`/`run-all.sh`/`reset-stage.sh`/`stage-notes.sh` now honor
  `V4_CONFIG` + `scheme`/`landing`/`api_key`; `record.mjs` matches the landing
  page by the config `landing` value (ports disambiguate the two localhost
  targets) and seg3b types the config landing host.
- golden-vault rebuilt from the LOCAL zip (`V4_CONFIG=... ./make-golden.sh`);
  the prod golden is kept at `golden-vault.prod-honesthesse/`.
- reset-stage now hides ALL server notes via `notePaths` (the static list
  missed the local zip's `robots.md`, so the space never went empty).
- stage-notes synthesizes the plugin sync-state baseline (`seed-baseline.mjs`):
  the sync CLI records `lastSyncedHash` only for files it pushed/pulled, so an
  already-in-sync vault seeded an EMPTY baseline and the first on-camera sync
  popped the one-time "Sync system update" migration dialog mid-take.
- clickSync matches the ribbon by class (`.sync-ribbon-icon`): the pending
  badge rewrites the aria-label to "Trip2g Sync (↑1 to push)".

Product findings (upstream, not harness-specific):

- The onboarding zip's directory entries carry no trailing slash AND are
  written as zero-byte *file* entries, so both plain `unzip` and Python
  `ZipFile.extractall` fail with "exists but is not a directory".
  `unzip -p <zip> <entry>` works; make-golden.sh now extracts file entries
  itself, skipping dir-prefix names. Possibly affects real users.
- Plugin/CLI: `executePlan` never records `lastSyncedHash` for unchanged
  files — a fresh state file over an in-sync vault stays empty, and the next
  edit is misclassified as a first-sync conflict (migration dialog).
