# modernc.org/sqlite — driver details, metrics & gaps

**TL;DR.** trip2g's production database runs on the **pure-Go `modernc.org/sqlite` driver** (v1.53.0), not the CGO `mattn/go-sqlite3`. The driver exposes more than we use: per-connection `DBStatus` counters, `FileControl`, online backup, custom SQL functions, and several DSN knobs. The one observability win worth shipping is **`sql.DBStats` + file/WAL gauges** (the `db_status` per-conn counters are the *wrong* granularity for our pooled, single-writer setup). The cheapest correctness win is **`_dqs=false`**. The `modernc.org/libc` version must stay pinned to exactly what the driver's `go.mod` requires (`v1.73.4` today) — a soft invariant worth a CI guard.

---

## 1. Which driver we actually use

| Path | Driver | Registered name | Notes |
|------|--------|-----------------|-------|
| App runtime (`internal/db/setup.go`, `internal/dbmate/sqlite`, `internal/simplebackup`) | `modernc.org/sqlite` | `"sqlite"` | Pure Go, no CGO. The real production path. |
| `cmd/tge2e` e2e tooling, `cmd/server/queue_test.go` | (was `mattn/go-sqlite3`) → `modernc.org/sqlite` | `"sqlite"` | Migrated off mattn. |

`github.com/mattn/go-sqlite3` is **no longer compiled into any binary** (`go list -test -deps ./...` shows only `modernc.org/sqlite`). It survives in `go.mod` as `// indirect` solely because `github.com/amacneil/dbmate/v2` requires it in *its own* `go.mod` (dbmate ships a mattn-based `pkg/driver/sqlite`). Fully removing the line needs a dbmate release/fork that drops mattn — out of scope.

> The doc string *"DBStatus\* are the operations accepted by `DBStatus.Status`. They report their value differently depending on the op"* comes from **`modernc.org/sqlite@v1.53.0/dbstatus.go`** — the production driver — not from mattn.

## 2. DSN params & pragmas in use

All connection settings go through **modernc DSN params** (`_pragma`, `_txlock`, `_dqs`), because the driver applies them to *every* pooled connection. A one-off `db.Exec("PRAGMA …")` would only configure the single pooled conn it happened to run on. Built in `openConnection()` (`internal/db/setup.go`).

