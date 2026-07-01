# pushNotes benchmark

How to re-run the performance benchmark and update the user-facing docs.

## Prerequisites

```sh
docker compose -f docker-compose.test.yml up -d minio
```

MinIO is required at boot (asset storage config is mandatory). No real assets are uploaded during the benchmark.

## Running

```sh
# Baseline (no vector search) — 1 and 4 CPU cores, all note counts:
./scripts/bench-pushnotes.sh

# With vector search — uses a local mock embedding server on :19001
# so all embedding jobs complete instantly (memory measurement only):
VECTOR=1 ./scripts/bench-pushnotes.sh

# Custom core counts:
./scripts/bench-pushnotes.sh 1 2 4 8
VECTOR=1 ./scripts/bench-pushnotes.sh 4
```

Results append to `tmp/bench/results.jsonl` (baseline runs truncate first; VECTOR=1 appends). A human-readable table and `tmp/bench/results.csv` are printed at the end.

## How it works

Each `(cores, notes)` pair:
1. Starts a fresh server binary (`tmp/bench/trip2g-bench`) with an empty SQLite DB, pinned to CPUs via `taskset` and `GOMAXPROCS`.
2. Authenticates, creates an API key, pushes N synthetic notes (initial push), fires a concurrent probe push to measure lock contention, then runs incremental pushes (1 note and 10% of vault, median of 3).
3. In `VECTOR=1` mode: waits for all embedding jobs to drain (`/debug/wait_all_jobs`), then pushes one more revision so the in-memory vector index is reloaded before reading RSS.
4. Reads RSS from `/metrics` and peak RSS from `/proc/<pid>/status`.
5. Kills the server (SIGKILL) and verifies the port is free before the next run.

Key knobs in the script:
- `PORT=28081` / `INTERNAL_PORT=28082` — ports used by the bench server
- `GLOBAL_QUEUE_POLL_INTERVAL=100ms` — speeds up job queue for VECTOR runs (default is 3 s in production)
- `MOCK_EMBED_PORT=19001` — mock embedding server port (started automatically in VECTOR=1 mode)

## Updating user docs

After a run, copy the relevant columns from `tmp/bench/results.csv` into:
- `docs/en/user/pushnotes_bench.datachart.csv`
- `docs/ru/user/pushnotes_bench_ru.datachart.csv`

Column mapping (wide format, one row per note count):

| CSV column | Source field |
|-----------|-------------|
| `notes` | `notes` |
| `initial_1c` / `initial_4c` | `initial_push_ms` where `cores=1` / `4`, `vector=false` |
| `incr1_1c` / `incr1_4c` | `incr_1_ms` |
| `incr10p_1c` / `incr10p_4c` | `incr_10p_ms` |
| `rss_1c` / `rss_4c` | `rss_mb` |
| `peak_1c` / `peak_4c` | `peak_rss_mb` |
| `rss_vec_1c` / `rss_vec_4c` | `rss_mb` where `vector=true` |
| `peak_vec_1c` / `peak_vec_4c` | `peak_rss_mb` where `vector=true` |

Then sync to the dev server:

```sh
cd obsidian-sync
npm run sync -- ../docs -u http://localhost:8081/_system/graphql -k <api-key>
```

The API key is in `docs/.obsidian/plugins/trip2g/data.json`.

## Notes

- The mock embedding server returns random unit vectors of the correct dimension (1024 for bge-m3). This is sufficient for memory measurement — the in-memory index size depends on count × dimensions, not on vector values.
- Benchmark results vary by machine. The shape of the curves is stable; absolute numbers are not. Always note the hardware and date when publishing results.
- The `GLOBAL_QUEUE_POLL_INTERVAL=100ms` setting is benchmark-only. Production default is 3 s and should not be changed without understanding the queue load implications.
