# SQLite "database is locked" — root causes and fixes

Debugging session (2026-06-10). Started from a recurring, seemingly random e2e
failure that had been haunting the suite for a long time:

```
failed to upsert API key log action: database is locked (517)
```

plus `wait_all_jobs` returning 500 in `queue.spec.js` and a flaky
`personal-tokens` test. The investigation found four independent problems
stacked on top of each other. This doc describes each one, the fix, and — most
importantly — how to find this class of error, because none of it was visible
from the error message alone.

## TL;DR

| Problem | Fix |
|---------|-----|
| DSN used mattn-style params; modernc.org/sqlite silently ignored them — most pool connections ran with `busy_timeout=0`, `foreign_keys=OFF` | Pass pragmas as `_pragma=` DSN params (applied per connection) |
| Write transactions began deferred → `SQLITE_BUSY_SNAPSHOT` (517) on read→write upgrade; busy_timeout can never help with 517 | `_txlock=immediate` for writer pools |
| `refresh_chart_data` jobs failed on unreachable demo URL, retried, and poisoned the queue forever | Fetch failures log + keep stale cache, job completes |
| `wait_all_jobs` treated *any* retry as failure | Fail only on dead jobs (exhausted `MaxReceive`) |
| Browser tests ran while embedding jobs were still indexing → nondeterministic search results | Drain the queue between sync and the main Playwright phase |
| Queue mutations held the write lock while waiting for in-flight jobs whose goqite delete needed that lock → ~20s stalls, completed jobs redelivered | `@skipTx` on stop/start/clearBackgroundQueue |

## Problem 1: pragmas were never applied to most connections

**Symptom.** `busy_timeout=20000` was configured, yet "database is locked"
errors returned instantly.

**Root cause.** `internal/db/setup.go` built the connection string with
mattn/go-sqlite3 parameters:

```go
q.Set("_journal", "WAL")
q.Set("_timeout", "20000")
q.Set("_busy_timeout", "20000")
```

The project uses **modernc.org/sqlite**, which supports only three DSN
parameters: `_pragma`, `_txlock`, `_time_format`. Everything else is
**silently ignored** (see `applyQueryParams` in the driver — it only reads
those three keys; no error for unknown ones).

The fallback `enablePragmas()` ran `db.Exec("PRAGMA busy_timeout = 20000; ...")`
— but `database/sql` is a *pool*. A one-off Exec configures only the single
pooled connection it happens to grab. The read pool holds up to 25
connections with a 30s idle timeout; every other/new connection ran with
SQLite defaults: `busy_timeout=0`, `foreign_keys=OFF`.

**Fix.** All pragmas moved into the DSN, where the driver applies them to
**every new connection**:

```go
q.Add("_pragma", "busy_timeout(20000)")  // driver applies this one first
q.Add("_pragma", "journal_mode(WAL)")
q.Add("_pragma", "foreign_keys(1)")
// ... synchronous, temp_store, mmap_size, cache_size, wal_autocheckpoint
```

`enablePragmas()` deleted (it also contained `PRAGMA strict = ON`, which is
not a real SQLite pragma — a silent no-op all along).

**Test.** `TestPragmasAppliedToAllConnections` pins 5 connections from the
pool simultaneously via `pool.Conn(ctx)` and asserts `PRAGMA busy_timeout`
and `PRAGMA foreign_keys` on each one.

## Problem 2: deferred transactions → SQLITE_BUSY_SNAPSHOT (517)

**Symptom.** The original error. Code 517 = `SQLITE_BUSY_SNAPSHOT`
(5 | 2<<8). Flaky, load-dependent, immune to busy_timeout.

**Root cause.** The app has multiple writer pools on the same file:
`writeConn` and `queueConn` (`cmd/server/main.go: initDBs`). All transactions
started with `BeginTx(ctx, nil)` → plain deferred `BEGIN`.

The failing sequence (GraphQL request → `AcquireTxEnvInRequest`):

1. `BEGIN` (deferred) — transaction starts as a *reader*.
2. First SELECT inside the tx takes a WAL read snapshot.
3. Meanwhile `queueConn` (background job) commits a write.
4. The tx's first write (`UpsertAPIKeyLogAction`) tries to upgrade
   read → write on a now-stale snapshot → **517, returned immediately**.

Key insight: **busy_timeout cannot fix 517.** SQLite returns it without
waiting, because waiting can't un-stale a snapshot — the transaction must be
rolled back and restarted. Per-statement retries inside the tx don't help
either. That's why 20 seconds of timeout changed nothing for years.

