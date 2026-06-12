#!/bin/bash
# pushNotes lock-time benchmark.
#
# For each core count and note count, starts an isolated server instance
# (fresh temp DB, no dev .env, pinned to CPUs via taskset) and runs
# scripts/bench-pushnotes.mjs against it.
#
# Usage: ./scripts/bench-pushnotes.sh [cores...]   # default: 1 4
#        VECTOR=1 ./scripts/bench-pushnotes.sh 4   # with vector search (bge-m3
#        via the e2e embedding container on localhost:11434); measures the
#        in-memory vector index cost. Results append to the same files.
# Results: tmp/bench/results.jsonl + results.csv + table on stdout.
#
# Requires the e2e MinIO container (docker compose -f docker-compose.test.yml
# up -d minio) — asset storage config is mandatory at boot.

set -e

CORES_LIST=("$@")
[ $# -eq 0 ] && CORES_LIST=(1 4)
NOTES_LIST=(10 100 1000 10000)

PORT=28081
INTERNAL_PORT=28082
BENCH_DIR=tmp/bench
RESULTS="$BENCH_DIR/results.jsonl"
BIN="$BENCH_DIR/trip2g-bench"

VECTOR="${VECTOR:-0}"
MOCK_EMBED_PORT=19001
MOCK_EMBED_PID=""
FEATURES_JSON="{}"
if [ "$VECTOR" = "1" ]; then
  # Use a local mock embedding server so 10k+ jobs complete instantly without
  # hitting the real model or maxReceive limits. The mock returns random unit
  # vectors of the correct dimension, which is sufficient for memory measurement.
  # go-openai appends /embeddings to BaseURL, so /v1 must be in the base_url.
  FEATURES_JSON="{\"vector_search\":{\"enabled\":true,\"model\":\"bge-m3\",\"base_url\":\"http://localhost:$MOCK_EMBED_PORT/v1\"}}"
fi

mkdir -p "$BENCH_DIR"
# Vector runs append to existing results so both variants land in one CSV.
[ "$VECTOR" = "1" ] || : > "$RESULTS"

echo "🔨 Building server binary..."
go build -o "$BIN" ./cmd/server

MOCK_EMBED_PID=""
stop_mock_server() {
  [ -n "$MOCK_EMBED_PID" ] && kill "$MOCK_EMBED_PID" 2>/dev/null || true
  MOCK_EMBED_PID=""
}

# SIGKILL: the server's graceful shutdown can hang, and a surviving instance
# squats on the port — the next run would silently measure the wrong server.
stop_server() {
  [ -n "$SERVER_PID" ] && kill -9 "$SERVER_PID" 2>/dev/null || true
  wait "$SERVER_PID" 2>/dev/null || true
  SERVER_PID=""
  for _ in $(seq 1 50); do
    ss -tln | grep -q ":$PORT " || return 0
    sleep 0.1
  done
  echo "✗ port $PORT still busy after stop" && exit 1
}
trap 'stop_server; stop_mock_server' EXIT

if [ "$VECTOR" = "1" ]; then
  echo "🤖 Starting mock embedding server on :$MOCK_EMBED_PORT..."
  node scripts/mock-embedding-server.mjs "$MOCK_EMBED_PORT" &
  MOCK_EMBED_PID=$!
  sleep 0.5
  curl -sf "http://localhost:$MOCK_EMBED_PORT/v1/embeddings" \
    -H 'Content-Type: application/json' \
    -d '{"input":["test"],"model":"bge-m3"}' > /dev/null \
    || { echo "✗ mock embedding server failed to start"; exit 1; }
  echo "   ✓ mock embedding server ready"
fi

for CORES in "${CORES_LIST[@]}"; do
  if [ "$CORES" = "1" ]; then CPUSET="0"; else CPUSET="0-$((CORES-1))"; fi

  for N in "${NOTES_LIST[@]}"; do
    DB_DIR="$BENCH_DIR/db"
    rm -rf "$DB_DIR" && mkdir -p "$DB_DIR"

    echo ""
    echo "▶ cores=$CORES notes=$N"

    if ss -tln | grep -q ":$PORT "; then
      echo "✗ port $PORT already in use — refusing to measure a foreign server"
      exit 1
    fi

    ENV_FILE=/dev/null \
    DB_FILE="$DB_DIR/bench.sqlite3" \
    LISTEN_ADDR="0.0.0.0:$PORT" \
    INTERNAL_LISTEN_ADDR=":$INTERNAL_PORT" \
    DEV=true \
    LOG_LEVEL=warn \
    OWNER_EMAIL=hello@example.com \
    PUBLIC_URL="http://localhost:$PORT" \
    JWT_SECRET=bench-secret-not-for-production \
    MINIO_ENDPOINT=localhost:29000 \
    MINIO_ACCESS_KEY_ID=testuser \
    MINIO_SECRET_KEY=testpassword \
    MINIO_BUCKET=test-bucket \
    MINIO_USE_SSL=false \
    SHUTDOWN_GRACE_PERIOD=1ms \
    SHUTDOWN_TIMEOUT=1ms \
    GLOBAL_QUEUE_POLL_INTERVAL=100ms \
    FEATURES="$FEATURES_JSON" \
    GOMAXPROCS="$CORES" \
    taskset -c "$CPUSET" "$BIN" > "$BENCH_DIR/server-$CORES-$N.log" 2>&1 &
    SERVER_PID=$!

    ./scripts/waitfor "localhost:$PORT" || {
      echo "✗ server failed to start (see $BENCH_DIR/server-$CORES-$N.log)"
      exit 1
    }

    if ! kill -0 "$SERVER_PID" 2>/dev/null; then
      echo "✗ server process died (see $BENCH_DIR/server-$CORES-$N.log)"
      exit 1
    fi

    APP_URL="http://localhost:$PORT" \
    INTERNAL_URL="http://localhost:$INTERNAL_PORT" \
    SERVER_PID="$SERVER_PID" \
    N="$N" \
    CORES="$CORES" \
    VECTOR="$VECTOR" \
    node scripts/bench-pushnotes.mjs | tee -a "$RESULTS"

    stop_server
  done
done

echo ""
echo "📊 Results ($RESULTS):"
node -e '
const fs = require("fs");
const rows = fs.readFileSync(process.argv[1], "utf8").trim().split("\n").map(JSON.parse);
const pad = (s, w) => String(s ?? "-").padStart(w);
const cols = ["cores", "notes", "vector", "embed_wait_ms", "initial_push_ms", "probe_ms", "incr_1_ms", "incr_10p_ms", "rss_mb", "peak_rss_mb"];
console.log(cols.map((h) => h.padStart(16)).join(""));
for (const r of rows) {
  console.log(cols.map((c) => pad(r[c], 16)).join("")
    + (r.initial_push_error ? `  INITIAL ERR: ${r.initial_push_error}` : "")
    + (r.probe_error ? `  PROBE ERR: ${r.probe_error}` : ""));
}
// CSV for datachart consumption.
const csv = [cols.join(",")].concat(rows.map((r) => cols.map((c) => r[c] ?? "").join(","))).join("\n") + "\n";
fs.writeFileSync(process.argv[1].replace(/\.jsonl$/, ".csv"), csv);
console.log("\nCSV written to " + process.argv[1].replace(/\.jsonl$/, ".csv"));
' "$RESULTS"
