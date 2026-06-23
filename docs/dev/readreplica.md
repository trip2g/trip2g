# Read replica — implementation & validation report

**TL;DR:** trip2g now has a read-only replica mode (`--leader-addr`). A replica serves
GET locally off a LiteFS-replicated SQLite file and reverse-proxies every mutating
request to the leader's internal port, authenticated with an `X-Replica-Auth` HMAC.
Built TDD, deployed to two Hetzner VMs, and validated: identical reads, write
forwarding, 401 on unauthenticated intake, **single-digit-ms replication lag (median
9 ms)**, read-only guardrail, and **60/60 reads with zero failures during a leader
restart**. User guide: [[read-replica]].

---

## What was built

A replica is a normal trip2g process started with one extra switch — `--leader-addr`
(env `TRIP2G_LEADER_ADDR`) set to the leader's internal `host:port`. That single flag:

- **Skips migrations** and opens the DB **strict read-only** (SQLite `query_only`
  pragma → a stray write errors instead of corrupting). `internal/db/setup.go`:
  `SkipMigrations` + `StrictReadOnly` on `SetupConfig`.
- **Starts no writer subsystems** — Block B in `cmd/server/main.go` (writer-slot
  acquire, owner creation, cron, patreon/boosty, queue runners) is skipped entirely.
  It reuses the existing zero-downtime Block A (read-only warmup) / Block B (writer)
  split.
- **Forwards writes by HTTP method.** A middleware prepended to the request pipeline
  forwards every non-GET (POST/PUT/PATCH/DELETE — GraphQL mutations, webhooks, sync,
  auth callbacks) to the leader and relays the response verbatim. The decision is made
  on the method *before any handler runs*, so no handler side effect (charge, email,
  Telegram message) is ever replayed. `internal/readreplica/`.

### Why method-based forwarding (not GraphQL parsing or post-hoc write detection)

- **Forward only `/_system/graphql`** was rejected: it misses webhooks, obsidian sync,
  auth callbacks — all writes on other POST endpoints.
- **Run locally, detect a write, replay on the leader** was rejected: a handler may run
  side effects (Stripe charge, Telegram send) *before* its first SQL write, so replaying
  it on the leader double-fires them. HTTP handlers are not pure functions.
- **Method-based** is decided up front, catches every write path, and needs no GraphQL
  parsing. The cost — GraphQL *read* queries are POST and also go to the leader — is
  acceptable: the admin/system GraphQL API is low-volume and gets the freshest data,
  while the high-volume public GET path stays local.

The DB-level `query_only` guardrail is kept as a **safety net** (catch a stray local
write → error, never corrupt), not as the routing mechanism.

## Security: X-Replica-Auth + private-NIC intake

Each forwarded request carries `X-Replica-Auth` — an HMAC-SHA256 over the shared
`--jwt-secret` with a 30 s TTL (`internal/readreplica/readreplica.go`,
`SignAuth`/`VerifyAuth`). The leader accepts forwarded writes only on its **internal
listener**, which must be bound to the private NIC.

The leader's internal server was **converted from `net/http` to `fasthttp`** so it can
run the full app pipeline for forwarded writes on the same port that already serves
health/metrics/pprof — no extra listener, no extra flag (reuses
`--internal-listen-addr`). Health/metrics/pprof keep their `net/http` handlers, adapted
via `fasthttpadaptor`; any other path is treated as a replica write: verify
`X-Replica-Auth` → run the real app handler, else `401`. The handler is published into
an `atomic.Pointer` once `startServer` has built it (returns `503` until then).

This is why `--leader-addr` is a `host:port` and not a URL: the intake is the leader's
internal port over the private network, always plain HTTP — the protocol is fixed by us,
not configured.

## Database replication: LiteFS, not Litestream

**LiteFS** (FUSE filesystem, static lease) is the live replication layer: it streams
every WAL page from the primary to the replica. The replica's `/litefs` mount is
read-only at the filesystem level too — belt-and-suspenders with `query_only`.

**Litestream VFS is not a CLI read replica.** Testing litestream v0.5.12: the VFS ships
as `litestream.so`, a SQLite extension loaded via `load_extension()` from application
code — not a standalone tool you point at S3. The litestream user docs were corrected
accordingly. Litestream remains valid as continuous *backup*, but it targets the same
WAL as LiteFS, so run it on a standalone (non-LiteFS) primary.

## Config files

All deploy artifacts live in `infra/readreplica/` and are reproduced in [[read-replica]]:

