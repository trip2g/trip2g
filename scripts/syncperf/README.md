# syncperf — obsidian-sync performance benchmark

Isolated harness to measure note-sync time on ~2000 notes, **without** vector
search noise, then iterate on fixes and re-measure.

## One-time per session

```bash
# builds plugin, seeds a fresh DB, starts the stack, mints an API key,
# generates tmp/syncperf-vault with 2000 notes + a working trip2g plugin
node scripts/syncperf/setup.mjs            # add --rebuild to rebuild the docker image
```

When it prints `READY`, open Obsidian on the vault for the live cross-check:

```bash
obsidian "$(pwd)/tmp/syncperf-vault"       # or File → Open vault
```

## Measure (CLI — precise, scriptable)

```bash
node scripts/syncperf/bench.mjs --label baseline
# ... apply a fix in obsidian-sync/src/sync/* ...
node scripts/syncperf/bench.mjs --label "fix:cache"
```

Results append to `docs/dev/obsidian_sync_bench_2026-06-21.md` (baseline vs fix
in one file). Scenarios: cold / noop / dry-noop / small-change / twoway-noop.
`noop − dry-noop ≈ execute/asset overhead`.

## Live plugin cross-check (Obsidian open)

The CLI runs the same `classify`/`execute` code. To sanity-check the real plugin,
trigger its sync command from the running Obsidian via `~/.local/bin/obsidian`
(obsidian-cli) and watch the result Notice. After a TS fix: rebuild
(`cd obsidian-sync && npm run build`) and re-copy `main.js` into the vault
(`node scripts/syncperf/gen-vault.mjs --out tmp/syncperf-vault` re-copies it);
the hot-reload plugin reloads trip2g.

## Teardown

```bash
docker compose -f docker-compose.syncperf.yml down -v
```

## Pieces

| File | Role |
|---|---|
| `docker-compose.syncperf.yml` | app + minio, vector search OFF, ports 20071/29100 |
| `gen-vault.mjs` | deterministic 2000-note vault + isolated `.obsidian`/plugin |
| `setup.mjs` | seed DB → up → mint key → generate vault |
| `bench.mjs` | timed scenarios → append to the bench report |
