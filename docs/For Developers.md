---
home_position: 9998
---

trip2g ships as a single Go binary with an embedded SQLite database. You run it on your own server, point it at an Obsidian vault via the API, and it serves a full publishing platform — website, paywalls, Telegram publishing, and AI search — with no external services required except MinIO for file storage.

## Simple to deploy

One binary, one database file. The binary embeds all assets and migrations; SQLite runs in WAL mode and handles concurrent reads without configuration. Litestream replicates the database to S3 in the background. To upgrade, replace the binary and restart — migrations run on startup automatically.
