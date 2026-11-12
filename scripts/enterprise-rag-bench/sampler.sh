#!/usr/bin/env bash
cd "${ERB_WORK:?set ERB_WORK to the work directory}"
# One CSV row per 15s: time, rss_mb, cpu_pct, db_mb, s3_mb, jobs_done
echo "time,rss_mb,cpu,db_mb,s3_mb"
while kill -0 "$(cat server.pid)" 2>/dev/null; do
  p=$(cat server.pid)
  echo "$(date +%s),$(($(ps -o rss= -p $p)/1024)),$(ps -o pcpu= -p $p | tr -d ' '),$(($(du -c bench.sqlite3* 2>/dev/null | tail -1 | cut -f1)/1024)),$(($(du -s s3 | cut -f1)/1024))"
  sleep 15
done
