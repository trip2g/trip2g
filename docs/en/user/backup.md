---
title: "Backups"
free: true
lang_redirect: "[[ru/user/backup]]"
---

trip2g keeps all of your content in a single SQLite database. There are two ways to back it up — pick one.

## Option 1: Simple backup (built-in)

Zero setup beyond S3 credentials. trip2g takes **periodic full snapshots** of the database, gzips them, and uploads to S3-compatible storage (AWS S3, MinIO, Backblaze, …).

```sh
trip2g --simple-backup ...
```

- **Snapshots** run on a schedule (hourly) with retention (oldest pruned).
- **Restore on startup**: if the server boots with **no local database**, it pulls the latest snapshot from S3 first. On a server that already has its database, it does nothing — your data is never overwritten by a restore.
- **Shutdown backup**: by default trip2g also takes one last snapshot on graceful shutdown, so a planned stop captures the final state. Control it with `--simple-backup-on-shutdown` (default `true`).

For a [[zerodowntime|zero-downtime rolling deploy]], set `--simple-backup-on-shutdown=false` on the departing instance: a replacement is already taking over, hourly snapshots continue, and a shutdown dump would only race the new writer and slow the drain.

**Trade-off:** between snapshots you can lose up to an hour of writes.

## Option 2: Litestream (continuous)

[[litestream|Litestream]] streams **every write** to S3 as it happens, so a crash loses seconds, not an hour — and it's the gateway to read replicas. It runs as a sidecar; see the [[litestream]] guide for setup.

When you use Litestream, turn trip2g's own maintenance off so the two don't fight over the WAL:

```sh
trip2g --simple-backup=false --vacuum-cron=false ...
```

(`--vacuum-cron` is already off by default. Never enable it under Litestream — `VACUUM` and `wal_checkpoint(TRUNCATE)` corrupt a Litestream replica.)

## Which one?

| | Simple backup | Litestream |
|---|---|---|
| Setup | S3 creds only | sidecar process + config |
| Data loss window (RPO) | up to ~1 hour | seconds |
| Read replicas | — | yes ([[litestream|VFS / LiteFS]]) |
| Best for | most self-hosters | low-RPO / scaling needs |

**Do not run both** on the same database — Litestream is your backup when it's enabled. Start with simple backup; move to Litestream when you need a tighter recovery window or read replicas.
