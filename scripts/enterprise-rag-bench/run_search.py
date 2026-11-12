#!/usr/bin/env python3
# usage: run_search.py <label> [limit]; writes logs/run-<label>.jsonl with ranked doc ids and latency
import json, re, sys, time, urllib.request
label = sys.argv[1]; limit = int(sys.argv[2]) if len(sys.argv) > 2 else None
token = open("token").read().strip()
url = "http://127.0.0.1:21201/_system/graphql"
S = """query($q: String!) { search(input: {query: $q}) { nodes { document { ... on PublicNote { pathId } } } } }"""
def gql(q, v):
    body = json.dumps({"query": q, "variables": v}).encode()
    req = urllib.request.Request(url, body, {"Content-Type": "application/json", "Cookie": "trip2g_bench=" + token})
    d = json.load(urllib.request.urlopen(req, timeout=120))
    if d.get("errors"): raise RuntimeError(d["errors"])
    return d["data"]
paths = {p["id"]: p["value"] for p in gql("{ notePaths { id value } }", {})["notePaths"]}
print("paths", len(paths), flush=True)
did = lambda p: (re.search(r"dsid_[0-9a-f]+", p) or [p])[0]
qs = [json.loads(l) for l in open("questions.jsonl")][:limit]
with open(f"logs/run-{label}.jsonl", "w") as f:
    for i, q in enumerate(qs):
        t = time.time(); err = None; docs = []
        try:
            nodes = gql(S, {"q": q["question"]})["search"]["nodes"]
            docs = [did(paths[n["document"]["pathId"]]) for n in nodes if n.get("document")]
        except Exception as e:
            err = repr(e)[:200]
        f.write(json.dumps({"id": q["question_id"], "type": q["question_type"], "docs": docs, "ms": round((time.time() - t) * 1000), "err": err}) + "\n"); f.flush()
        if i % 50 == 0: print(i, flush=True)
