#!/usr/bin/env python3
# usage: score.py <run.jsonl> <questions.jsonl>
import collections, json, math, sys
run = {json.loads(l)["id"]: json.loads(l) for l in open(sys.argv[1])}
qs = [json.loads(l) for l in open(sys.argv[2])]
K = (1, 3, 5, 10, 20)
agg = collections.defaultdict(lambda: collections.defaultdict(list))
empty = collections.Counter(); ms = []
for q in qs:
    r = run.get(q["question_id"])
    if not r: continue
    ms.append(r["ms"])
    if not r["docs"]: empty[q["question_type"]] += 1
    g = set(q["expected_doc_ids"])
    if not g: continue
    t = q["question_type"]; docs = r["docs"]
    first = next((i + 1 for i, d in enumerate(docs) if d in g), None)
    for bucket in (t, "ALL"):
        for k in K:
            agg[bucket][f"R{k}"].append(len(g & set(docs[:k])) / len(g))
            agg[bucket][f"H{k}"].append(1.0 if g & set(docs[:k]) else 0.0)
        agg[bucket]["MRR"].append(1 / first if first else 0)
        dcg = sum(1 / math.log2(i + 2) for i, d in enumerate(docs[:10]) if d in g)
        agg[bucket]["nDCG10"].append(dcg / sum(1 / math.log2(i + 2) for i in range(min(len(g), 10))))
print(f"{'type':28}{'n':>4}{'R@1':>7}{'R@5':>7}{'R@10':>7}{'R@20':>7}{'Hit@20':>8}{'MRR':>7}{'nDCG10':>8}")
m = lambda x: sum(x) / len(x)
for b in sorted(agg, key=lambda b: (b == "ALL", b)):
    a = agg[b]
    print(f"{b:28}{len(a['MRR']):4d}{m(a['R1']):7.3f}{m(a['R5']):7.3f}{m(a['R10']):7.3f}{m(a['R20']):7.3f}{m(a['H20']):8.3f}{m(a['MRR']):7.3f}{m(a['nDCG10']):8.3f}")
ms.sort(); print("queries", len(ms), "empty results", sum(empty.values()), dict(empty))
print("latency ms p50", ms[len(ms)//2], "p95", ms[int(len(ms)*.95)], "max", ms[-1], "errors", sum(1 for r in run.values() if r["err"]))
