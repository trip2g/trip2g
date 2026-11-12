#!/usr/bin/env python3
import collections, json, re, sqlite3, time, urllib.request
import numpy as np
qs = [json.loads(l) for l in open("questions.jsonl")]
Q = np.load("logs/question-vectors.npy")
var = json.load(open("logs/variants.json"))
dense, bm = var["dense chunk maxsim"], var["bm25 OR (fts5)"]
K = open(".key").read().strip()
did = lambda p: re.search(r"dsid_[0-9a-f]+", p)[0]
c = sqlite3.connect("file:bench.sqlite3?mode=ro", uri=True, timeout=10)
docs = []; srcs = []; lens = []; vecs = []; contents = collections.defaultdict(list)
for p, e, t in c.execute("select p.value, k.embedding, k.content from note_version_chunks k join note_versions v on v.id=k.version_id join note_paths p on p.id=v.path_id where k.embedding is not null and p.value like '%dsid_%'"):
    d = did(p); docs.append(d); srcs.append(p.split("/")[0]); lens.append(len(t)); vecs.append(np.frombuffer(e, dtype=np.float32)); contents[d].append(t)
V = np.stack(vecs); del vecs; L = np.array(lens); S = np.array(srcs); D = np.array(docs)
uniq, inv = np.unique(D, return_inverse=True); idx = {d: i for i, d in enumerate(uniq)}
print("=== chunk size by source (chars; ~4 chars per token, encoder limit 8192 tokens on this API, TEI would cut at 512)")
print(f"{'source':14}{'docs':>7}{'chunks/doc':>11}{'p50':>7}{'p90':>7}{'max':>8}{'>2048':>8}{'>4096':>8}")
for s in sorted(set(srcs)):
    m = S == s; x = L[m]; nd = len(set(D[m]))
    print(f"{s:14}{nd:7d}{m.sum()/nd:11.2f}{int(np.median(x)):7d}{int(np.percentile(x,90)):7d}{x.max():8d}{(x>2048).mean():8.2f}{(x>4096).mean():8.2f}")
orig = {}
for l in open("corpus50k.clean.jsonl"):
    r = json.loads(l); orig[did(r["path"])] = r["content"]
gold = sorted({d for q in qs for d in q["expected_doc_ids"]})
print("=== gold document shape: paragraphs and whether chunks cover the text")
print("mean newlines-per-1000-chars, literal \\\\n per 1000", )
for s in sorted(set(srcs)):
    ds = [d for d in gold if d in idx and S[inv == idx[d]][0] == s] if False else []
stats = collections.defaultdict(list)
srcof = {d: s for d, s in zip(docs, srcs)}
for d in gold:
    t = orig.get(d, "")
    stats[srcof.get(d, "?")].append((len(t), t.count("\n\n"), t.count("\\n"), len(contents.get(d, [])), max([len(x) for x in contents.get(d, [])] or [0])))
print(f"{'source':14}{'gold':>6}{'chars':>8}{'paras':>7}{'lit\\n':>7}{'chunks':>8}{'maxchunk':>9}")
for s, v in sorted(stats.items()):
    a = np.array(v); print(f"{s:14}{len(v):6d}{a[:,0].mean():8.0f}{a[:,1].mean():7.1f}{a[:,2].mean():7.1f}{a[:,3].mean():8.1f}{a[:,4].mean():9.0f}")
print("=== taxonomy of gold docs (per question-doc pair)")
tax = collections.Counter(); bytype = collections.defaultdict(collections.Counter); rows = []
for q in qs:
    for g in q["expected_doc_ids"]:
        dr = dense[q["question_id"]].index(g) + 1 if g in dense[q["question_id"]] else 99999
        br = bm[q["question_id"]].index(g) + 1 if g in bm[q["question_id"]] else 99999
        k = ("dense" if dr <= 10 else "") + ("+" if dr <= 10 and br <= 10 else "") + ("bm25" if br <= 10 else "") or "neither"
        tax[k] += 1; bytype[q["question_type"]][k] += 1
        ch = contents.get(g, []); rows.append((q["question_id"], g, dr, br, len(ch), max([len(x) for x in ch] or [0]), srcof.get(g, "?")))
