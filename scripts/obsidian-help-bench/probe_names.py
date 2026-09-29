#!/usr/bin/env python3
"""Which field names do small models fill best? Three naming hypotheses, same fixed
descriptions (answer written as the note states it; knowledge-base language named).
Looks only at the first tool call.   usage: OPENROUTER_KEY=... probe_names.py openai/gpt-5.4-mini openai/gpt-5.4-nano anthropic/claude-haiku-4.5
"""
import json
import os
import random
import re
import sys
import urllib.request

from common import WORK, queries

KEY = os.environ["OPENROUTER_KEY"]
LANG_NAME = {"ru": "Russian", "en": "English"}

HYPOTHESES = {
    "H1": ("short_query", "rephrased_query", "expected_answer"),
    "H2": ("short_query", "rephrased_query", "answer_excerpt"),
    "H3": ("lex_query", "vec_query", "hyde_passage"),
}


def schema(names, lang):
    short, reph, ans = names
    L = LANG_NAME[lang]
    return {"type": "function", "function": {
        "name": "search",
        "description": f"Search notes in the knowledge base (Obsidian's help documentation, written in {L}). "
                       "Returns matching notes with snippets. The optional fields each run an extra search "
                       "whose results are merged with the main one; fill them to find more.",
        "parameters": {"type": "object", "required": ["query"], "properties": {
            "query": {"type": "string", "description": "The user's question exactly as they asked it. Do not rewrite it."},
            short: {"type": "string", "description":
                f"Optional. 2-3 words in {L} that the answering note is sure to contain: the main term and one "
                "qualifier, in dictionary form (nominative singular, infinitive). A short phrase, not a list; "
                "no punctuation. Keyword search needs every word to match, so fewer precise words beat many."},
            reph: {"type": "string", "description":
                f"Optional. The question asked another way, in {L}: different words, and the feature's canonical "
                "name if you know it. One sentence."},
            ans: {"type": "string", "description":
                f"Optional. One or two sentences in {L} that the answering note itself could contain, written the "
                "way documentation states facts (e.g. 'To pin a note, right-click its tab and select Pin.'). "
                "Not a description of what the note covers, not a question. If unsure of details, stay general "
                "but keep it a statement of fact."},
        }}}}


NOTE_HTML = {"type": "function", "function": {
    "name": "note_html", "description": "Read a note in full by its path from a search result.",
    "parameters": {"type": "object", "required": ["path"], "properties": {"path": {"type": "string"}}}}}
SYSTEM = ("You answer questions about Obsidian using ONLY the knowledge base reached through the given "
          "tools. Search first, then read what you need.")
META = re.compile(r"(documentation|the note|the guide|should (explain|describe)|explains? (how|that|whether)|"
                  r"describes|документац|справк|заметк\w* (описыва|объясня)|должн\w* (объясн|описа)|описани\w* того)", re.I)
CYR = re.compile("[а-яё]", re.I)


def pick(lang, n):
    qs = queries(lang)
    random.Random(7).shuffle(qs)
    return qs[:n]


def first_call(model, tools, question):
    body = json.dumps({"model": model, "temperature": 0, "tool_choice": "auto", "tools": tools,
                       "messages": [{"role": "system", "content": SYSTEM}, {"role": "user", "content": question}],
                       "usage": {"include": True}}).encode()
    req = urllib.request.Request("https://openrouter.ai/api/v1/chat/completions", body, {
        "Authorization": f"Bearer {KEY}", "Content-Type": "application/json",
        "HTTP-Referer": "https://trip2g.com", "X-Title": "trip2g-field-probe"})
    d = json.load(urllib.request.urlopen(req, timeout=180))
    calls = d["choices"][0]["message"].get("tool_calls") or []
    args = json.loads(calls[0]["function"]["arguments"]) if calls else None
    return args, float((d.get("usage") or {}).get("cost") or 0)


print(f"{'model':24s} {'hyp':3s} {'call':>4s} {'short':>5s} {'reph':>4s} {'ans':>4s} {'words':>5s} {'1word':>5s} {'lang!':>5s} {'meta':>4s} {'q-kept':>6s}")
total = 0.0
for model in sys.argv[1:]:
    for hyp, names in HYPOTHESES.items():
        rows = []
        for lang in ("ru", "en"):
            for q in pick(lang, 10):
                args, c = first_call(model, [schema(names, lang), NOTE_HTML], q["query"])
                total += c
                rows.append({"lang": lang, "id": q["id"], "question": q["query"], "args": args})
        os.makedirs(WORK, exist_ok=True)
        json.dump(rows, open(os.path.join(WORK, f"probe-{hyp}-{model.split('/')[-1]}.json"), "w"),
                  ensure_ascii=False, indent=1)
        short, reph, ans = names
        got = [r for r in rows if r["args"]]
        shorts = [r for r in got if r["args"].get(short)]
        words = [len(r["args"][short].split()) for r in shorts]
        lang_bad = sum(1 for r in got for f in names if r["args"].get(f)
                       and bool(CYR.search(r["args"][f])) != (r["lang"] == "ru"))
        meta = sum(1 for r in got if r["args"].get(ans) and META.search(r["args"][ans]))
        kept = sum(1 for r in got if r["args"].get("query", "").strip() == r["question"].strip())
        print(f"{model.split('/')[-1]:24s} {hyp:3s} {len(got):4d} {len(shorts):5d} "
              f"{sum(1 for r in got if r['args'].get(reph)):4d} {sum(1 for r in got if r['args'].get(ans)):4d} "
              f"{(sum(words)/len(words) if words else 0):5.1f} {sum(1 for w in words if w < 2):5d} "
              f"{lang_bad:5d} {meta:4d} {kept:6d}", flush=True)
print(f"total cost ${total:.3f}")
