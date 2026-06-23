---
title: "Live read replica with LiteFS"
free: true
lang_redirect: "[[ru/user/read-replica]]"
---

**TL;DR:** Set `TRIP2G_LEADER_ADDR` on a second server. That server serves all GET requests locally from a LiteFS-replicated SQLite file and forwards every mutating request to the leader. Reads stay fast; writes are always consistent. Tested on two Hetzner VMs; zero read failures during leader restart.

---

A read replica lets you split traffic: a second trip2g instance handles all public reads locally while the primary handles all writes. This lowers read latency, removes read load from the primary, and lets you restart the primary without interrupting readers. See also [[zerodowntime]] and [[litestream]].

## How it works

Three moving parts work together:

**LiteFS** streams every SQLite write from the primary to the replica at the filesystem level. The replica's `/litefs` mount is read-only; every WAL page the primary commits appears on the replica within single-digit milliseconds (median 9 ms in our test). The app on the replica reads from this local copy — no S3 round-trip, no network hop to the primary.

**Read-only mode** (`TRIP2G_LEADER_ADDR` set) changes trip2g's behavior at startup: it skips migrations, opens the SQLite file with SQLite's `query_only` pragma (stray writes error immediately instead of corrupting data), and starts no background writer, queue, or cron subsystems.

**Write forwarding** — when the replica receives a mutating request (POST, PUT, PATCH, DELETE), it forwards the full request to the leader's internal HTTP address, waits for the response, and returns it verbatim to the caller. The decision is made by HTTP method before any handler runs, so no side effect is ever executed twice.

Security: each forwarded request carries an `X-Replica-Auth` header — an HMAC signed with the shared `TRIP2G_JWT_SECRET`, valid for 30 seconds. The leader accepts forwarded writes only on its internal port (`TRIP2G_INTERNAL_LISTEN_ADDR`), which must be bound to the private NIC. Requests without a valid `X-Replica-Auth` get a 401.

> **Note on GraphQL:** GraphQL queries are HTTP POST, so they also go to the leader. This gives you always-fresh data for the low-volume admin API. The high-volume public page path is HTTP GET, served locally on the replica.

## What you need

- Two servers on the same private network (the examples use Hetzner private networking: primary `10.20.0.2`, replica `10.20.0.3`)
- `litefs` binary on both servers (`/usr/local/bin/litefs`)
- `trip2g` binary on both servers

Note: Litestream (continuous S3 backup) and LiteFS (live replication) are separate tools targeting the same WAL. If you run both, run Litestream on a standalone primary that is not on the LiteFS mount. See [[litestream]] and [[backup]].

## Step 1 — Install LiteFS

Place the `litefs` binary at `/usr/local/bin/litefs` on both servers.

Create the directories:

```sh
mkdir -p /litefs /var/lib/litefs
```

`/litefs` is the FUSE mount the app opens. `/var/lib/litefs` holds LiteFS internal state (LTX pages). Only SQLite database files may live under `/litefs`.

### Primary: `/etc/litefs.yml`

```yaml
# litefs.primary.yml — save as /etc/litefs.yml on the primary

fuse:
  # Where the application sees the database. trip2g opens /litefs/data.sqlite3.
  # On the primary this mount is read-write.
  dir: "/litefs"

data:
  # LiteFS internal state. Put this on a persistent volume.
  dir: "/var/lib/litefs"

# Bind the LiteFS API on all interfaces (default is localhost only) so the
# replica can reach it. The firewall keeps :20202 off the public internet.
http:
  addr: ":20202"

lease:
  type: "static"
  # This node's LiteFS API URL. Replicas connect here to stream.
  # Use the private NIC so replication never crosses the public network.
  advertise-url: "http://10.20.0.2:20202"
  # candidate: true means this node is the primary.
  candidate: true
```

### Replica: `/etc/litefs.yml`

