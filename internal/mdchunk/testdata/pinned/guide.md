---
free: true
title: Deploy guide
---

This guide walks through a first deployment. It assumes a Linux host with
systemd and a domain that already points at the machine.

## Requirements

- A server with at least 1 GB of memory
- A domain name and access to its DNS records
- An SMTP account, if you want email sign-in codes

The binary is statically linked, so nothing else has to be installed. Backups
need an S3-compatible bucket, but you can add that later without downtime.

## Install {#install}

Download the release and unpack it into `/opt/app`:

```bash
curl -fsSL https://example.com/release.tar.gz -o release.tar.gz

tar -xzf release.tar.gz -C /opt/app

systemctl enable --now app.service
```

The service listens on port 8081 by default. Put a reverse proxy in front of it
to terminate TLS; any proxy that forwards the `Host` header works.

### Configuration

| Variable | Default | Meaning |
|---|---|---|
| `PORT` | `8081` | HTTP listen port |
| `DATA_DIR` | `./data` | Where the SQLite file lives |
| `PUBLIC_URL` | none | External address used in emails |

Restart the service after changing a variable. Values are read once at startup
and never reloaded, which keeps the running process predictable.

## Troubleshooting

If the page does not open, check the service log first: `journalctl -u app`.
Most failures at this stage are a wrong `PUBLIC_URL` or a port that the
firewall still blocks. Ask in the community chat when the log is not enough.
