#!/bin/bash
# Page render benchmark: measures per-request latency with k6.
# Compares default template vs custom Jet layout template across
# note counts and CPU core counts.
#
# Usage: ./scripts/bench-render.sh [cores...]   # default: 1 4
#
# Results: tmp/bench/render-results.jsonl + render-results.csv + table on stdout.
# Requires the e2e MinIO container:
#   docker compose -f docker-compose.test.yml up -d minio

set -e

CORES_LIST=("$@")
[ $# -eq 0 ] && CORES_LIST=(1 4)
NOTES_LIST=(100 1000 10000)

PORT=28083
INTERNAL_PORT=28084
BENCH_DIR=tmp/bench
RESULTS="$BENCH_DIR/render-results.jsonl"
BIN="$BENCH_DIR/trip2g-bench"

VUS=10
DURATION=20s

mkdir -p "$BENCH_DIR"
: > "$RESULTS"

echo "🔨 Building server binary..."
go build -o "$BIN" ./cmd/server

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
trap stop_server EXIT

k6_p() {
  local file="$1" metric="$2"
  node -e "
    const d = JSON.parse(require('fs').readFileSync('$file','utf8'));
    const v = d.metrics && d.metrics.http_req_duration;
    process.stdout.write(v ? String(Math.round(v['$metric'] || 0)) : '-');
  " 2>/dev/null
}

for CORES in "${CORES_LIST[@]}"; do
  if [ "$CORES" = "1" ]; then CPUSET="0"; else CPUSET="0-$((CORES-1))"; fi

  for N in "${NOTES_LIST[@]}"; do
    DB_DIR="$BENCH_DIR/db"
    rm -rf "$DB_DIR" && mkdir -p "$DB_DIR"

    echo ""
    echo "▶ cores=$CORES notes=$N"

    if ss -tln | grep -q ":$PORT "; then
      echo "✗ port $PORT already in use"
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
    GOMAXPROCS="$CORES" \
    taskset -c "$CPUSET" "$BIN" > "$BENCH_DIR/render-server-$CORES-$N.log" 2>&1 &
    SERVER_PID=$!

    ./scripts/waitfor "localhost:$PORT" || {
      echo "✗ server failed to start"
      exit 1
    }

    APP_URL="http://localhost:$PORT" N="$N" node scripts/bench-render-setup.mjs

    SUMMARY_DEFAULT="$BENCH_DIR/k6-default-$CORES-$N.json"
    BASE_URL="http://localhost:$PORT" \
    NOTE_COUNT="$N" \
    NOTE_PREFIX="/bench/note_" \
    VUS="$VUS" \
    DURATION="$DURATION" \
    SCENARIO=default \
    k6 run --summary-export="$SUMMARY_DEFAULT" --quiet k6/bench-render.js

    APP_URL="http://localhost:$PORT" N="$N" LAYOUT=1 node scripts/bench-render-setup.mjs

    SUMMARY_LAYOUT="$BENCH_DIR/k6-layout-$CORES-$N.json"
    BASE_URL="http://localhost:$PORT" \
    NOTE_COUNT="$N" \
    NOTE_PREFIX="/bench_layout/note_" \
    VUS="$VUS" \
    DURATION="$DURATION" \
    SCENARIO=layout \
    k6 run --summary-export="$SUMMARY_LAYOUT" --quiet k6/bench-render.js

    node -e "
      const fs = require('fs');
      function p(file, metric) {
        try {
          const d = JSON.parse(fs.readFileSync(file,'utf8'));
          const v = d.metrics && d.metrics.http_req_duration;
          return v ? Math.round(v[metric] || 0) : null;
        } catch { return null; }
      }
      const row = {
        cores: '$CORES', notes: $N,
        default_p50: p('$SUMMARY_DEFAULT','p(50)'),
        default_p95: p('$SUMMARY_DEFAULT','p(95)'),
        default_p99: p('$SUMMARY_DEFAULT','p(99)'),
        default_avg: p('$SUMMARY_DEFAULT','avg'),
        layout_p50:  p('$SUMMARY_LAYOUT','p(50)'),
        layout_p95:  p('$SUMMARY_LAYOUT','p(95)'),
        layout_p99:  p('$SUMMARY_LAYOUT','p(99)'),
        layout_avg:  p('$SUMMARY_LAYOUT','avg'),
      };
      process.stdout.write(JSON.stringify(row) + '\n');
    " | tee -a "$RESULTS"

    stop_server
  done
done

echo ""
echo "📊 Render results ($RESULTS):"
node -e '
const fs = require("fs");
const rows = fs.readFileSync(process.argv[1], "utf8").trim().split("\n").map(JSON.parse);
const pad = (s, w) => String(s ?? "-").padStart(w);
const cols = ["cores","notes","default_p50","default_p95","default_p99","layout_p50","layout_p95","layout_p99"];
console.log(cols.map((h) => h.padStart(14)).join(""));
for (const r of rows) console.log(cols.map((c) => pad(r[c], 14)).join(""));
const csv = [cols.join(",")].concat(rows.map((r) => cols.map((c) => r[c] ?? "").join(","))).join("\n") + "\n";
fs.writeFileSync(process.argv[1].replace(/\.jsonl$/, ".csv"), csv);
console.log("\nCSV written to " + process.argv[1].replace(/\.jsonl$/, ".csv"));
' "$RESULTS"
