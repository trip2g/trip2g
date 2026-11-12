#!/usr/bin/env bash
cd "${ERB_WORK:?set ERB_WORK to the work directory}"
# Load test of the final build on 50k notes. Raises concurrency stepwise, stops if the host load climbs.
T=$(cat token); B=http://127.0.0.1:21201; OUT=logs/http; mkdir -p $OUT
NOTE=/confluence/dsid_05841423a55e4c35abd276d9899b7134
SEARCH='{"query":"query($q:String!){search(input:{query:$q}){nodes{url}}}","variables":{"q":"How do we verify whether a blended output token price jump is a rate card change?"}}'
step() { # name conc dur args...
  n=$1; c=$2; d=$3; shift 3
  for i in $(seq 1 40); do l=$(cut -d' ' -f1 /proc/loadavg); python3 -c "import sys;sys.exit(0 if $l<5 else 1)" && break; sleep 15; done
  r0=$(ps -o rss= -p $(cat server.pid))
  ./hey -z $d -c $c -t 60 "$@" > $OUT/$n-c$c.txt 2>&1
  r1=$(ps -o rss= -p $(cat server.pid)); p=$(ps -o pcpu= -p $(cat server.pid))
  python3 - "$n" "$c" "$r0" "$r1" "$OUT/$n-c$c.txt" "$l" <<'PY'
import re,sys
n,c,r0,r1,f,l=sys.argv[1:]; t=open(f).read()
g=lambda p:(re.search(p,t) or [None,'-'])[1]
codes=re.findall(r"\[(\d+)\]\s+(\d+) responses",t)
print(f"{n:12} c={c:3} rps={g(r'Requests/sec:\s+([\d.]+)'):>9} p50={g(r'50%+ in ([\d.]+)'):>7}s p95={g(r'95%+ in ([\d.]+)'):>7}s p99={g(r'99%+ in ([\d.]+)'):>7}s codes={codes} rssMB={int(r0)//1024}->{int(r1)//1024} load={l}",flush=True)
PY
  sleep 5
}
C="Cookie: trip2g_bench=$T"
curl -s -m 40 -o logs/pprof/cpu-note.pb.gz "http://127.0.0.1:21202/debug/pprof/profile?seconds=20" & P=$!
sleep 1; ./hey -z 18s -c 4 -t 60 -H "$C" $B$NOTE > $OUT/note-profile.txt 2>&1; wait $P
for c in 1 10 50; do step note-anon $c 15s $B$NOTE; done
for c in 1 10 50; do step livez $c 10s http://127.0.0.1:21202/livez; done
for c in 1 5 20; do step search $c 30s -m POST -H "$C" -H 'Content-Type: application/json' -d "$SEARCH" $B/_system/graphql; done
echo DONE