**Fix.** `_txlock=immediate` on non-read-only pools. Writers take the write
lock at `BEGIN`, so the read-snapshot-upgrade scenario cannot occur;
concurrent writers now queue on busy_timeout (which finally works, see
Problem 1) instead of failing.

**Test.** `TestWriteTxUpgradeNoBusySnapshot` reproduces the exact race with
two `Setup()` pools — it failed with `database is locked (517)` before the
fix, deterministic red/green.

## Problem 3: chart-refresh jobs poisoned the queue

**Symptom.** `wait_all_jobs` returned 500 in `queue.spec.js`; 229 messages
sitting in `goqite` with `received=3`.

**Root cause chain.**

- The e2e vault is a copy of `docs/demo` (`scripts/test-sync-cli.sh`).
- `docs/demo/datachart_fenced.md` and `datachart_types.md` hardcode
  `http://localhost:8090/v1/query` — a SQL endpoint that exists only on the
  dev machine, not in the test containers.
- Every vault sync enqueued `refresh_chart_data` jobs → "connection refused"
  → error → goqite retried (`MaxReceive=3`) → after the third failure the
  message is **never delivered again but stays in the table forever**.
- `wait_all_jobs` saw `retry_count > 0` → 500. One unreachable URL in demo
  content = permanently broken e2e (and dead rows accumulating in any
  production DB with a dead chart source).

**Fix.** `refreshchartdata.Resolve` treats fetch problems (unreachable host,
non-200, non-JSON) as *expected conditions*: log a warning, keep the stale
cache, return nil so the job completes. The chart TTL / refresh cron retries
later anyway. Only `SaveChartData` failures (our own storage) remain errors.

## Problem 4: wait_all_jobs failed on any retry

**Symptom.** Even after Problem 3 was fixed, `queue.spec.js` failed once:
20 `generate_note_version_embedding` jobs transiently failed
("context canceled" under load), were retried, and **succeeded** — but
`wait_all_jobs` had already returned 500 because its rule was
"`received > 1` on any message = failure".

**Root cause.** The invariant was too strict: a retry is goqite's designed
recovery mechanism for transient errors, not a failure.

**Fix.** The endpoint now fails only on **dead** jobs — `received >=
MaxReceive` *and* visibility timeout expired, i.e. messages goqite will never
redeliver (`isDeadJob` in `cmd/server/debug.go`). Jobs still being retried
keep the endpoint polling. `appQueue` remembers its `MaxReceive` (3 default,
2 for some telegram queues). The 500 response now lists each dead job's id,
attempt count, and payload, so the Playwright error names the culprit
directly instead of "queue has failed jobs with retries".

## Problem 5: browser tests raced the embedding backlog

**Symptom.** After Problem 4's fix, two new flakes surfaced:
`personal-tokens` test 3 (two parallel MCP searches returned different result
*sets* — an embedding chunk got indexed between them) and `queue.spec`
timing out at Playwright's default 30s while `wait_all_jobs` correctly waited
out unrelated retrying jobs.

**Root cause.** The vault syncs enqueue hundreds of
`generate_note_version_embedding` jobs, and the main Playwright phase started
while they were still being processed — every search-dependent test ran
against a moving index.

**Fix.** `scripts/test-e2e.sh` drains the queue (`wait_all_jobs`) after the
sync/seedvault phase, before the main Playwright run. The queue test also got
`test.setTimeout(150_000)` so its budget covers the endpoint's 120s wait.

## Problem 6: queue mutations deadlocked against their own transaction

**Symptom.** `queue.spec.js` "clear removes all pending jobs" timed out at
30s; every Stop/Start/Clear queue mutation in the e2e logs took ~20s
(suspiciously close to busy_timeout). Reproduced live on the dev server:
`startBackgroundQueue` with one in-flight 20s job took **21.2s** and the log

```
Ran job duration=20000ms debug_sleep_job
Error deleting job from queue, it will be retried error="cannot start tx: context deadline exceeded"
```

— the *completed* job got redelivered and ran two more times.

