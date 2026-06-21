# E2E runner rewrite — proposal & spike

Status: exploratory spike on branch `spike/e2e-runner-rewrite`. Nothing on `main`
is modified. The original `scripts/test-e2e.sh` is untouched.

## 1. What `scripts/test-e2e.sh` actually does (phase-by-phase)

The script orchestrates a 4-instance docker stack, seeds data, and runs an
ordered set of Playwright specs whose ordering is load-bearing (global serving
state gets "poisoned" by the last few specs). Phases:

1. **Arg parse / env setup** — extracts `--update-snapshots` (or `UPDATE_SNAPSHOTS=1`),
   passes the rest through; exports `APP_URL` (`:20081`), `GIT_API_BASE_PATH=/git`,
   `USER_TOKEN_COOKIE_NAME=trip2g_e2e`, `ENDPOINT=$APP_URL/graphql`. `set -e`.
2. **Cleanup containers** — `docker compose -f docker-compose.test.yml stop/rm -f`
   for `app app-peer app-peer2 app-peer3 minio test-data` (embedding is kept warm).
3. **Prepare DB** — `DB_PATH=tmp/data/test.sqlite3`; rm db + `-shm`/`-wal`;
   seed with `testdata/e2e_seed.sql`. Then either (`ENABLE_TG=1`) run
   `go run ./cmd/tge2e ... patch-db`/`cleanup`, or wipe ~20 telegram tables via sqlite3.
4. **Start services** — `up -d --no-recreate embedding`, then `up -d --build app
   app-peer app-peer2 app-peer3 minio`; health-wait all four via `./scripts/waitfor`
   on `:20081/:20091/:20093/:20095`.
5. **Setup spec** — `npx playwright test e2e/setup.spec.js`; reads minted key from
   `.test-api-key` into `$API_KEY`.
6. **CLI sync suite** — `./scripts/test-sync-cli.sh --api-key … --endpoint … [--update-snapshots]`.
   That sub-script is a ~900-line harness of ~20 sync test cases (conflict modes,
   assets, deletes, dry-run, exclude, meta injection) with golden-snapshot diffing.
7. **Seedvault → peers** — three near-identical functions (`sync_seedvault_to_peer`,
   `…2_to_peer2`, `…3_to_peer3`): per peer (`:20091/:20093/:20095`) request sign-in
   code → sign in (dev code `111111`) → create API key (peer-specific cookie) →
   `npx tsx obsidian-sync/src/sync/cli/cmd.ts --folder testdata/seedvaultN …`.
8. **(ENABLE_TG) telegram cron** — `curl /debug/run_cron_job?name=send_scheduled_telegram_publishposts`.
9. **MANUAL mode** — print URLs/key, wait on ENTER, exit.
10. **Drain jobs** — `wait_all_jobs` (`curl /debug/wait_all_jobs`, grep `ok:`).
11. **Main Playwright group** — `npx playwright test --grep-invert
    "Setup|Layout CSS|Webhook|Screenshot|Bidirectional Federation"` (+ `--headed/--debug/--ui`).
12. **Screenshots** — `e2e/screenshots.spec.js`.
13. **CSS hot-reload** — append `body { color:#f00 }` to a layout CSS, `sync_vault`
    (`tmp/testvault0`), then `e2e/layoutcss.spec.js`.
14. **(ENABLE_TG) telegram update block** — wait jobs, `tge2e … check` step0,
    mutate `tmp/testvault0/telegram_*.md` (+ photo embed, sed photo swap), re-sync,
    wait jobs, `tge2e … check` step1.
15. **Ordered tail (ordering is critical)** —
    `federation-bidir.spec.js` → `webhooks.spec.js` →
    `RUN_ISOLATED_SPECS=1 unreleased-changes.spec.js` →
    `RUN_ISOLATED_SPECS=1 show-draft-versions.spec.js` (DEAD LAST: flips global
    `show_draft_versions`, poisons public serving for anything after it).
16. **Cleanup trap** — currently mostly commented out (down/rm disabled to keep the
    stack warm); just prints "Cleanup complete".

Load-bearing details that any rewrite MUST preserve:
- Exact env var names/values and the per-peer cookie names (`trip2g_e2e_peer{,2,3}`).
- `RUN_ISOLATED_SPECS=1` on the last two specs.
- Spec ordering (webhooks before unreleased-changes before show-draft-versions).
- Snapshot-verify default; `--update-snapshots` only on opt-in.

