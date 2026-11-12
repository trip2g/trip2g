#!/usr/bin/env python3
# usage: answers.py <questions> <head|chunks> <label:runfile> ...
# Answers a per-category sample of the questions from each run's retrieved documents and
# scores the answers with the benchmark's own judge prompts (answer_evaluation_prompts.py,
# copied from EnterpriseRAG-Bench src/prompts/answer_evaluation.py).
#   head    first CHARS characters of the top TOPK documents (what the published numbers used)
#   chunks  the chunks of the top DOCS documents closest to the question, up to BUDGET characters
# Set DRY=1 to print context sizes without calling the model.
import collections, concurrent.futures as cf, json, random, re, sys, threading, time, urllib.request
limit = int(sys.argv[1]); mode = sys.argv[2]; runs = [a.split(":") for a in sys.argv[3:]]
import os
DRY = os.environ.get("DRY") == "1"; DOCS = 10; BUDGET = 24000
MODEL = "anthropic/claude-haiku-5.5"; CAP = 0.50; TOPK = 5; CHARS = 4000
K = "" if DRY else open(".key").read().strip(); ns = {}; exec(open("answer_evaluation_prompts.py").read(), ns)
W, F = ns["ANSWER_WHOLISTIC_EVALUATION_PROMPT"], ns["INDIVIDUAL_FACT_VALIDATOR_PROMPT"]
did = lambda p: re.search(r"dsid_[0-9a-f]+", p)[0]
docs = {did(json.loads(l)["path"]): json.loads(l)["content"] for l in open("corpus50k.clean.jsonl")}
qs = [json.loads(l) for l in open("questions.jsonl")]
by = collections.defaultdict(list)
for q in qs: by[q["question_type"]].append(q)
rnd = random.Random(11); pick = []
for t, l in sorted(by.items()):
    rnd.shuffle(l); pick += l[:max(2, round(len(l) * limit / 500))]
print("questions", len(pick), flush=True)
lock = threading.Lock(); spent = [0.0]; toks = [0, 0]
def chat(prompt, tokens=3000):
    b = json.dumps({"model": MODEL, "messages": [{"role": "user", "content": prompt}], "max_tokens": tokens, "usage": {"include": True}}).encode()
    for a in range(5):
        if spent[0] > CAP: raise SystemExit("CAP")
        try:
            r = json.load(urllib.request.urlopen(urllib.request.Request("https://openrouter.ai/api/v1/chat/completions", b, {"Authorization": "Bearer " + K, "Content-Type": "application/json"}), timeout=120))
            u = r.get("usage", {})
            with lock: spent[0] += float(u.get("cost") or 0); toks[0] += u.get("prompt_tokens", 0); toks[1] += u.get("completion_tokens", 0)
            return r["choices"][0]["message"]["content"] or ""
        except urllib.error.HTTPError as e:
            if e.code == 402: raise SystemExit("402 no credit")
            time.sleep(3 * (a + 1))
        except Exception: time.sleep(3 * (a + 1))
    return ""
def yes(text, key):
    m = re.search(r"\{.*\}", text, re.S)
    if not m: return None
    try: v = json.loads(m.group(0))
    except Exception: return None
    v = str(v.get(key, "")).lower()
    return "yes" in v or "true" in v
ANSWER = """Answer the question using only the documents below, in one to four sentences. If the documents do not contain the answer, say that the information was not found. Do not guess.

## Question
{q}

## Documents
{d}"""
if mode == "chunks":
    import sqlite3
    import numpy as np
    QV = np.load("logs/question-vectors.npy"); qi = {q["question_id"]: i for i, q in enumerate(qs)}
    by_doc = collections.defaultdict(list)
    c = sqlite3.connect("file:bench.sqlite3?mode=ro", uri=True, timeout=10)
    for p, e, t in c.execute("select p.value, k.embedding, k.content from note_version_chunks k join note_versions v on v.id=k.version_id join note_paths p on p.id=v.path_id where k.embedding is not null and p.value like '%dsid_%'"):
        by_doc[did(p)].append((np.frombuffer(e, dtype=np.float32), t))
def context(q, ids):
    if mode == "head":
        ids = ids[:TOPK]
        return ids, "\n\n".join(f"[{i+1}] {docs[d][:CHARS]}" for i, d in enumerate(ids) if d in docs)
    ids = ids[:DOCS]; v = QV[qi[q["question_id"]]]; cand = []
    for rank, d in enumerate(ids):
        for e, t in by_doc.get(d, []): cand.append((float(e @ v), rank, t))
    cand.sort(key=lambda x: -x[0]); out = []; used = 0
    for s_, rank, t in cand:
        if used + len(t) > BUDGET: continue
        out.append((rank, t)); used += len(t)
    out.sort(key=lambda x: x[0])
    return ids, "\n\n".join(f"[{r+1}] {t}" for r, t in out)
def one(q, ran):
    ids, ctx = context(q, ran.get(q["question_id"], []))
    if DRY: return {"id": q["question_id"], "type": q["question_type"], "answer": "", "correct": None, "facts": [], "hit": bool(set(q["expected_doc_ids"]) & set(ids)), "chars": len(ctx)}
    a = chat(ANSWER.format(q=q["question"], d=ctx))
    ok = yes(chat(W.format(query=q["question"], gold_answer=q["gold_answer"], candidate_answer=a), 200), "aligned")
    fs = [chat(F.format(answer=a, statement=f), 40).strip().lower().startswith("yes") for f in q["answer_facts"]]
    return {"id": q["question_id"], "type": q["question_type"], "answer": a, "correct": ok, "facts": fs, "hit": bool(set(q["expected_doc_ids"]) & set(ids))}
results = {}
for label, f in runs:
    ran = {json.loads(l)["id"]: json.loads(l)["docs"] for l in open(f)}
    with cf.ThreadPoolExecutor(8) as ex: out = list(ex.map(lambda q: one(q, ran), pick))
    results[label] = out; json.dump(out, open(f"logs/answers-{label}-{mode}.json", "w"), ensure_ascii=False)
    if DRY:
        ch = sorted(r["chars"] for r in out); g = [r for r in out if r["type"] not in ("high_level", "info_not_found")]
        print(label, mode, "context chars p50", ch[len(ch)//2], "max", ch[-1], "gold doc in context", round(100*sum(r["hit"] for r in g)/len(g), 1), flush=True)
    print(label, "spent", round(spent[0], 4), "tokens", toks, flush=True)
def pct(x): x = [v for v in x if v is not None]; return (100 * sum(map(bool, x)) / len(x)) if x else 0
types = ["ALL"] + sorted(by)
if DRY: raise SystemExit(0)
print(f"{'type':28}{'n':>4}" + "".join(f"{l+' corr':>13}{l+' facts':>13}" for l, _ in runs))
for t in types:
    row = f"{t:28}"; n = 0
    for l, _ in runs:
        o = [r for r in results[l] if t == "ALL" or r["type"] == t]; n = len(o)
        row += f"{pct([r['correct'] for r in o]):13.1f}{pct([v for r in o for v in r['facts']]):13.1f}"
    print(row[:28] + f"{n:4d}" + row[28:], flush=True)
print("TOTAL spent", round(spent[0], 4))
