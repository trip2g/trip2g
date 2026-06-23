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

Litestream ships a **[VFS](https://fly.io/blog/litestream-vfs/)** — a SQLite virtual filesystem extension (`litestream.so`) that an application loads via `load_extension()` to query the S3 backup stream directly from code. It is not a standalone CLI feature or a separate read-replica server. Using it requires application integration; it is not something you point at an S3 bucket from the command line. Per-page S3 fetches also add latency, so it is not suitable for a hot read path.

For a validated, production-ready live read replica of trip2g — reads served locally with sub-second replication lag, writes forwarded to the primary, zero read failures during primary restarts — see [[read-replica]]. That setup uses **LiteFS** ([docs](https://fly.io/docs/litefs/)) as the replication layer: a FUSE filesystem that streams SQLite changes across machines. It pairs naturally with trip2g's single-writer model and with [[zerodowntime|zero-downtime deploys]].

## Which backup should I use?

See [[backup]] for the full comparison. Short version: **simple backup** (built-in S3 snapshots) is the zero-setup default; **Litestream** is the upgrade when you want continuous replication and minimal data loss.

If you also run a live [[read-replica|read replica]] using LiteFS, note that Litestream and LiteFS both target the SQLite WAL. Run Litestream backup on a standalone primary that is not mounted by LiteFS — not on the LiteFS FUSE mount itself.