print(dict(tax)); [print(" ", t, dict(v)) for t, v in sorted(bytype.items())]
print("=== dense R@10 by the gold doc's largest chunk")
for lo, hi in [(0, 1000), (1000, 2000), (2000, 3000), (3000, 5000), (5000, 10**9)]:
    x = [r for r in rows if lo <= r[5] < hi]
    if x: print(f"  maxchunk {lo:6d}-{hi if hi<10**9 else 'inf':>6}: n={len(x):4d} dense@10={np.mean([r[2]<=10 for r in x]):.3f} bm25@10={np.mean([r[3]<=10 for r in x]):.3f}")
print("=== dense R@10 by chunks in gold doc")
for lo, hi in [(1, 2), (2, 3), (3, 5), (5, 8), (8, 999)]:
    x = [r for r in rows if lo <= r[4] < hi]
    if x: print(f"  chunks {lo}-{hi-1}: n={len(x):4d} dense@10={np.mean([r[2]<=10 for r in x]):.3f} bm25@10={np.mean([r[3]<=10 for r in x]):.3f}")
print("=== near-duplicate pressure: top-1 dense doc vs gold, for dense misses")
titles = {d: orig[d].split("\n", 2)[1][:70] if d in orig else "" for d in orig}
miss = [r for r in rows if r[2] > 10]
print("misses", len(miss))
qt = {q["question_id"]: q for q in qs}
for r in miss[:0]: pass
import random; random.Random(3).shuffle(miss)
for r in miss[:12]:
    q = qt[r[0]]; t1 = dense[r[0]][0]
    print(f" {r[0]} {q['question_type'][:8]} src={r[6]} rank={r[2]} bm={r[3]} chunks={r[4]} maxchunk={r[5]}\n   Q: {q['question'][:125]}\n   gold: {titles.get(r[1],'')}\n   top1: {titles.get(t1,'')}")
print("=== re-chunk experiment: only gold docs, fixed windows, others unchanged (optimistic for gold)")
def windows(t, size=1200, over=200):
    m = re.match(r'---\ntitle: (".*?")\n---\n\n(.*)', t, re.S); title, body = (json.loads(m.group(1)), m.group(2)) if m else ("", t)
    body = body.replace("\\n", "\n"); out = []; i = 0
    while i < len(body):
        out.append(title + "\n\n" + body[i:i + size]); i += size - over
    return out or [title]
wins = {d: windows(orig[d]) for d in gold if d in orig}; flat = [(d, w) for d, ws in wins.items() for w in ws]
print("windows", len(flat), "est tokens", sum(len(w) for _, w in flat) // 4, flush=True)
vec = []
for i in range(0, len(flat), 64):
    b = json.dumps({"model": "baai/bge-m3", "input": [w for _, w in flat[i:i + 64]]}).encode()
    for a in range(8):
        try:
            r = json.load(urllib.request.urlopen(urllib.request.Request("https://openrouter.ai/api/v1/embeddings", b, {"Authorization": "Bearer " + K, "Content-Type": "application/json"}), timeout=180)); vec += [x["embedding"] for x in r["data"]]; break
        except Exception: time.sleep(4 * (a + 1))
    else: raise SystemExit("embed fail")
G = np.array(vec, dtype=np.float32); gd = np.array([d for d, _ in flat]); np.save("logs/gold-windows.npy", G)
base = []; new = []
for i, q in enumerate(qs):
    if not q["expected_doc_ids"]: continue
    s = V @ Q[i]; best = np.full(len(uniq), -9.0, dtype=np.float32); np.maximum.at(best, inv, s)
    gs = G @ Q[i]
    for g in q["expected_doc_ids"]:
        if g not in idx: continue
        old = best[idx[g]]; nw = gs[gd == g].max() if (gd == g).any() else old
        base.append((q["question_type"], int((best > old).sum()) + 1)); new.append((q["question_type"], int((best > nw).sum()) + 1))
for name, data in (("current chunks", base), ("1200-char windows", new)):
    a = np.array([r for _, r in data]); sem = np.array([r for t, r in data if t == "semantic"]); bas = np.array([r for t, r in data if t == "basic"])
    print(f"  {name:20} doc-level hit@10 all={np.mean(a<=10):.3f} semantic={np.mean(sem<=10):.3f} basic={np.mean(bas<=10):.3f}  @50 all={np.mean(a<=50):.3f}")
