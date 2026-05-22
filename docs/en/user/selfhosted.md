---
title: "Self-hosted"
free: true
lang_redirect: "[[ru/user/selfhosted]]"
---

Minimal self-hosted setup for trip2g: `ghcr.io/trip2g/trip2g:latest` + `MinIO` + `Caddy`.

[★ GitHub — github.com/trip2g/trip2g](https://github.com/trip2g/trip2g)

This setup is for a single server, a single `docker-compose.yml`, and a straightforward `docker compose up -d`.

## What this setup runs

- `trip2g` is a platform for publishing Obsidian notes as a website.
- `minio` stores uploaded files and database backups.
- `caddy` handles incoming HTTP/HTTPS traffic and proxies it into the compose network.
- Vector search can stay disabled, or you can enable it later with OpenAI or another OpenAI-compatible embeddings API.

## Easy-to-miss requirements

- A public server should use `HTTPS`. Otherwise secure auth cookies will not work correctly.
- Email sign-in needs a `resend.com` account, an API key, and a verified sender domain or subdomain.
- In production you must set your own `JWT_SECRET` and `DATA_ENCRYPTION_KEY`.
- In practice, only `80` and `443` should be exposed externally by `caddy`.

## Prerequisites

You need:

- a Linux server with Docker and the Docker Compose plugin
- a site domain such as `docs.example.com`
- a sender subdomain such as `mg.example.com`
- DNS access

Create a directory such as `/opt/trip2g` and put two files there: `docker-compose.yml` and `.env`.

### Check your server before setup

If the server is not fresh, verify the following before starting:

- Ports `80` and `443` are free — compose hands them to Caddy:
  ```bash
  ss -tlnp | grep -E ':80 |:443 '
  ```
- No other Caddy, Nginx, or Traefik is already listening on those ports. If there is, stop it or move it to a different port.
- No conflicting Docker networks from other projects (rare, but can happen with non-default bridge or overlay setups).

If those ports are already claimed by an existing reverse proxy (Nginx, Caddy, Traefik), there is no need to remove it — just add trip2g as an upstream in your existing config, drop the `caddy` service from `docker-compose.yml`, and publish port `8081` directly. The same applies to MinIO: if you already have your own object storage, skip the `minio` service entirely and point `.env` at your existing bucket.

## `docker-compose.yml`

```yaml
services:
  caddy:
    image: caddy:2
    restart: unless-stopped
    depends_on:
      trip2g:
        condition: service_started
      minio:
        condition: service_healthy
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile:ro
      - caddy-data:/data
      - caddy-config:/config

  minio:
    image: minio/minio:latest
    restart: unless-stopped
    command: server /data --console-address ":9001"
    environment:
      MINIO_ROOT_USER: ${MINIO_ROOT_USER}
      MINIO_ROOT_PASSWORD: ${MINIO_ROOT_PASSWORD}
    volumes:
      - minio-data:/data
    expose:
      - "9000"
      - "9001"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:9000/minio/health/live"]
      interval: 5s
      timeout: 5s
      retries: 20

  trip2g:
    image: ghcr.io/trip2g/trip2g:latest
    restart: unless-stopped
    depends_on:
      minio:
        condition: service_healthy
    env_file:
      - .env
    volumes:
      - trip2g-data:/data
    expose:
      - "8081"
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8082/healthz"]
      interval: 10s
      timeout: 5s
      retries: 12
      start_period: 15s

volumes:
  caddy-data:
  caddy-config:
  trip2g-data:
  minio-data:
```

Why these choices matter:

- `caddy` is the only service exposing `80` and `443`.
- `trip2g-data` keeps trip2g data and the internal bare git repository.
- `minio-data` keeps MinIO objects.
- `trip2g` and `minio` stay internal to the compose network.

## `.env`

Minimal production `.env`:

```dotenv
PUBLIC_URL=https://docs.example.com
LISTEN_ADDR=0.0.0.0:8081
INTERNAL_LISTEN_ADDR=:8082
DB_FILE=/data/data.sqlite3
GIT_API_REPO_PATH=/data/git
LOG_LEVEL=info
DEV=false

OWNER_EMAIL=owner@example.com
MAIL_FROM=no-reply@mg.example.com
RESEND_API_KEY=re_xxxxxxxxx

JWT_SECRET=replace-with-long-random-secret
DATA_ENCRYPTION_KEY=0123456789abcdef0123456789abcdef

MINIO_ROOT_USER=trip2g
MINIO_ROOT_PASSWORD=replace-with-long-random-password
MINIO_ENDPOINT=minio:9000
MINIO_PUBLIC_URL=https://files.example.com
MINIO_ACCESS_KEY_ID=trip2g
MINIO_SECRET_KEY=replace-with-long-random-password
MINIO_BUCKET=trip2g
MINIO_REGION=us-east-1
MINIO_USE_SSL=false
MINIO_INIT_TIMEOUT=30s
MINIO_URL_EXPIRES_IN=10m

SIMPLE_BACKUP=true

FEATURES={}

# OpenAI embeddings:
# OPENAI_API_KEY=sk-...
# FEATURES={"vector_search":{"enabled":true,"model":"text-embedding-3-small"}}

# OpenAI-compatible embeddings API:
# OPENAI_API_KEY=provider-token-if-needed
# FEATURES={"vector_search":{"enabled":true,"model":"bge-m3","base_url":"https://embeddings.example.com/v1"}}
```

## Using a `TRIP2G_` prefix

By default, trip2g reads configuration from plain env vars like `LISTEN_ADDR` or `JWT_SECRET`.

If you run trip2g alongside other services that share the same environment (e.g. a single `.env` file for multiple containers), you can prefix every trip2g var with `TRIP2G_` to avoid name collisions:

```dotenv
TRIP2G_PUBLIC_URL=https://docs.example.com
TRIP2G_LISTEN_ADDR=0.0.0.0:8081
TRIP2G_JWT_SECRET=replace-with-long-random-secret
# … and so on for every setting
```

When a `TRIP2G_` var is present it takes precedence over the plain counterpart. Any `TRIP2G_` var that does not map to a known setting is logged as a warning on startup — useful for catching typos without crashing the server.

## What each setting does

- `PUBLIC_URL` is the external URL of your site. trip2g uses it for links, email flows, and integrations.
- `LISTEN_ADDR` is the main HTTP listen address.
- `INTERNAL_LISTEN_ADDR` is the internal address used for health checks and service endpoints.
- `DB_FILE` is the data file path inside the container.
- `GIT_API_REPO_PATH` is the path to trip2g's built-in git repository.
- `LOG_LEVEL` sets server log verbosity.
- `DEV=false` enables production behavior.
- `OWNER_EMAIL` is the owner account email.
- `MAIL_FROM` is the sender address. It must belong to a domain verified in Resend.
- `RESEND_API_KEY` is used to send email sign-in codes.
- `JWT_SECRET` signs user session tokens. Rotating it invalidates existing sessions.
- `DATA_ENCRYPTION_KEY` is a 32-byte key used to encrypt sensitive stored data. To generate one:
  ```bash
  openssl rand -base64 32 | head -c 32
  ```
- `MINIO_ROOT_USER` / `MINIO_ROOT_PASSWORD` are the MinIO root credentials.
- `MINIO_ENDPOINT` is the MinIO address as seen from the `trip2g` container.
- `MINIO_PUBLIC_URL` is the public MinIO hostname used in presigned file URLs.
- `MINIO_ACCESS_KEY_ID` / `MINIO_SECRET_KEY` are the credentials trip2g uses for MinIO.
- `MINIO_BUCKET` is the S3 bucket for assets and backup objects.
- `MINIO_REGION` is the S3 region string. For MinIO, `us-east-1` is fine.
- `MINIO_USE_SSL=false` is normal for container-to-container traffic on one host.
- `MINIO_INIT_TIMEOUT` controls how long startup waits for MinIO.
- `MINIO_URL_EXPIRES_IN` controls presigned URL lifetime for file downloads.
- `SIMPLE_BACKUP=true` enables simple SQLite backups to MinIO.
- `FEATURES` is the JSON feature-flag config. Vector search lives here.
- `OPENAI_API_KEY` is the key used for OpenAI or a compatible embeddings provider.

Optional HTTP-only smoke test:

```dotenv
PUBLIC_URL=http://SERVER_IP:8081
USER_TOKEN_INSECURE=true
```

Use that only for temporary testing. For a public instance, keep secure cookies and use TLS.

## `Caddyfile`

For a clean public setup, give the site and file storage separate hostnames:

- `docs.example.com` → trip2g
- `files.example.com` → MinIO

Create a `Caddyfile` next to `docker-compose.yml`:

```caddyfile
docs.example.com {
	encode zstd gzip
	reverse_proxy trip2g:8081
}

files.example.com {
	encode zstd gzip
	reverse_proxy minio:9000
}

# Optional, if you want the MinIO console externally:
# minio-admin.example.com {
# 	reverse_proxy minio:9001
# }
```

With that setup, these values in `.env` should match:

```dotenv
PUBLIC_URL=https://docs.example.com
MINIO_PUBLIC_URL=https://files.example.com
```

Why this matters:

- trip2g stays on the main site hostname;
- file URLs use a public MinIO hostname;
- `caddy` reaches `trip2g` and `minio` by service name inside the docker network.

## External object storage instead of MinIO

By default MinIO runs on the same server as trip2g. That is convenient to start but offers no protection against server loss: if the disk dies, files and backups go with it.

For production, we recommend moving storage to a separate S3-compatible service: Backblaze B2, Hetzner Object Storage, Timeweb S3, or similar.

In that case you can remove the `minio` service from `docker-compose.yml` entirely and point `.env` at the external service:

```dotenv
MINIO_ENDPOINT=s3.us-east-005.backblazeb2.com
MINIO_PUBLIC_URL=https://files.example.com
MINIO_ACCESS_KEY_ID=your-key-id
MINIO_SECRET_KEY=your-secret
MINIO_BUCKET=trip2g
MINIO_REGION=us-east-005
MINIO_USE_SSL=true
```

With that in place, `SIMPLE_BACKUP=true` stores SQLite backups on the external service automatically — no extra work, and protected from server-level failure.

## SQLite replication with Litestream

`SIMPLE_BACKUP=true` takes periodic SQLite snapshots to MinIO. For continuous replication with a one-second interval, add [Litestream](https://litestream.io).

Litestream runs on the host as a systemd service and streams the database file directly to any S3-compatible target. The `infra/` directory already has a ready-made setup:

- `infra/generate-litestream-config.sh` — generates `/etc/litestream.yml` from environment variables
- `infra/litestream.service` — the systemd unit

The config reads the same variables as trip2g's `.env`: `MINIO_ACCESS_KEY_ID`, `MINIO_SECRET_KEY`, `MINIO_ENDPOINT`, `MINIO_BUCKET`, `DB_FILE`. After installing litestream:

```bash
sudo cp infra/generate-litestream-config.sh /usr/local/bin/generate-litestream-config.sh
sudo chmod +x /usr/local/bin/generate-litestream-config.sh
sudo cp infra/litestream.service /etc/systemd/system/litestream.service
sudo systemctl enable --now litestream
```

Litestream and `SIMPLE_BACKUP` can run together — they do not conflict. The combination is especially useful with external object storage: both files and the database then live outside the server.

## Create a free Resend account

On `resend.com`:

1. Create a free account.
2. Add a sending domain, preferably a subdomain such as `mg.example.com`.
3. Add the DNS records Resend asks for.
4. Create an API key.
5. Put that key into `RESEND_API_KEY`.
6. Set `MAIL_FROM` to an address inside the verified domain, for example `no-reply@mg.example.com`.

Why a subdomain is better:

- it isolates sender reputation;
- it keeps trip2g transactional mail separate from your main domain mail.

If you do not verify a sender domain in Resend, email delivery is effectively just for your own address. That is enough if only the owner signs in by email. If other users need email login, verify the sender domain.

## Enable vector search with OpenAI or another compatible service

trip2g works fine without vector search. Full-text search still works.

If you want semantic search:

### OpenAI

```dotenv
OPENAI_API_KEY=sk-...
FEATURES={"vector_search":{"enabled":true,"model":"text-embedding-3-small"}}
```

Recommended starting model: `text-embedding-3-small`.

### OpenAI-compatible embeddings API

```dotenv
OPENAI_API_KEY=provider-token-if-needed
FEATURES={"vector_search":{"enabled":true,"model":"bge-m3","base_url":"https://embeddings.example.com/v1"}}
```

Important: trip2g validates the embedding model name. The currently supported values are:

- `text-embedding-3-small`
- `text-embedding-3-large`
- `text-embedding-ada-002`
- `multilingual-e5-base`
- `bge-m3`

So “any OpenAI-compatible service” only works if it exposes a compatible `/v1` embeddings API and you configure one of those supported model names.

## Start the stack

Inside the directory with `docker-compose.yml`:

```bash
docker compose up -d
```

If you still use the old syntax:

```bash
docker-compose up -d
```

Check the result:

```bash
docker compose ps
docker compose logs -f caddy trip2g
```

After startup:

- open `https://docs.example.com`
- sign in with the owner email from `OWNER_EMAIL`
- on an empty instance, the service itself will offer a link to download a preconfigured vault ZIP
- configure the Obsidian plugin with your `PUBLIC_URL`
From there, continue with [[en/user/getting-started|Getting started]].

## Things people often forget

- `A` / `AAAA` DNS record for `PUBLIC_URL`
- `A` / `AAAA` DNS record for `MINIO_PUBLIC_URL`
- TLS certificate for the site domain
- Resend DNS records for the sender domain
- persistent storage for `trip2g-data`
- do not publish the MinIO console on `9001` unless you actually need it
- checking logs after the first login and the first outbound email

For the smoothest rollout, start without vector search, verify email sign-in first, and only then enable embeddings in `FEATURES`.
