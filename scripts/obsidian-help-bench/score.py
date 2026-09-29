#!/usr/bin/env python3
"""Score ranked results against a golden set with the obsidian-hybrid-search
metric definitions (eval/metrics.ts): graded nDCG (relevant 1.0, partial 0.5),
MRR / Hit@k / Recall@k over fully relevant paths only.

usage: score.py golden.json run1.json [run2.json ...]
run file: {"system": "...", "results": {"<query id>": ["path.md", ...]}}
"""
import collections
import json
import math
import sys
import unicodedata


def dcg(scores, k):
    return sum(s / math.log2(i + 2) for i, s in enumerate(scores[:k]))


def ndcg(res, rel, part, k):
    r, p = set(rel), set(part)
    actual = [1.0 if x in r else 0.5 if x in p else 0.0 for x in res[:k]]
    ideal = ([1.0] * len(rel) + [0.5] * len(part))[:k]
    i = dcg(ideal, k)
    return dcg(actual, k) / i if i else 0.0


def mrr(res, rel):
    r = set(rel)
    return next((1 / (i + 1) for i, x in enumerate(res) if x in r), 0.0)


def hit(res, rel, k):
    return any(x in set(rel) for x in res[:k])


def recall(res, rel, k):
    return len(set(res[:k]) & set(rel)) / len(rel)


def nfc(paths):
    # obsidian-hybrid-search returns NFD paths; compare canonically.
    return [unicodedata.normalize("NFC", p) for p in paths]


def metrics(golden, results):
    rows = []
    for q in golden:
        res = nfc(results.get(q["id"], []))
        rel, part = nfc(q["relevant_paths"]), nfc(q.get("partial_paths", []))
        rows.append((q["category"], {
            "nDCG@5": ndcg(res, rel, part, 5), "nDCG@10": ndcg(res, rel, part, 10),
            "MRR": mrr(res, rel), "Hit@1": float(hit(res, rel, 1)),
            "Hit@5": float(hit(res, rel, 5)), "R@10": recall(res, rel, 10),
        }))
    return rows


def avg(rows):
    keys = rows[0][1].keys()
    return {k: sum(r[1][k] for r in rows) / len(rows) for k in keys}


golden = json.load(open(sys.argv[1]))
runs = [json.load(open(p)) for p in sys.argv[2:]]
cols = ["nDCG@5", "nDCG@10", "MRR", "Hit@1", "Hit@5", "R@10"]
print(f"{'system':28s} " + " ".join(f"{c:>7s}" for c in cols) + f"  (n={len(golden)})")
cats = sorted({q["category"] for q in golden})
per_cat = {}
for run in runs:
    rows = metrics(golden, run["results"])
    a = avg(rows)
    print(f"{run['system']:28s} " + " ".join(f"{a[c]:7.3f}" for c in cols))
    by = collections.defaultdict(list)
    for c, m in rows:
        by[c].append((c, m))
    per_cat[run["system"]] = {c: avg(v)["nDCG@5"] for c, v in by.items()}
print("\nnDCG@5 by category")
print(f"{'category':16s} {'n':>3s} " + " ".join(f"{r['system'][:14]:>14s}" for r in runs))
for c in cats:
    n = sum(1 for q in golden if q["category"] == c)
    print(f"{c:16s} {n:3d} " + " ".join(f"{per_cat[r['system']].get(c, 0):14.3f}" for r in runs))
