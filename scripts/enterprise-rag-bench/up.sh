#!/usr/bin/env bash
cd "${ERB_WORK:?set ERB_WORK to the work directory}"
# usage: up.sh <port>; expects ./trip2g and ./.key in the current directory
set -euo pipefail
port=$1; G="http://127.0.0.1:$port/_system/graphql"
docker inspect erb-bench-s3 >/dev/null 2>&1 || docker run -d --name erb-bench-s3 -p 127.0.0.1:22010:22010 \
  -e MINIO_ROOT_USER=bench -e MINIO_ROOT_PASSWORD=benchpass123 -v $PWD/s3:/data \
  pgsty/silo:RELEASE.2026-10-18T00-00-00Z server /data --address :22010 >/dev/null
sleep 3
F='{"vector_search":{"enabled":true,"model":"baai/bge-m3","base_url":"https://openrouter.ai/api/v1","dimensions":1024,"max_input_tokens":8192,"embed_batch_size":32}}'
nohup nice -n 10 env -i PATH="$PATH" HOME="$HOME" SEARCH_INDEX_PATH="$PWD/searchindex" \
  LISTEN_ADDR="127.0.0.1:$port" INTERNAL_LISTEN_ADDR="127.0.0.1:$((port + 1))" \
  DB_FILE=bench.sqlite3 DEV=true LOG_LEVEL=warn OWNER_EMAIL=hello@example.com \
  SHUTDOWN_GRACE_PERIOD=1ms SHUTDOWN_TIMEOUT=1ms PUBLIC_URL="http://localhost:$port" \
  JWT_SECRET=bench-secret-not-for-prod USER_TOKEN_COOKIE_NAME=trip2g_bench MAIL_FROM=bench@example.com \
  MINIO_ENDPOINT=127.0.0.1:22010 MINIO_ACCESS_KEY_ID=bench MINIO_SECRET_KEY=benchpass123 \
  MINIO_BUCKET=erbbench MINIO_USE_SSL=false OPENAI_API_KEY="$(cat .key)" FEATURES="$F" \
  ./trip2g -embedding-reload-interval=5m -storage-db-limit=0 -global-queue-limit=20 -global-queue-poll-interval=500ms > server.log 2>&1 &
echo $! > server.pid
until curl -sf -m 5 -X POST "$G" -H 'Content-Type: application/json' -d '{"query":"{ __typename }"}' >/dev/null; do sleep 2; done
curl -sf -X POST "$G" -H 'Content-Type: application/json' -d '{"query":"mutation { requestEmailSignInCode(input: { email: \"hello@example.com\" }) { __typename } }"}' >/dev/null
token=$(curl -sf -X POST "$G" -H 'Content-Type: application/json' -d '{"query":"mutation { signInByEmail(input: { email: \"hello@example.com\", code: \"111111\" }) { ... on SignInPayload { token } } }"}' | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
echo "$token" > token
[ -s apikey ] || curl -sf -X POST "$G" -H 'Content-Type: application/json' -H "Cookie: trip2g_bench=$token" -d '{"query":"mutation($input: CreateApiKeyInput!) { admin { createApiKey(input: $input) { ... on CreateApiKeyPayload { value } } } }","variables":{"input":{"description":"bench"}}}' | grep -o '"value":"[^"]*"' | cut -d'"' -f4 > apikey
echo up
