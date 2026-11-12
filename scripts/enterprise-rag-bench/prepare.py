#!/usr/bin/env python3
# usage: prepare.py <all_documents.zip> <questions.jsonl> <distractors> <out.jsonl> [seed]
import collections, json, random, re, sys, zipfile
zp, qp, n, out = sys.argv[1], sys.argv[2], int(sys.argv[3]), sys.argv[4]
seed = int(sys.argv[5]) if len(sys.argv) > 5 else 42
gold = {d for l in open(qp) for d in json.loads(l)["expected_doc_ids"]}
z = zipfile.ZipFile(zp)
docs = []
for i in z.infolist():
    m = re.search(r"(dsid_[0-9a-f]+)", i.filename)
    if m:
        docs.append((i.filename.split("/")[0], m.group(1), i))
print("docs in zip", len(docs))
g = [d for d in docs if d[1] in gold]
rest = [d for d in docs if d[1] not in gold]
random.Random(seed).shuffle(rest)
pick = g + rest[:n]
print("gold found", len(g), "of", len(gold), "picked", len(pick))
print(collections.Counter(d[0] for d in pick).most_common())
total = 0
with open(out, "w") as f:
    for src, did, i in pick:
        text = z.read(i).decode("utf-8", "replace")
        head, _, body = text.partition("\n")
        title = head.strip()[:200] or did
        md = "---\ntitle: " + json.dumps(title, ensure_ascii=False) + "\n---\n\n" + body.lstrip("\n")
        total += len(md)
        f.write(json.dumps({"path": f"{src}/{did}.md", "content": md}, ensure_ascii=False) + "\n")
print("bytes", total, "est tokens x2", total // 2)
