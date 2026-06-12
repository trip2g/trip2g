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

## Render latency benchmark

Measures per-request response time for public note pages: default template vs custom Jet layout.

### Running

```sh
# 1 and 4 cores, note counts 100/1000/10000:
./scripts/bench-render.sh

# Custom core counts:
./scripts/bench-render.sh 1 2 4
```

Results: `tmp/bench/render-results.jsonl` + `render-results.csv` + table on stdout.

### How it works

For each `(cores, notes)` pair:
1. Starts a fresh server with an empty DB.
2. Pushes N synthetic notes without a layout (`bench/note_*`).
3. Runs k6 (10 VUs, 20 s) against those pages — **default scenario**.
4. Pushes `_layouts/bench.html` (a minimal Jet template) + N notes with `layout: /bench` (`bench_layout/note_*`).
5. Runs k6 (10 VUs, 20 s) against the layout pages — **layout scenario**.
6. Extracts p50/p95/p99/avg from `--summary-export` JSON.

Key notes:
- Layout file content must **not** include YAML frontmatter — `note.Content` is passed directly to the Jet parser.
- Note paths must not contain hyphens — the URL normalization converts `-` to `_`. Use `bench_layout/`, not `bench-layout/`.
- `--summary-export` in k6 v1.x stores percentiles directly on the metric object (not under `.values`).

### Reference results (2026-06-12, ARM64 dev machine)

| cores | notes | default p50 | default p99 | layout p50 | layout p99 |
|------:|------:|------------:|------------:|-----------:|-----------:|
| 1 | 100 | 6 ms | 11 ms | 6 ms | 14 ms |
| 1 | 1 000 | 5 ms | 25 ms | 5 ms | 35 ms |
| 1 | 10 000 | 5 ms | 46 ms | 5 ms | 55 ms |
| 4 | 100 | 3 ms | 7 ms | 3 ms | 8 ms |
| 4 | 1 000 | 3 ms | 9 ms | 3 ms | 13 ms |
| 4 | 10 000 | 3 ms | 21 ms | 3 ms | 23 ms |

p50 is identical for both render paths. Jet layouts add ~20% to p99 at large vault sizes (10 k notes).
Both paths serve from the in-memory NoteViews cache; the Jet template is compiled once per layout reload,
not per request. The p99 spread is driven by GC pauses and concurrent pushNotes reloads, not layout rendering itself.

## Notes

- The mock embedding server returns random unit vectors of the correct dimension (1024 for bge-m3). This is sufficient for memory measurement — the in-memory index size depends on count × dimensions, not on vector values.
- Benchmark results vary by machine. The shape of the curves is stable; absolute numbers are not. Always note the hardware and date when publishing results.
- The `GLOBAL_QUEUE_POLL_INTERVAL=100ms` setting is benchmark-only. Production default is 3 s and should not be changed without understanding the queue load implications.