```yaml
# litefs.replica.yml — save as /etc/litefs.yml on the replica

fuse:
  dir: "/litefs"

data:
  dir: "/var/lib/litefs"

lease:
  type: "static"
  # advertise-url points at the PRIMARY (not this node).
  # The replica streams the database from the primary's API.
  advertise-url: "http://10.20.0.2:20202"
  # candidate: false means this node is always a read-only replica.
  candidate: false
```

### systemd unit (identical on both nodes): `/etc/systemd/system/litefs.service`

```ini
[Unit]
Description=LiteFS distributed SQLite
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/litefs mount
ExecStopPost=/bin/fusermount -uz /litefs
Restart=on-failure
RestartSec=2
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
```

The only difference between nodes is `/etc/litefs.yml`. The systemd unit is identical.

## Step 2 — Configure trip2g

Build locally and ship the binary — do not build on the server:

```sh
make build-amd64
scp ./tmp/amd64 root@10.20.0.2:/usr/local/bin/trip2g
scp ./tmp/amd64 root@10.20.0.3:/usr/local/bin/trip2g
```

### Primary: `/etc/trip2g.env`

```sh
TRIP2G_DB_FILE=/litefs/data.sqlite3
TRIP2G_LISTEN_ADDR=:8081

# Internal port: health/metrics AND the leader-side write intake for replicas.
# Bind to the private NIC — never expose this to the public internet.
TRIP2G_INTERNAL_LISTEN_ADDR=10.20.0.2:8082

# Shared with the replica. Also signs X-Replica-Auth HMAC tokens.
TRIP2G_JWT_SECRET=your-shared-secret-here
TRIP2G_DATA_ENCRYPTION_KEY=0123456789abcdef0123456789abcdef  # exactly 32 bytes

TRIP2G_OWNER_EMAIL=admin@example.com
TRIP2G_PUBLIC_URL=https://yourdomain.com

TRIP2G_MINIO_ENDPOINT=10.20.0.2:9000
TRIP2G_MINIO_ACCESS_KEY_ID=your-key
TRIP2G_MINIO_SECRET_KEY=your-secret
TRIP2G_MINIO_BUCKET=trip2g-backups
```

Key points:
- The DB env var is `TRIP2G_DB_FILE` (flag `--db-file`), not `TRIP2G_DATABASE_FILE`.
- `TRIP2G_INTERNAL_LISTEN_ADDR` must be on the private NIC. This is the write intake the replica forwards to.
- `TRIP2G_DATA_ENCRYPTION_KEY` must be exactly 32 bytes.

### Replica: `/etc/trip2g.env`

```sh
TRIP2G_DB_FILE=/litefs/data.sqlite3
TRIP2G_LISTEN_ADDR=:8081
TRIP2G_INTERNAL_LISTEN_ADDR=:8082

# This single variable activates read-only replica mode.
# Point it at the leader's INTERNAL address (private NIC, plain HTTP).
TRIP2G_LEADER_ADDR=10.20.0.2:8082

# Must match the leader exactly — shared DB + X-Replica-Auth signing.
TRIP2G_JWT_SECRET=your-shared-secret-here
TRIP2G_DATA_ENCRYPTION_KEY=0123456789abcdef0123456789abcdef

TRIP2G_OWNER_EMAIL=admin@example.com
TRIP2G_PUBLIC_URL=https://replica.yourdomain.com

TRIP2G_MINIO_ENDPOINT=10.20.0.2:9000
TRIP2G_MINIO_ACCESS_KEY_ID=your-key
TRIP2G_MINIO_SECRET_KEY=your-secret
TRIP2G_MINIO_BUCKET=trip2g-backups
```

`TRIP2G_LEADER_ADDR` is the single switch. Non-empty → read-only mode.

### systemd unit (identical on both nodes): `/etc/systemd/system/trip2g.service`

```ini
[Unit]
Description=trip2g
After=litefs.service network-online.target
Requires=litefs.service
Wants=network-online.target

[Service]
Type=simple
EnvironmentFile=/etc/trip2g.env
# WorkingDirectory must be a normal writable directory.
# Never point it at /litefs — the FUSE mount only accepts SQLite files.
# gitapi creates a tmp/ subdirectory and will fail on the FUSE mount.
WorkingDirectory=/var/lib/trip2g
ExecStart=/usr/local/bin/trip2g
Restart=on-failure
RestartSec=2
KillSignal=SIGTERM
TimeoutStopSec=20
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
```

