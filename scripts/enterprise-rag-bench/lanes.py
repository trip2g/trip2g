#!/usr/bin/env python3
# Offline retrieval variants over the same corpus and vectors as the running instance.
import collections, json, os, re, sqlite3, time, urllib.request
import numpy as np
qs = [json.loads(l) for l in open("questions.jsonl")]
K = open(".key").read().strip()
cache = "logs/question-vectors.npy"
if os.path.exists(cache):
    Q = np.load(cache)
else:
    out = []
    for i in range(0, len(qs), 32):
        b = json.dumps({"model": "baai/bge-m3", "input": [q["question"] for q in qs[i:i+32]]}).encode()
        for attempt in range(6):
            try:
                r = json.load(urllib.request.urlopen(urllib.request.Request("https://openrouter.ai/api/v1/embeddings", b, {"Authorization": "Bearer " + K, "Content-Type": "application/json"}), timeout=120))
                out += [x["embedding"] for x in r["data"]]; break
            except Exception as e:
                time.sleep(3 * (attempt + 1))
        else: raise SystemExit("embedding failed")
    Q = np.array(out, dtype=np.float32); np.save(cache, Q)
print("Q", Q.shape, "norm", round(float(np.linalg.norm(Q[0])), 4), flush=True)
c = sqlite3.connect("file:bench.sqlite3?mode=ro", uri=True, timeout=10)
print([r[1] for r in c.execute("pragma table_info(note_version_chunks)")], flush=True)
did = lambda p: re.search(r"dsid_[0-9a-f]+", p)[0]
rows = c.execute("select p.value, k.embedding from note_version_chunks k join note_versions v on v.id=k.version_id join note_paths p on p.id=v.path_id where k.embedding is not null and p.value like '%dsid_%'")
docs = []; vecs = []
for p, e in rows:
    docs.append(did(p)); vecs.append(np.frombuffer(e, dtype=np.float32))
V = np.stack(vecs); del vecs
print("chunks", V.shape, flush=True)
D = np.array(docs); uniq, inv = np.unique(D, return_inverse=True)
# whole-note vectors
wd = []; wv = []
for p, e in c.execute("select p.value, w.embedding from note_version_embeddings w join note_versions v on v.id=w.version_id join note_paths p on p.id=v.path_id where p.value like '%dsid_%'"):
    wd.append(did(p)); wv.append(np.frombuffer(e, dtype=np.float32))
W = np.stack(wv); wd = np.array(wd); print("whole", W.shape, flush=True)
TOP = 2000
dense = {}; whole = {}
for i, q in enumerate(qs):
    s = V @ Q[i]; best = np.full(len(uniq), -9.0, dtype=np.float32); np.maximum.at(best, inv, s)
    o = np.argsort(-best)[:TOP]; dense[q["question_id"]] = [str(uniq[j]) for j in o]
    s2 = W @ Q[i]; whole[q["question_id"]] = [str(wd[j]) for j in np.argsort(-s2)[:TOP]]
print("dense done", flush=True)
fts = "logs/fts.db"
if not os.path.exists(fts):
    f = sqlite3.connect(fts); f.execute("create virtual table d using fts5(id unindexed, title, body, tokenize='porter unicode61')")
    with f:
        for l in open("corpus50k.clean.jsonl"):
            r = json.loads(l); m = re.match(r'---\ntitle: (".*?")\n---\n\n(.*)', r["content"], re.S)
            t, b = (json.loads(m.group(1)), m.group(2)) if m else ("", r["content"])
            f.execute("insert into d values (?,?,?)", (did(r["path"]), t, b))
    f.close()
