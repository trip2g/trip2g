#!/usr/bin/env bash
cd "${ERB_WORK:?set ERB_WORK to the work directory}"
# Full Prometheus snapshot every 60s into logs/metrics/<epoch>.prom.gz
mkdir -p logs/metrics
while kill -0 "$(cat server.pid)" 2>/dev/null; do
  curl -s -m 20 http://127.0.0.1:21202/metrics | gzip > logs/metrics/$(date +%s).prom.gz
  sleep 60
done