Create the working directory before starting:

```sh
mkdir -p /var/lib/trip2g
```

## Step 3 — Start in order

Start the primary first. trip2g on the primary runs migrations and creates the database. Then start the replica — LiteFS streams the database before trip2g starts.

```sh
# On the primary
systemctl enable --now litefs
systemctl enable --now trip2g

# Wait ~5 seconds for LiteFS to replicate the DB to the replica, then:

# On the replica
systemctl enable --now litefs
systemctl enable --now trip2g
```

The replica's trip2g waits for `/litefs/data.sqlite3` to exist (guaranteed by `Requires=litefs.service`) before it opens the database.

## Verification

All commands below were run against a two-Hetzner-VM test stand (primary `10.20.0.2`, replica `10.20.0.3`).

**Replica is alive:**

```sh
curl -s http://10.20.0.3:8082/livez
# 200 OK

curl -s -o /dev/null -w "%{http_code}" http://10.20.0.3:8081/
# 200
```

**Read parity — leader and replica return identical content:**

```sh
curl -s http://10.20.0.2:8081/ | wc -c   # 43979
curl -s http://10.20.0.3:8081/ | wc -c   # 43979
```

**Write forwarding works — POST to replica is executed on the leader and relayed back:**

```sh
curl -s -X POST \
  -H "Content-Type: application/json" \
  -d '{"query":"{__typename}"}' \
  http://10.20.0.3:8081/_system/graphql
# {"data":{"__typename":"Query"}}
```

**Auth enforced — direct POST to leader's internal port without the header gets 401:**

```sh
curl -s -o /dev/null -w "%{http_code}" \
  -X POST -d '{"query":"{__typename}"}' \
  http://10.20.0.2:8082/_system/graphql
# 401
```

**Replication lag — write on the leader, check the replica:**

```sh
# Write on the primary, watch it appear on the replica. Measured properly
# (replica polling before each write, NTP-synced clocks, 12 writes), the
# replication lag was: min 6 ms, median 9 ms, max 12 ms — single-digit
# milliseconds on a private network.
cat /litefs/.lag   # on the replica; live lag reported by LiteFS
```

**Read-only guardrail — direct write to the replica's FUSE mount fails:**

```sh
sqlite3 /litefs/data.sqlite3 "INSERT INTO notes(id) VALUES(1);"
# Error: disk I/O error (read-only filesystem)
```

**Zero-downtime — restart the leader while the replica serves reads:**

```sh
# Loop 60 GETs against the replica while restarting the leader:
for i in $(seq 1 60); do
  curl -s -o /dev/null -w "%{http_code}\n" http://10.20.0.3:8081/
done | sort | uniq -c
# 60 200
# 0 non-200
```

Sixty out of sixty reads succeeded. The replica never stopped serving. See [[zerodowntime]] for the full zero-downtime deploy flow.

## Tradeoffs

**Replication lag:** A write forwarded by the replica (login session, note publish) takes sub-second to appear in the replica's local read. If you immediately read back that data from the replica, you may see the pre-write state for a brief window. For the public read path this is acceptable. For transactional flows (login → immediate redirect), the leader handles both the write and the subsequent read anyway (the redirect goes through the leader's response).

**GraphQL mutations and queries:** GraphQL uses HTTP POST, so all GraphQL traffic — queries and mutations — goes to the leader. This keeps admin API data fresh at the cost of sending low-volume GraphQL reads over the internal network. The high-volume public content path (GET) stays entirely local on the replica.

**Static lease, no failover:** `candidate: false` means the replica never becomes primary. This is intentional for a simple two-node setup. If the primary is down, the replica continues serving reads but write forwarding fails until the primary is restored.

See also [[backup]], [[litestream]], [[zerodowntime]].
