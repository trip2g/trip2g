---
title: Local quickstart — run trip2g and push content
description: Run a trip2g instance locally from the Docker image (or a binary) and publish a content vault over the API in a few minutes.
---

# Local quickstart

Run a trip2g instance on your machine and push a content vault to it over the API. No Git remote, no `GIT_API_REPO_PATH` — just the server plus the sync CLI.

You need: the `trip2g` Docker image (or a locally built binary) and a MinIO (S3-compatible) endpoint for assets.

## 1. Run the server

trip2g needs an S3-compatible store (MinIO) for assets. Start one, then the app.

```bash
# MinIO (skip if you already have one)
docker run -d --name trip2g-minio -p 9000:9000 -p 9001:9001 \
  -e MINIO_ROOT_USER=trip2g -e MINIO_ROOT_PASSWORD=trip2g-secret \
  minio/minio:latest server /data --console-address ":9001"

# trip2g app on port 24081 (health on 24082), fresh local DB
# This pulls the current published image. To build locally from source instead,
# run `docker build -t trip2g:local .` from a trip2g checkout and replace the
# image tag below with `trip2g:local`.
mkdir -p /tmp/trip2g-local
docker run -d --name trip2g-local --network host \
  -e LISTEN_ADDR=0.0.0.0:24081 -e INTERNAL_LISTEN_ADDR=:24082 \
  -e DB_FILE=/data/local.sqlite3 \
  -e DEV=true \
  -e OWNER_EMAIL=hello@example.com \
  -e MINIO_ENDPOINT=localhost:9000 \
  -e MINIO_ACCESS_KEY_ID=trip2g -e MINIO_SECRET_KEY=trip2g-secret \
  -e MINIO_BUCKET=trip2g-local -e MINIO_USE_SSL=false \
  -e PUBLIC_URL=http://localhost:24081 \
  -e JWT_SECRET=dev-secret-not-for-prod \
  -e USER_TOKEN_INSECURE=true \
  -e GIT_API_REPO_PATH=/data/git -e GIT_API_BASE_PATH=/git \
  -e RESEND_API_KEY=dev -e MAIL_FROM=dev@example.com \
  -v /tmp/trip2g-local:/data \
  ghcr.io/trip2g/trip2g:latest

# wait until healthy — also verify the container is actually running
docker ps | grep trip2g-local
until curl -sf http://localhost:24082/healthz >/dev/null; do sleep 1; done; echo "up"
```

Notes:
- `DEV=true` enables the dev sign-in code (no real email needed) — **never use in production**.
- You do **not** need the `FEATURES` / vector-search env for plain content; omit it to avoid the embedding-server dependency.
- `--network host` lets the app reach MinIO on `localhost:9000` and exposes the app port directly.

Building from source instead of the image: `make build` produces `./tmp/server`; run it with the same env vars (and a reachable MinIO).

## 2. Mint an API key (dev flow)

In `DEV=true` the server accepts a fixed sign-in code (`111111`, `000000` also works). Sign in as the owner, then create a key:

```bash
GQL=http://localhost:24081/graphql

curl -s -X POST "$GQL" -H 'Content-Type: application/json' \
  -d '{"query":"mutation($i:RequestEmailSignInCodeInput!){requestEmailSignInCode(input:$i){__typename}}","variables":{"i":{"email":"hello@example.com"}}}' >/dev/null

TOKEN=$(curl -s -X POST "$GQL" -H 'Content-Type: application/json' \
  -d '{"query":"mutation($i:SignInByEmailInput!){signInByEmail(input:$i){__typename ... on SignInPayload{token} ... on ErrorPayload{message}}}","variables":{"i":{"email":"hello@example.com","code":"111111"}}}' \
  | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

API_KEY=$(curl -s -X POST "$GQL" -H 'Content-Type: application/json' -H "Cookie: trip2g_token=$TOKEN" \
  -d '{"query":"mutation($i:CreateApiKeyInput!){admin{createApiKey(input:$i){__typename ... on CreateApiKeyPayload{value} ... on ErrorPayload{message}}}}","variables":{"i":{"description":"local"}}}' \
  | grep -o '"value":"[^"]*"' | cut -d'"' -f4)

echo "API key: $API_KEY"
```

(If you set a custom `USER_TOKEN_COOKIE_NAME`, use that cookie name instead of `trip2g_token`.)

## 3. Push your content

Use the sync CLI to publish a folder of notes (`.md`) plus any `_layouts/` templates and assets. The CLI (`obsidian-sync/dist/trip2g-sync.mjs`) lives in the trip2g source repository — run the following from the root of a trip2g checkout:

```bash
node obsidian-sync/dist/trip2g-sync.mjs \
  --folder /path/to/your/vault \
  --api-key "$API_KEY" \
  --api-url http://localhost:24081/graphql \
  --verbose
```

## 4. View it

Open the note's permalink, e.g. `http://localhost:24081/<path>/<note>`.

A note with `route: yourdomain.com/` in frontmatter is served on that custom domain (needs a `Host:` header or DNS locally). To preview without DNS, hit the plain permalink instead.

Using this instance as AI-agent memory? See [[en/user/agent-memory]].

## Gotchas

- **Dev sign-in code** is `111111` (also `000000`). Only with `DEV=true`.
- **`route:` frontmatter** maps a note to a custom domain. Locally, view via the plain permalink (`/path/note`) or send a `Host:` header.
- **Custom Jet layouts**: a note with `layout: <theme>/<page>` renders through `_layouts/<theme>/<page>.html`. If the template fails to parse, the server silently falls back to the default layout (HTTP 200) — so the page "works" but looks wrong.
- **`{{/* ... */}}` block comments break the template engine** in current builds — don't put them in `_layouts/*.html`.
- A note is public (served to anonymous visitors) only with `free: true` in frontmatter, or a `**/*.md → { free: true }` frontmatter patch.