**Root cause.** A consequence of `_txlock=immediate` (Problem 2's fix)
meeting the per-request transaction design: the GraphQL tx middleware opens
`BEGIN IMMEDIATE` for every mutation, so the mutation holds the **SQLite
write lock for its entire request duration**. `startBackgroundQueue` →
`appQueue.start()` → waits for the runner's in-flight job → job finishes →
goqite tries to DELETE the message on `queueConn` → its `BEGIN` needs the
write lock *our own mutation holds* → blocks until deadline → delete fails →
message redelivered. Circular wait, broken only by the timeout. The same
mechanism explains the `generate_note_version_embedding` "context canceled"
retries: completed jobs redelivered because some request's long-held tx
blocked their delete.

**Fix.** `@skipTx` on `stopBackgroundQueue` / `startBackgroundQueue` /
`clearBackgroundQueue` — they manage the runner, not data (precedent:
`runCronJob` already had `@skipTx` with a "would deadlock" comment).
Verified live: 15.1s (only the inherent wait for the in-flight job), job ran
once, delete succeeded.

**Architectural note.** The deeper tension remains: the GraphQL middleware
(`internal/graph/handler.go`, `AcquireTxEnvInRequest`) holds the write
transaction for the entire mutation, so with `_txlock=immediate` the slowest
mutation serializes all writers. The worst offender is `PushNotes`: after its
DB writes it runs the full note reload (`PrepareLatestNotes`,
`internal/case/pushnotes/resolve.go`) — seconds of rendering — with the tx
still open. (Embeddings are fine: they run in the
`generate_note_version_embedding` background job on the plain write pool, not
inside the request tx; they were victims of the held lock, not holders.)
Any resolver that blocks on the queue/runner must be `@skipTx`; long-term,
shrinking tx scope to the actual write section — e.g. committing before the
reload — would remove this class of stall entirely.

## How to find this class of error

What actually located each root cause — none of it was in the error message:

1. **Read the extended result code.** The number in parentheses is the
   answer: 5 = BUSY (lock contention, busy_timeout/retry helps),
   517 = BUSY_SNAPSHOT (deferred-tx upgrade, only `_txlock=immediate` helps),
   261 = BUSY_RECOVERY. Table in `docs/dev/sqlite.md`.
2. **Verify pragmas per connection, not per pool.** Pin several connections
   (`pool.Conn(ctx)`) and query `PRAGMA busy_timeout` on each. A green
   "pragmas configured at startup" log means nothing for the rest of the pool.
3. **Count the writer pools.** `grep "db.Setup\|sql.Open"` — every `*sql.DB`
   on the same file is an independent connection set. 517 needs at least two
   writers; we had `writeConn`, `queueConn`, plus migrations and host-side
   CLI access from tests.
4. **The app runs in Docker during e2e.** `pgrep` on the host finds nothing;
   logs are at `docker compose -f docker-compose.test.yml logs app`
   (containers stay up after the run — the script stops them at the *start*
   of the next run).
5. **The job queue is just a table.** When `wait_all_jobs` complains, look at
   the evidence directly:
   `sqlite3 tmp/data/test.sqlite3 "select queue, count(*), sum(received > 1) from goqite group by queue"`.
   The payload is gob-encoded; `hex(body)` is readable enough to spot the job
   name (`refresh_chart_data` was visible in plain hex).
6. **Driver docs over assumptions.** The decisive fact — modernc supports
   only `_pragma`/`_txlock`/`_time_format` and ignores everything else — came
   from reading `applyQueryParams` in the driver source in the module cache,
   not from any error output.

## Files changed

- `internal/db/setup.go` — `_pragma` DSN params, `_txlock=immediate` for
  writers, `enablePragmas` removed
- `internal/db/setup_test.go` — `TestPragmasAppliedToAllConnections`,
  `TestWriteTxUpgradeNoBusySnapshot`
- `internal/case/backjob/refreshchartdata/resolve.go` — fetch failures keep
  stale cache, complete the job
- `internal/case/backjob/refreshchartdata/resolve_test.go` — updated contract
  + `TestResolve_UnreachableHost`
- `cmd/server/queue.go` — `appQueue.maxReceive`
- `cmd/server/debug.go` — `isDeadJob`, `describeDeadJobs`, wait_all_jobs
  fails only on dead jobs
- `cmd/server/debug_test.go` — `TestIsDeadJob`
- `scripts/test-e2e.sh` — drain queue before the main Playwright phase
- `e2e/queue.spec.js` — timeout covering the full wait budget
- `internal/graph/schema.graphqls` — `@skipTx` on queue-control mutations
  (+ `make gqlgen`)
- `docs/dev/sqlite.md` — corrected connection-parameter docs, added the
  busy-code troubleshooting table

## Residual known flake

One `stepping, database is locked (5)` (plain BUSY, not 517) appeared once in
`personal-tokens` test 6 and passed on retry. Plausible cause: long
`PushNotes` transactions (embedding chunk upserts inside one immediate tx)
holding the write lock while `queueConn` writes wait past 20s under full-suite
load. If it recurs, candidates: shorten the sync transaction (move embedding
HTTP calls out of the tx), or wrap cross-pool writes in `db.WithRetry`.
