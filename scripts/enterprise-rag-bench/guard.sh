#!/usr/bin/env bash
cd "${ERB_WORK:?set ERB_WORK to the work directory}"
# Stops the server when the key's usage passes the budget in dollars.
budget=$1; k=$(cat .key)
start=$(curl -sS -m 20 https://openrouter.ai/api/v1/key -H "Authorization: Bearer $k" | python3 -c "import sys,json;print(json.load(sys.stdin)['data']['usage'])")
while kill -0 "$(cat server.pid)" 2>/dev/null; do
  u=$(curl -sS -m 20 https://openrouter.ai/api/v1/key -H "Authorization: Bearer $k" | python3 -c "import sys,json;print(json.load(sys.stdin)['data']['usage'])" 2>/dev/null) || u=$start
  echo "$(date +%T) usage $u"
  if python3 -c "import sys;sys.exit(0 if $u-$start>$budget else 1)"; then echo "BUDGET HIT"; kill -9 "$(cat server.pid)"; exit 1; fi
  sleep 30
done
