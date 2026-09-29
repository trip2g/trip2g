#!/usr/bin/env python3
"""Have a real (small) agent model fill the optional search fields for every query.

The model gets a `search` tool whose optional fields carry one of two description
variants and a user question; its first tool call is recorded. Output has the
agent-fields format consumed by `agent_fields.py --fields`.

  v1  names and descriptions as chosen by probe_names.py (H1)
  v3  v1 with short_query asking for the feature's name (1-2 words, as a page title would
      say it) instead of words from the question; see the mini runs in the thoughts article
  v2  v1 plus rules adapted from qmd's query-expansion reward (finetune/SCORING.md):
      keep names and technical tokens verbatim, no filler, don't repeat the question,
      one-line 50-200 char answer, a good and a bad example per field. qmd's quoting and
      -negation are left out: trip2g's full-text lane doesn't parse them.

usage: OPENROUTER_KEY=... agent_fields_gen.py <v1|v2|v3> <model> [lang...]
writes $OHB_WORK/fields-<variant>-<model>-<lang>.json
"""
import concurrent.futures
import json
import os
import sys
import urllib.request

from common import WORK, queries

KEY = os.environ["OPENROUTER_KEY"]
LANG_NAME = {"ru": "Russian", "en": "English"}
SYSTEM = ("You answer questions about Obsidian using ONLY the knowledge base reached through the given "
          "tools. Search first, then read what you need.")


def descriptions(variant, L):
    if variant == "v1":
        return {
            "short_query": f"Optional. 2-3 words in {L} that the answering note is sure to contain: the main term and one "
                           "qualifier, in dictionary form (nominative singular, infinitive). A short phrase, not a list; "
                           "no punctuation. Keyword search needs every word to match, so fewer precise words beat many.",
            "rephrased_query": f"Optional. The question asked another way, in {L}: different words, and the feature's "
                               "canonical name if you know it. One sentence.",
            "expected_answer": f"Optional. One or two sentences in {L} that the answering note itself could contain, written "
                               "the way documentation states facts (e.g. 'To pin a note, right-click its tab and select Pin.'). "
                               "Not a description of what the note covers, not a question. If unsure of details, stay "
                               "general but keep it a statement of fact.",
        }
    if variant == "v3":
        d = descriptions("v1", L)
        d["short_query"] = (f"Optional. The name of the feature, setting or concept the question is about, in {L}, 1-2 "
                            "words, the way it would appear as the title of a help page (e.g. 'Tags', 'Graph view', "
                            "'Publish', 'Встраивание файлов'). Name the thing; don't copy words from the question and "
                            "don't add qualifiers. Keyword search needs every word to match.")
        return d
    return {
        "short_query": f"Optional. 2-3 words in {L} that the answering note is sure to contain: the main term plus one "
                       "qualifier, in dictionary form (nominative singular, infinitive). Keep names of products, features "
                       "and technical tokens from the question exactly as written (Obsidian Sync, iCloud, {{title}}, $$). "
                       "No filler words (how, find, guide, information about), no punctuation, and never just the question "
                       "repeated. Keyword search needs every word to match, so fewer precise words beat many. "
                       "Good: 'синхронизация iCloud', 'file recovery snapshot'. Bad: 'как настроить' (filler), "
                       "'перенести' (one vague word), 'guide to sync' (filler).",
        "rephrased_query": f"Optional. The question asked another way, in {L}: a complete phrase with different words than "
                           "the question, using the feature's canonical name if you know it. Keep product names and "
                           "technical tokens. Good: 'restore a deleted note from File recovery snapshots'. "
                           "Bad: the question repeated, or a bare list of keywords.",
        "expected_answer": f"Optional. One line of 50-200 characters in {L} that the answering note itself could contain, "
                           "stated the way documentation states facts. Good: 'To restore a deleted note, open File recovery "
                           "in Settings and choose a snapshot.' Bad: 'The documentation should explain how to restore notes' "
                           "(describes the answer instead of stating it).",
    }


def tools(variant, lang):
    L = LANG_NAME[lang]
    props = {"query": {"type": "string", "description": "The user's question exactly as they asked it. Do not rewrite it."}}
    props.update({k: {"type": "string", "description": v} for k, v in descriptions(variant, L).items()})
    return [{"type": "function", "function": {
        "name": "search",
        "description": f"Search notes in the knowledge base (Obsidian's help documentation, written in {L}). "
                       "Returns matching notes with snippets. The optional fields each run an extra search whose "
                       "results are merged with the main one; fill them to find more.",
        "parameters": {"type": "object", "required": ["query"], "properties": props}}},
        {"type": "function", "function": {
            "name": "note_html", "description": "Read a note in full by its path from a search result.",
            "parameters": {"type": "object", "required": ["path"], "properties": {"path": {"type": "string"}}}}}]


def first_call(model, tl, question):
    body = json.dumps({"model": model, "temperature": 0, "tool_choice": "auto", "tools": tl,
                       "messages": [{"role": "system", "content": SYSTEM}, {"role": "user", "content": question}],
                       "usage": {"include": True}}).encode()
    req = urllib.request.Request("https://openrouter.ai/api/v1/chat/completions", body, {
        "Authorization": f"Bearer {KEY}", "Content-Type": "application/json",
        "HTTP-Referer": "https://trip2g.com", "X-Title": "trip2g-field-gen"})
    for attempt in range(3):
        try:
            d = json.load(urllib.request.urlopen(req, timeout=180))
            break
        except Exception:
            if attempt == 2:
                raise
    calls = [c for c in (d["choices"][0]["message"].get("tool_calls") or []) if c["function"]["name"] == "search"]
    args = json.loads(calls[0]["function"]["arguments"]) if calls else {}
    return args, float((d.get("usage") or {}).get("cost") or 0)


variant, model = sys.argv[1], sys.argv[2]
langs = sys.argv[3:] or ["en", "ru"]
total = 0.0
for lang in langs:
    tl, qs = tools(variant, lang), queries(lang)
    with concurrent.futures.ThreadPoolExecutor(8) as pool:
        res = list(pool.map(lambda q: first_call(model, tl, q["query"]), qs))
    out = []
    for q, (args, cost) in zip(qs, res):
        total += cost
        out.append({"id": q["id"], "called": bool(args),
                    **{k: (args.get(k) or "").strip() for k in ("short_query", "rephrased_query", "expected_answer")}})
    path = os.path.join(WORK, f"fields-{variant}-{model.split('/')[-1]}-{lang}.json")
    json.dump(out, open(path, "w", encoding="utf-8"), ensure_ascii=False, indent=1)
    filled = {k: sum(1 for o in out if o[k]) for k in ("short_query", "rephrased_query", "expected_answer")}
    print(f"{variant} {model} {lang}: {len(out)} queries, called {sum(o['called'] for o in out)}, filled {filled} -> {path}")
print(f"cost ${total:.3f}")
