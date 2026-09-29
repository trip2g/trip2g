#!/usr/bin/env bash
# Start a throwaway trip2g instance for one vault, push the vault, wait for embeddings.
#
# usage: trip2g_up.sh <lang> <port> <binary> [features-json]
#   binary        a dev build: go build -tags dev -o "$OHB_WORK/trip2g" ./cmd/server
#   features-json default: bge-m3 at $EMBED_URL (default http://localhost:11439/v1);
#                 pass '{}' for a text-only instance (used by agent_fields.py)
# Needs an S3 endpoint (MinIO/Silo) at $S3_ENDPOINT (default localhost:22010) with
# credentials bench/benchpass123, and obsidian-sync built (obsidian-sync/dist).
#
# Stop an instance with kill -9: a benchmark instance may not exit on SIGTERM.
set -euo pipefail
here=$(cd "$(dirname "$0")" && pwd); repo=$(cd "$here/../.." && pwd)
WORK=${OHB_WORK:-$repo/tmp/obsidian-help-bench}
lang=$1 port=$2 bin=$3
features=${4:-"{\"vector_search\": {\"enabled\": true, \"model\": \"bge-m3\", \"base_url\": \"${EMBED_URL:-http://localhost:11439/v1}\"}}"}
dir="$WORK/t2g-$lang-$port"; mkdir -p "$dir"
G="http://127.0.0.1:$port/_system/graphql"

(cd "$dir" && nohup env -i PATH="$PATH" HOME="$HOME" \
  LISTEN_ADDR="127.0.0.1:$port" INTERNAL_LISTEN_ADDR="127.0.0.1:$((port + 1))" \
  DB_FILE=bench.sqlite3 DEV=true LOG_LEVEL=info OWNER_EMAIL=hello@example.com \
  SHUTDOWN_GRACE_PERIOD=1ms SHUTDOWN_TIMEOUT=1ms PUBLIC_URL="http://localhost:$port" \
  JWT_SECRET=bench-secret-not-for-prod USER_TOKEN_COOKIE_NAME=trip2g_bench MAIL_FROM=bench@example.com \
  MINIO_ENDPOINT="${S3_ENDPOINT:-localhost:22010}" MINIO_ACCESS_KEY_ID=bench MINIO_SECRET_KEY=benchpass123 \
  MINIO_BUCKET="ohb-$lang-$port" MINIO_USE_SSL=false FEATURES="$features" \
  "$bin" > server.log 2>&1 &)
until curl -sf -X POST "$G" -H 'Content-Type: application/json' -d '{"query":"{ __typename }"}' >/dev/null; do sleep 2; done

curl -sf -X POST "$G" -H 'Content-Type: application/json' \
  -d '{"query":"mutation { requestEmailSignInCode(input: { email: \"hello@example.com\" }) { __typename } }"}' >/dev/null
token=$(curl -sf -X POST "$G" -H 'Content-Type: application/json' \
  -d '{"query":"mutation { signInByEmail(input: { email: \"hello@example.com\", code: \"111111\" }) { ... on SignInPayload { token } } }"}' \
  | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
echo "$token" > "$WORK/token-$port"
key=$(curl -sf -X POST "$G" -H 'Content-Type: application/json' -H "Cookie: trip2g_bench=$token" \
  -d '{"query":"mutation($input: CreateApiKeyInput!) { admin { createApiKey(input: $input) { ... on CreateApiKeyPayload { value } } } }","variables":{"input":{"description":"bench"}}}' \
  | grep -o '"value":"[^"]*"' | cut -d'"' -f4)

node "$repo/obsidian-sync/dist/trip2g-sync.mjs" -u "$G" -k "$key" "$WORK/vault-$lang" | grep -E "Pushed|❌" || true
until curl -s --max-time 30 "http://127.0.0.1:$port/debug/wait_all_jobs" | grep -q '^ok'; do sleep 30; done
echo "trip2g $lang up on :$port, embeddings done"
