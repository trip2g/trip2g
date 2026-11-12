#!/usr/bin/env python3
# usage: push.py <corpus.jsonl> <port> <batch> [limit]
import json, sys, time, urllib.request
cp, port, bs = sys.argv[1], sys.argv[2], int(sys.argv[3])
limit = int(sys.argv[4]) if len(sys.argv) > 4 else None
key = open("apikey").read().strip()
url = f"http://127.0.0.1:{port}/_system/graphql"
M = "mutation($i: PushNotesInput!){ pushNotes(input:$i){ __typename ... on ErrorPayload { message } } }"
rows = [json.loads(l) for l in open(cp)][:limit]
for s in range(0, len(rows), bs):
    b = rows[s:s + bs]; t = time.time()
    body = json.dumps({"query": M, "variables": {"i": {"updates": b}}}).encode()
    r = json.load(urllib.request.urlopen(urllib.request.Request(url, body, {"Content-Type": "application/json", "X-API-Key": key}), timeout=3600))
    tn = r.get("data", {}).get("pushNotes", {}).get("__typename") if r.get("data") else None
    print(s + len(b), tn, r.get("errors") or r.get("data", {}).get("pushNotes", {}).get("message", ""), round(time.time() - t, 1), flush=True)
