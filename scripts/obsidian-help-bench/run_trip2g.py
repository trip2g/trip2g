#!/usr/bin/env python3
"""Run every query of one language through a trip2g instance (GraphQL `search`, as admin).

usage: run_trip2g.py <lang> <port> <label> [--rerank]
The session token comes from $OHB_WORK/token-<port> (written by trip2g_up.sh).
"""
import json
import os
import sys
import urllib.request

from common import WORK, queries, write_run

lang, port, label = sys.argv[1], sys.argv[2], sys.argv[3]
rerank = "--rerank" in sys.argv[4:]
token = open(os.path.join(WORK, f"token-{port}")).read().strip()
url = f"http://127.0.0.1:{port}/_system/graphql"
SEARCH = """query($q: String!, $r: Boolean) { search(input: {query: $q, rerank: $r}) {
  nodes { document { ... on PublicNote { pathId } } } } }"""


def gql(query, variables):
    body = json.dumps({"query": query, "variables": variables}).encode()
    req = urllib.request.Request(url, body, {"Content-Type": "application/json",
                                             "Cookie": f"trip2g_bench={token}"})
    data = json.load(urllib.request.urlopen(req, timeout=600))
    if data.get("errors"):
        raise RuntimeError(data["errors"])
    return data["data"]


# PublicNote.path is the permalink; map pathId back to the vault file path.
path_of = {p["id"]: p["value"] for p in gql("{ notePaths { id value } }", {})["notePaths"]}
results = {}
for q in queries(lang):
    nodes = gql(SEARCH, {"q": q["query"], "r": rerank})["search"]["nodes"]
    results[q["id"]] = [path_of[n["document"]["pathId"]] for n in nodes if n.get("document")]
write_run(label, lang, results)
