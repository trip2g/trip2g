#!/usr/bin/env bash
# qmd with its defaults, then with Qwen3-Embedding (the one-variable Russian fix).
# Resumable. qmd needs a GPU to be practical: on a shared CPU a query took 3-7 min,
# on an M3 Max with Metal ~10 s.
#   (cd "$OHB_WORK" && npm i @tobilu/qmd@2.8.3)   # Node >= 22 with N-API 10 (Node 24 works)
set -eu
here=$(cd "$(dirname "$0")" && pwd); repo=$(cd "$here/../.." && pwd)
WORK=${OHB_WORK:-$repo/tmp/obsidian-help-bench}
export XDG_CACHE_HOME="$WORK/qmd-cache"
Q="$WORK/node_modules/.bin/qmd"
mkdir -p "$WORK/logs"

drive() { # <index> <port> <label> <lang>
  local idx=$1 port=$2 label=$3 lang=$4
  if [ ! -f "$WORK/logs/.embedded-$idx" ]; then
    "$Q" --index "$idx" collection add "$WORK/vault-$lang" --name "$lang" >> "$WORK/logs/qmd-setup.log" 2>&1 || true
    "$Q" --index "$idx" embed --timeout 0 >> "$WORK/logs/qmd-setup.log" 2>&1
    touch "$WORK/logs/.embedded-$idx"
  fi
  for attempt in $(seq 1 30); do
    if ! curl -sf "localhost:$port/health" >/dev/null; then
      (nohup "$Q" --index "$idx" mcp --http --port "$port" >> "$WORK/logs/qmd-$idx.log" 2>&1 &)
      until curl -sf "localhost:$port/health" >/dev/null; do sleep 2; done
    fi
    python3 "$here/run_qmd.py" "$lang" "$port" "$label" >> "$WORK/logs/run-$label-$lang.log" 2>&1 && break
    echo "attempt $attempt failed, restarting" >> "$WORK/logs/run-$label-$lang.log"; sleep 5
  done
  lsof -ti "tcp:$port" -sTCP:LISTEN | xargs kill 2>/dev/null || true
}

drive en 18181 qmd en
drive ru 18182 qmd ru
export QMD_EMBED_MODEL="hf:Qwen/Qwen3-Embedding-0.6B-GGUF/Qwen3-Embedding-0.6B-Q8_0.gguf"
drive en-qwen 18183 qmd-qwen en
drive ru-qwen 18184 qmd-qwen ru
echo "qmd done"