| File | Role |
|------|------|
| `litefs.primary.yml` | primary LiteFS (`candidate: true`, `http.addr ":20202"`, advertise on private IP) |
| `litefs.replica.yml` | replica LiteFS (`candidate: false`, `advertise-url` → primary) |
| `litefs.service` | systemd, identical on both nodes |
| `trip2g.primary.env` | leader env (internal addr on private NIC = write intake) |
| `trip2g.replica.env` | replica env (`TRIP2G_LEADER_ADDR` set) |
| `trip2g.service` | systemd, identical on both nodes |

### Gotchas hit during deployment (all now documented)

- **`TRIP2G_DB_FILE`**, not `TRIP2G_DATABASE_FILE` — the flag is `--db-file`. The wrong
  name is silently ignored and the node falls back to a local default DB (the replica
  then opens an un-migrated DB → `no such table`).
- **`WorkingDirectory` must not be `/litefs`** — the FUSE mount only accepts SQLite
  files; gitapi creates a relative `tmp/` dir there and panics (`operation not
  permitted`). Use a normal dir (`/var/lib/trip2g`); only `TRIP2G_DB_FILE` points into
  `/litefs`.
- **LiteFS `http.addr: ":20202"`** is required — the default binds localhost only, so the
  replica can't reach the primary's API. Bind all interfaces; the firewall keeps it
  private.
- **`data-encryption-key` must be exactly 32 bytes.**
- Leader and replica must share `TRIP2G_JWT_SECRET` and `TRIP2G_DATA_ENCRYPTION_KEY`.

## Build & ship

Binary is built locally and scp'd — never built on the server:

```sh
make build-amd64            # GOOS=linux GOARCH=amd64 CGO_ENABLED=0 → ./tmp/amd64
scp ./tmp/amd64 root@<node>:/usr/local/bin/trip2g
```

## Test stand

Two Hetzner VMs (cx23, Ubuntu 22.04, nbg1), private network `10.20.0.0/16`:

| Node | Private IP | Public IP | Role |
|------|-----------|-----------|------|
| `t2g-rr-primary` | `10.20.0.2` | `46.224.223.64` | leader + MinIO (`:9000`) + LiteFS primary |
| `t2g-rr-replica` | `10.20.0.3` | `178.104.71.86` | read replica + LiteFS replica |

Public port `:8081`, internal/intake `:8082`, LiteFS API `:20202`. Firewall: inbound
SSH + ICMP from anywhere, all TCP/UDP within the private network.

## Validation results

| Check | Method | Result |
|-------|--------|--------|
| Replica boots read-only | journal | `read-only replica mode: skipping writer subsystems, forwarding writes to leader 10.20.0.2:8082` |
| Read parity | `GET /` on both | both `200`, identical `43979` bytes |
| Write forwarding | `POST {__typename}` to replica | `{"data":{"__typename":"Query"}}` (executed on leader, relayed) |
| Auth enforced | same POST to leader intake, no header | `401` |
| Read-only guardrail | direct write to replica `/litefs` DB | `disk I/O error` (read-only mount) |
| Zero-downtime | restart leader, loop replica reads | **60/60 reads, 0 failures** |
| Post-restart recovery | re-run forward after restart | leader `readyz 200`, forward returns data again |

### Replication lag

Measured with the replica polling **before** each write (so only real replication delay
counts), each row carrying the leader's nanosecond timestamp, NTP-synced clocks, 12
writes:

```
min=6  median=9  max=12  avg=8   (ms)
```

Single-digit-millisecond lag on the Hetzner private network. (Includes ~poll
granularity, so actual lag is marginally lower.) LiteFS also exposes live lag in
`/litefs/.lag`.

## Known limitations

- **Static lease, no failover.** `candidate: false` means the replica never promotes. If
  the primary is down, the replica keeps serving reads but write forwarding fails until
  the primary returns.
- **Replication lag window.** A write forwarded by the replica is visible in the
  replica's local read only after replication (single-digit ms here). For read-after-write
  on the same node there's a brief stale window — fine for the public read path.
- **GraphQL reads go to the leader** (POST). Low-volume; keeps admin data fresh.

## Code map

- `internal/readreplica/` — `IsWrite`, `SignAuth`/`VerifyAuth` (pure, unit-tested),
  `Forwarder` (fasthttp HostClient, preserves Host for multidomain routing).
- `internal/db/setup.go` — `SkipMigrations`, `StrictReadOnly` (`query_only`),
  `openConnection(config, queryLog)`.
- `cmd/server/main.go` — `--leader-addr` wiring, replica branch in startup, `DBSet`,
  forward middleware, fasthttp internal server + `handleReplicaIntake`.
- `internal/appconfig/config.go` — `LeaderAddr`, `IsReadReplica()`.

Tests: `internal/readreplica/*_test.go` (routing, sign/verify, end-to-end forward→intake
over an in-memory listener), `internal/db/setup_test.go` (`TestStrictReadOnlyRejectsWrites`).
