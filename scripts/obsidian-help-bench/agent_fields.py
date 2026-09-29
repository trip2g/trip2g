#!/usr/bin/env python3
"""Simulate agent-side query expansion on top of trip2g's retrieval.

An MCP agent passes three optional fields with the query (see
testdata/eval/obsidian-help/agent-fields-*.json, written by a model that saw only
the query and the field descriptions):
  short_query      2-3 key words      -> text lane only
  rephrased_query  another phrasing   -> vector lane only
  expected_answer  a sentence the answering note would contain -> vector lane only
trip2g's lanes are rebuilt outside the server from the same data:
  text(q)  a trip2g instance with vector search off (bleve, <=20 results)
  vec(q)   bge-m3 query vector · every stored chunk, best chunk per note, top 50
           (internal/case/sitesearch/retrieve.go vectorSearch)
and fused with RRF k=60, capped at 20. The `t2g-sim` run must equal a real trip2g
run on the same data; that is the check that the rebuild is faithful.

usage: agent_fields.py <lang> <text-only port> <db of the vector instance> [embed url]
                       [--fields FILE --label NAME]
With --fields (output of agent_fields_gen.py) only one run, NAME, is written: all three
fields at weight 1; empty fields add no list.
"""
import json
import os
import sqlite3
import sys
import urllib.request

import numpy as np

from common import DATA, WORK, queries, write_run

args = sys.argv[1:]
fields_file = label = None
if "--fields" in args:
    i = args.index("--fields")
    fields_file, label = args[i + 1], args[args.index("--label") + 1]
    args = args[:i]
lang, port, db = args[0], args[1], args[2]
embed_url = args[3] if len(args) > 3 else "http://localhost:11439/v1/embeddings"
K, VEC_TOP, CAP = 60, 50, 20
cache_path = os.path.join(WORK, f"lane-cache-{lang}.json")
cache = json.load(open(cache_path)) if os.path.exists(cache_path) else {}

con = sqlite3.connect(f"file:{db}?mode=ro", uri=True)
rows = con.execute(
    "select p.value, c.embedding from note_version_chunks c join note_versions v on v.id = c.version_id "
    "join note_paths p on p.id = v.path_id where v.version = p.version_count and c.embedding is not null").fetchall()
chunk_paths = np.array([r[0] for r in rows])
chunks = np.vstack([np.frombuffer(r[1], dtype=np.float32) for r in rows])

token = open(os.path.join(WORK, f"token-{port}")).read().strip()


def gql(query, variables):
    body = json.dumps({"query": query, "variables": variables}).encode()
    req = urllib.request.Request(f"http://127.0.0.1:{port}/_system/graphql", body,
                                 {"Content-Type": "application/json", "Cookie": f"trip2g_bench={token}"})
    d = json.load(urllib.request.urlopen(req, timeout=600))
    if d.get("errors"):
        raise RuntimeError(d["errors"])
    return d["data"]


path_of = {p["id"]: p["value"] for p in gql("{ notePaths { id value } }", {})["notePaths"]}


def text_lane(q):
    key = "text:" + q
    if key not in cache:
        nodes = gql("query($q: String!) { search(input: {query: $q}) { nodes { document { ... on PublicNote { pathId } } } } }",
                    {"q": q})["search"]["nodes"]
        cache[key] = [path_of[n["document"]["pathId"]] for n in nodes if n.get("document")]
    return cache[key]


def vec_lane(q):
    key = "vec:" + q
    if key not in cache:
        body = json.dumps({"input": [q], "model": "bge-m3"}).encode()
        req = urllib.request.Request(embed_url, body, {"Content-Type": "application/json"})
        v = np.array(json.load(urllib.request.urlopen(req, timeout=600))["data"][0]["embedding"], dtype=np.float32)
        seen, out = set(), []
        for j in np.argsort(-(chunks @ v)):
            p = str(chunk_paths[j])
            if p not in seen:
                seen.add(p)
                out.append(p)
                if len(out) == VEC_TOP:
                    break
        cache[key] = out
    return cache[key]


def rrf(lists):
    score = {}
    for weight, lst in lists:
        for rank, p in enumerate(lst):
            score[p] = score.get(p, 0.0) + weight / (K + rank + 1)
    return [p for p, _ in sorted(score.items(), key=lambda kv: (-kv[1], kv[0]))][:CAP]


fields = {f["id"]: f for f in json.load(open(fields_file or os.path.join(DATA, f"agent-fields-{lang}.json"),
                                             encoding="utf-8"))}
if fields_file:
    res = {}
    for q in queries(lang):
        f = fields[q["id"]]
        lists = [(1.0, text_lane(q["query"])), (1.0, vec_lane(q["query"]))]
        if f.get("short_query"):
            lists.append((1.0, text_lane(f["short_query"])))
        for k in ("rephrased_query", "expected_answer"):
            if f.get(k):
                lists.append((1.0, vec_lane(f[k])))
        res[q["id"]] = rrf(lists)
    json.dump(cache, open(cache_path, "w", encoding="utf-8"), ensure_ascii=False)
    write_run(label, lang, res)
    sys.exit(0)
runs = {name: {} for name in ("t2g-sim", "fields-short", "fields-w05", "fields-w1")}
for q in queries(lang):
    f = fields[q["id"]]
    base = [(1.0, text_lane(q["query"])), (1.0, vec_lane(q["query"]))]
    extra = lambda w: [(w, text_lane(f["short_query"])), (w, vec_lane(f["rephrased_query"])),
                       (w, vec_lane(f["expected_answer"]))]
    runs["t2g-sim"][q["id"]] = rrf(base)
    runs["fields-short"][q["id"]] = rrf(base + [(1.0, text_lane(f["short_query"]))])
    runs["fields-w05"][q["id"]] = rrf(base + extra(0.5))
    runs["fields-w1"][q["id"]] = rrf(base + extra(1.0))
json.dump(cache, open(cache_path, "w", encoding="utf-8"), ensure_ascii=False)
for name, res in runs.items():
    write_run(name, lang, res)