f = sqlite3.connect(fts)
STOP = set("a an and are as at be by can could did do does for from had has have how i if in into is it its of on or our s should so than that the their them then there these they this to was we were what when where which who why will with would you your".split())
def toks(s): return [t for t in re.findall(r"[a-z0-9]+", s.lower()) if t not in STOP and len(t) > 1]
bm_or = {}; bm_and = {}
for q in qs:
    t = list(dict.fromkeys(toks(q["question"])))
    def run(op):
        return [r[0] for r in f.execute("select id from d where d match ? order by bm25(d, 0, 5.0, 1.0) limit ?", (op.join(f'"{x}"' for x in t), TOP))] if t else []
    bm_or[q["question_id"]] = run(" OR "); bm_and[q["question_id"]] = run(" AND ")
print("bm25 done; AND nonempty", sum(1 for v in bm_and.values() if v), "of", len(bm_and), flush=True)
def rrf(a, b, k=60, cap=None, n=50):
    s = collections.Counter()
    for l in (a, b):
        for i, d in enumerate((l[:cap] if cap else l)): s[d] += 1 / (k + i + 1)
    return [d for d, _ in s.most_common(n)]
actual = {json.loads(l)["id"]: json.loads(l)["docs"] for l in open("logs/run-full-v1.jsonl")}
variants = {"trip2g actual": actual, "dense chunk maxsim": dense, "dense whole note": whole, "bm25 OR (fts5)": bm_or, "bm25 AND (fts5)": bm_and,
 "rrf dense50 + bm25 AND20": {q: rrf(dense[q], bm_and[q][:20], cap=None if False else 50) for q in dense},
 "rrf dense + bm25 OR": {q: rrf(dense[q][:200], bm_or[q][:200]) for q in dense},
 "rrf chunk + whole + OR": {q: [d for d, _ in (lambda s: s.most_common(50))(collections.Counter({d: sum(1 / (60 + l.index(d) + 1) for l in (dense[q][:200], whole[q][:200], bm_or[q][:200]) if d in l) for d in set(dense[q][:200] + whole[q][:200] + bm_or[q][:200])}))] for q in dense}}
types = ["ALL", "basic", "semantic", "completeness", "project_related", "conflicting_info", "constrained", "intra_document_reasoning", "miscellaneous"]
def score(run, k, t):
    x = [len(set(q["expected_doc_ids"]) & set(run.get(q["question_id"], [])[:k])) / len(q["expected_doc_ids"]) for q in qs if q["expected_doc_ids"] and (t == "ALL" or q["question_type"] == t)]
    return sum(x) / len(x)
def mrr(run):
    x = []
    for q in qs:
        g = set(q["expected_doc_ids"])
        if g: x.append(next((1 / (i + 1) for i, d in enumerate(run.get(q["question_id"], [])) if d in g), 0))
    return sum(x) / len(x)
print(f"{'variant':28}{'R@1':>7}{'R@5':>7}{'R@10':>7}{'R@20':>7}{'R@50':>7}{'MRR':>7}{'sem R10':>9}{'bas R10':>9}{'cmp R10':>9}")
for n, r in variants.items():
    print(f"{n:28}{score(r,1,'ALL'):7.3f}{score(r,5,'ALL'):7.3f}{score(r,10,'ALL'):7.3f}{score(r,20,'ALL'):7.3f}{score(r,50,'ALL'):7.3f}{mrr(r):7.3f}{score(r,10,'semantic'):9.3f}{score(r,10,'basic'):9.3f}{score(r,10,'completeness'):9.3f}")
ranks = []
for q in qs:
    for g in q["expected_doc_ids"]:
        l = dense[q["question_id"]]; ranks.append(l.index(g) + 1 if g in l else 99999)
ranks.sort(); n = len(ranks)
print("dense gold rank pctl p50", ranks[n//2], "p75", ranks[n*3//4], "p90", ranks[n*9//10], "share>50", round(sum(r > 50 for r in ranks)/n, 3), "share>2000", round(sum(r > 2000 for r in ranks)/n, 3))
json.dump({k: v for k, v in variants.items() if k != "trip2g actual"}, open("logs/variants.json", "w"))