| Setting | Value | Why |
|---------|-------|-----|
| `busy_timeout` | `20000` ms | Queue instead of failing on lock contention. |
| `journal_mode` | `WAL` | Concurrent readers + single writer. |
| `foreign_keys` | `1` | Enforce FKs (immediate). |
| `synchronous` | `NORMAL` | WAL-appropriate durability/throughput. |
| `temp_store` | `2` (MEMORY) | Temp tables/indexes in RAM. |
| `mmap_size` | `268435456` (256 MB) | Memory-mapped reads. |
| `cache_size` | `-64000` (64 MB) | Per-conn page cache. |
| `wal_autocheckpoint` | `1000` pages | Bound WAL growth. |
| `query_only` | `1` *(replica only)* | Hard read-only guardrail. |
| `_txlock` | `immediate` *(write pool only)* | `BEGIN IMMEDIATE` takes the write lock at txn start, avoiding the deferred→write upgrade that fails instantly with `SQLITE_BUSY_SNAPSHOT` (517). **Correctly scoped to the writer; do not add to the read pool.** |
| `_dqs` | `false` | Disable the double-quoted-string-literal fallback so DQS mistakes become parse errors instead of silent string literals (<https://www.sqlite.org/quirks.html#dblquote>). Cheap correctness hardening. |

**Pool topology** (`cmd/server/boot.go`, `internal/db/setup.go`): three separate `*sql.DB` pools — `Read` (`MaxOpenConns=25`, `ConnMaxIdleTime=30s`), `Write` (`MaxOpenConns=1`), `Queue` (`MaxOpenConns=1`, isolated so goqite polling never blocks app writes). In read-replica mode all three collapse to one strict-read-only pool.

## 3. `DBStatus` (`sqlite3_db_status`) — why it is *not* our primary metric source

modernc exposes `sqlite3_db_status` via an escape hatch:

```go
err := sqlConn.Raw(func(dc any) error {
    cur, hi, err := dc.(sqlite.DBStatus).Status(sqlite.DBStatusCacheSpill, false)
    // …
    return err
})
```

These counters live on an **individual `sqlite3*` handle**. We never hold one: `database/sql` owns 25 read conns + 1 writer and rotates which one `Raw()` hands you, recycling them on `ConnMaxIdleTime=30s`. Consequences:

- **No per-conn identity** — you sample a random, drifting subset; recycled conns reset their counters to zero, so a Prometheus *counter* built on this sawtooths and lies.
- **`reset=true` is destructive/racy** across samplers.
- The *memory* ops (`CacheUsed`/`StmtUsed`/`SchemaUsed`) are gauges and less broken, but still scoped to "whichever conn answered" — not actionable.

If we ever chase a cache-tuning question, the only ops with signal for *this* workload (WAL + 256 MB mmap + 64 MB cache + single writer) are, ranked: **`CacheMiss`/`CacheHit`** (is the 64 MB cache sized right?), **`CacheSpill`** (dirty-page flush pressure on the writer), **`CacheWrite`**, and maybe **`CacheUsed`**. Everything else (`StmtUsed`, `SchemaUsed`, `Lookaside*`, `DeferredFKs`, `CacheUsedShared`, `TempbufSpill`) is noise here. Even those four would be a best-effort *smell test* behind an opt-in flag, never SLO metrics.

## 4. What we export to Prometheus instead

Added in `internal/metrics/db.go`, registered in `cmd/server/server.go` `startInternalServer()` before the `/metrics` handler.

**Pool metrics** — `collectors.NewDBStatsCollector(db, name)` for each pool, label `db_name="read"|"write"|"queue"`. Emits `go_sql_*`, including:
- `go_sql_open_connections`, `go_sql_in_use_connections`, `go_sql_idle_connections`
- `go_sql_wait_count_total`, **`go_sql_wait_duration_seconds_total`** ← the write-pool (`MaxOpenConns=1`) backpressure signal we were flying blind on
- `go_sql_max_idle_closed_total`, `go_sql_max_lifetime_closed_total` (churn)

**File / WAL gauges** — a small custom collector sampling the read pool (PRAGMAs) + the filesystem:
- `trip2g_db_size_bytes` = `page_count * page_size`
- `trip2g_db_freelist_pages` (`PRAGMA freelist_count`) — fragmentation / VACUUM-need
- `trip2g_db_wal_bytes` = `os.Stat(<dbfile>-wal)` size

> WAL size is read from the **filesystem**, *not* via `PRAGMA wal_checkpoint(PASSIVE)`: a checkpoint has side effects, fails on `query_only` replicas, and is Litestream-hostile. The PRAGMAs used are pure reads and work under `query_only`.

This closes the audit gaps (`WaitDuration` invisible, read pool unobserved, no file/WAL signal) with ~20 lines and no per-conn-identity hand-wringing.

## 5. modernc features trip2g does **not** use yet

| Capability | What it enables | Relevance | Verdict |
|---|---|---|---|
| `DBStatus` | Per-conn cache/mem counters via `Raw` | low | Pool rotation kills the signal — prefer `DBStats` + PRAGMA (§4). |
| `FileControl` → `FileControlDataVersion` | Cheap conn-local "did anyone write since I last looked" (pager `DATA_VERSION`) | **med** | Worth a spike as a **read-side cache-invalidation** primitive (addresses the read-replica "silent hole"). |
| Online Backup API (`Backup.Step`) | Incremental page-copy snapshot, yields the write lock between steps | **med** | Only if backup-time lock contention ever shows up in `WaitDuration`. `VACUUM INTO` (current) is simpler and defragments — keep it for now. |
| `Serialize`/`Deserialize` | DB ↔ `[]byte` | skip | Memory bomb for multi-GB DBs; the S3 gzip-stream path is correct. Test fixtures only. |
| `RegisterDeterministicScalarFunction` / scalar / aggregate UDFs | Run Go in SQL (deterministic ones usable in indexes) | low | Vector scoring stays faster in-app; couples ranking to the DB. Only useful for a future computed/partial index. |
| `RegisterCollationUtf8` | Custom sort/compare (locale/Cyrillic) | low | Defer until a concrete `NOCASE`-insufficient sort bug appears. |
| `Limit` (`sqlite3_limit`) | Per-conn caps (SQL length, expr depth, …) | low | Hardening for untrusted SQL; all SQL today is first-party (sqlc/dbmate). |
| preupdate/commit/rollback hooks | In-driver change reactions | low | Redundant with the app-level `note_changes`/SSE change feed. |
| Virtual tables, custom VFS, pluggable page cache | Expose Go as SQL / replace IO layer | skip | No use case; Litestream/standard VFS is what we want. |
| DSN `_time_*` / `_inttotime` / `_texttotime` | Time storage/parse format | skip | We control schema + Go types; changing is a data-format migration risk for zero gain. |
| DSN `_error_rc=true` | Clearer open-time error strings | low | Harmless nice-to-have. |
| FTS5 | Keyword full-text index | low | Different feature from embedding search; would be net-new (hybrid lexical+semantic), not a gap. |
| json1 / math / `DBSTAT_VTAB` | JSON/math in SQL; page-usage introspection | low | `STAT4` already helps via the `ANALYZE` vacuum cron; `DBSTAT_VTAB` is an ad-hoc debugging aid. |

**Genuinely actionable:** `_dqs=false` (done, §2), a `FileControlDataVersion` spike for read-side invalidation, and revisiting the online Backup API only if `WaitDuration` later proves backup lock contention.

## 6. `modernc.org/libc` version invariant

The driver is transpiled C running on `modernc.org/libc`'s runtime, so a libc that is newer or older than the one the driver was generated against can mismatch in subtle, hard-to-debug ways (bad codegen interactions, not a clean compile error). The upstream guidance: **use the exact `modernc.org/libc` version from the driver's `go.mod`.**

**Status: satisfied today.** `modernc.org/sqlite@v1.53.0`'s `go.mod` requires `modernc.org/libc v1.73.4`; trip2g's `go.mod` pins `v1.73.4`; the resolved build list agrees.

**Risk:** nothing enforces it, and libc is an `// indirect` dep — drift leaves no diff in our own code. It can move via `go get -u`, an unrelated dependency raising its libc floor, or a `modernc.org/sqlite` bump that pins a *different* libc (making our stale explicit pin wrong).

**Guard (recommended):**
1. Annotate the pin: `modernc.org/libc v1.73.4 // indirect; MUST match modernc.org/sqlite's required libc`
2. CI assertion:
   ```bash
   want=$(grep -E '^\s*modernc.org/libc ' "$(go mod download -json modernc.org/sqlite | jq -r .Dir)/go.mod" | awk '{print $2}')
   have=$(go list -m -f '{{.Version}}' modernc.org/libc)
   [ "$want" = "$have" ] || { echo "libc drift: sqlite wants $want, build has $have"; exit 1; }
   ```
3. Re-pin libc in the same PR whenever you bump `modernc.org/sqlite`.

## 7. References

- modernc DSN params: `modernc.org/sqlite@v1.53.0/driver.go` (`_pragma`, `_txlock`, `_dqs`, `_time_format`, `_error_rc`, …)
- `DBStatus`: `modernc.org/sqlite@v1.53.0/dbstatus.go`
- SQLite double-quote quirk: <https://www.sqlite.org/quirks.html#dblquote>
- `sqlite3_db_status` ops: <https://www.sqlite.org/c3ref/c_dbstatus_options.html>
- Related docs: [`sqlite.md`](./sqlite.md) (config, WAL, R/W split), [`simplebackup.md`](./simplebackup.md).
