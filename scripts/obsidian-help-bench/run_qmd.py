#!/usr/bin/env python3
"""Run queries through a running `qmd mcp --http` daemon via its `query` tool with the
plain query text (qmd's default path: its own query expansion + reranker). Resumable.

usage: run_qmd.py <lang> <port> <label>
Normally driven by qmd.sh, which starts and restarts the daemon.
"""
import json
import os
import sys
import time
import urllib.request

from common import RUNS, queries, write_run

lang, port, label = sys.argv[1], sys.argv[2], sys.argv[3]
out = os.path.join(RUNS, f"{label}-{lang}.json")
results = json.load(open(out))["results"] if os.path.exists(out) else {}


def call(method, params, i):
    body = json.dumps({"jsonrpc": "2.0", "id": i, "method": method, "params": params}).encode()
    req = urllib.request.Request(f"http://localhost:{port}/mcp", body, {
        "Content-Type": "application/json", "Accept": "application/json, text/event-stream"})
    raw = urllib.request.urlopen(req, timeout=7200).read().decode()
    for line in raw.splitlines():
        if line.startswith("data:"):
            raw = line[5:]
    return json.loads(raw)


call("initialize", {"protocolVersion": "2025-06-18", "capabilities": {},
                    "clientInfo": {"name": "bench", "version": "1"}}, 1)
for q in queries(lang):
    if q["id"] in results:
        continue
    t = time.time()
    r = call("tools/call", {"name": "query", "arguments": {"query": q["query"], "limit": 20}}, 2)["result"]
    files = [x["file"] for x in r.get("structuredContent", {}).get("results", [])]
    results[q["id"]] = [f.split("/", 1)[1] if "/" in f else f for f in files]  # drop the collection name
    write_run(label, lang, results, quiet=True)
    print(q["id"], round(time.time() - t), "s", len(files), flush=True)