## 2. Options & recommendation

| Option | Pros | Cons |
|--------|------|------|
| **google/zx (.mjs)** | Same ecosystem as Playwright + `tsx`; no new runtime (Node 24 already here, `npx zx 8.8.5` already resolves); top-level await; `$\`…\`` shells out cleanly; real `fs`/`fetch`/JSON for the GraphQL sign-in/key-mint dance (replaces brittle `grep -o '"token":"…"' \| cut`); functions/loops to kill the 3× copy-pasted peer blocks. | Adds one devDep; still imperative glue. |
| **Python (subprocess/sh)** | Readable; great data handling. | Introduces a **Python toolchain into a Node/Go repo + CI** purely for a test runner — new venv/lockfile/setup-python step, context-switch for contributors, and it still shells out to the same `npx`/`docker`/`playwright`. No real payoff here. |
| **Modular bash (`scripts/e2e/*.sh`)** | Zero new runtime; smallest change. | Still bash: the genuinely painful parts (GraphQL JSON parsing via grep/cut, snapshot diffing, the 3 duplicated peer flows) stay painful. |

**Recommendation: google/zx.** This is a Node/Go repo; the runner's hard parts are
JSON/GraphQL handling and shelling out to `npx playwright` / `npx tsx` / `docker` —
all of which zx does natively while staying in the existing ecosystem. Python's only
real edge (data wrangling) doesn't apply, and it taxes CI/onboarding. Modular bash
is the cheapest but leaves the actual pain (grep-parsing GraphQL, duplicated peer
blocks, snapshot diffs) untouched. zx is the best tradeoff: meaningfully simpler
code, no new language toolchain.

Caveat: the win is real but moderate. The biggest source of complexity is the
**required spec ordering + global-state poisoning**, which is inherent to the test
design, not the shell — no runtime choice removes it. If the team wants to keep
bash, the modular split is a legitimate fallback.

## 3. What the prototype covers

`scripts/test-e2e.mjs` ports a representative vertical slice in zx:
- arg parsing (`--update-snapshots`, `--headed/--debug/--ui`, `--manual`),
- container cleanup (phase 2),
- DB prepare + seed + telegram-table wipe (phase 3, incl. `ENABLE_TG` branch),
- service up + 4× health-wait via `./scripts/waitfor` (phase 4),
- setup spec + `.test-api-key` read (phase 5),
- main Playwright group with the exact `--grep-invert` filter (phase 11),
- `waitAllJobs()` helper implemented (used by stubs),
- MANUAL-mode short-circuit.

Verification done in this spike (no heavy infra launched, per constraints):
- `node --check scripts/test-e2e.mjs` → **SYNTAX OK**
- `npx zx --version` → **8.8.5** (already resolvable; no global install needed)

The docker/playwright phases were **not executed** (would rebuild the stack /
run the suite — out of scope for the spike).

## 4. Remaining work to port the rest

Marked `STUB:` in `scripts/test-e2e.mjs`:
- **CLI sync suite** — port `scripts/test-sync-cli.sh` (~20 cases, snapshot diff via
  `jq sort_by(.path)`). Biggest chunk; best done as its own `scripts/e2e/sync-cli.mjs`
  module with a tiny assert/snapshot helper. This is where zx pays off most (real
  JSON compare instead of `jq` + `diff`).
- **Seedvault → peer pushes** — collapse the 3 duplicated bash functions into one
  parametrized `pushSeedvault({port, cookie, folder})` using `fetch` for the GraphQL
  sign-in/key-mint (drops the `grep -o … | cut` parsing).
- **Telegram block** (`ENABLE_TG`) — cron trigger, `tge2e … check` step0/step1, the
  `telegram_*.md` mutations (append, photo embed, `sed`-style photo swap), re-sync.
- **Ordered Playwright tail** — screenshots, CSS hot-reload + layoutcss,
  federation-bidir, webhooks, then `RUN_ISOLATED_SPECS=1` unreleased-changes and
  show-draft-versions (preserve order + env exactly).

Adoption steps if accepted:
1. `npm i -D zx`
2. Repoint `package.json` `test:e2e*` scripts to `npx zx scripts/test-e2e.mjs …`.
3. Port the stubs module-by-module, diffing behaviour against the bash original.
4. Delete `scripts/test-e2e.sh` (and fold `test-sync-cli.sh`) once parity is verified.
