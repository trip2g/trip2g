---
title: "Continuous backup & read replicas with Litestream"
free: true
lang_redirect: "[[ru/user/litestream]]"
---

[Litestream](https://litestream.io) replicates your SQLite database to S3 **continuously** — every write, streamed to object storage — instead of the periodic full snapshots trip2g's built-in [[backup|simple backup]] takes. If the server dies, you lose seconds, not an hour. It runs as a small **sidecar** next to trip2g; nothing is compiled in.

## Run it alongside trip2g

Litestream watches the WAL of your SQLite file and ships it to S3. Two trip2g flags make them coexist cleanly:

```sh
trip2g --vacuum-cron=false --simple-backup=false ...
```

- `--vacuum-cron=false` — **this is the important one** (and it is the default). trip2g's optional maintenance job runs `wal_checkpoint(TRUNCATE)` + `VACUUM`. Litestream **must be the only process that touches the WAL**: a `TRUNCATE` checkpoint can drop WAL frames Litestream hasn't replicated yet, and `VACUUM` rewrites the whole database into a new Litestream generation. So with Litestream, never enable `--vacuum-cron`. (Litestream does its own checkpointing; it does **not** vacuum, and you don't need to.)
- `--simple-backup=false` — don't run trip2g's S3 snapshot backup too. Litestream **is** your backup; running both is redundant.

A minimal Litestream config (`litestream.yml`):

```yaml
dbs:
  - path: /data/data.sqlite3
    replicas:
      - type: s3
        endpoint: https://your-s3-or-minio
        bucket: trip2g-backups
        path: data.sqlite3
```

Start `litestream replicate -config litestream.yml` next to trip2g. To restore on a fresh server: `litestream restore -config litestream.yml /data/data.sqlite3` **before** starting trip2g.

## Encryption

Litestream 0.3.13+ supports native client-side **age encryption** (end-to-end) — the simplest way to meet "backups encrypted at rest and in transit" for compliance. Add an `age` key to the replica config; the bucket only ever sees ciphertext.

## Beyond backup: read replicas

Litestream's newer **[VFS](https://fly.io/blog/litestream-vfs/)** can query the database *directly from the S3 backup* without downloading it, giving a near-real-time **read-only replica** — handy for "read production data without touching production" and point-in-time queries, though per-page S3 fetches add latency (not a hot read path).

For live, low-latency read replicas across machines (reads served locally, writes forwarded to a primary, automatic primary election), the sibling project is **[LiteFS](https://fly.io/docs/litefs/)** ([introduction](https://fly.io/blog/introducing-litefs/)) — a FUSE filesystem that replicates SQLite across a cluster. It pairs naturally with trip2g's single-writer model and with [[zerodowntime|zero-downtime deploys]] (replicas keep serving reads while the primary is replaced). It's the backbone of SQLite apps on [Fly.io](https://fly.io/docs/litefs/).

## Which backup should I use?

See [[backup]] for the full comparison. Short version: **simple backup** (built-in S3 snapshots) is the zero-setup default; **Litestream** is the upgrade when you want continuous replication / minimal data loss, and the prerequisite for the read-replica path above.
