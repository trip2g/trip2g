---
title: "Token economy: measured on this site"
free: true
lang_redirect: "[[ru/user/token-economy-bench]]"
chart: "[[token_economy_bench.datachart.csv]]"
---

**In short:** reading the one section that holds the answer is cheaper than dumping the whole note. Across these questions it is **15× cheaper at the median**. The numbers below come from the live server behind this site, the [[mcp|MCP]] interface an AI agent uses to read the docs. There is a short, dependency-free script at the bottom: copy it, run it, get your own table.

```datachart
{
  "data": { "source": "frontmatter", "ref": "chart" },
  "config": {
    "title": { "text": "Tokens to read the answer (less is better)" },
    "tooltip": { "trigger": "axis" },
    "legend": {},
    "xAxis": { "type": "category", "name": "question" },
    "yAxis": { "type": "log", "name": "tokens (approx)" },
    "series": [
      { "type": "bar", "name": "whole note", "encode": { "x": "query", "y": "whole_note" } },
      { "type": "bar", "name": "focused read", "encode": { "x": "query", "y": "focused_read" } }
    ]
  }
}
```

### What we compared

An agent asks a question and gets the answer from the knowledge base. There are two ways to read it, and we measured both:

- **Whole note**, the anti-pattern. The agent finds the note and loads all of it by its id: `note_html(pid)`. It pays for the entire text even when the answer is a single paragraph.
- **Focused section**, the right way. The search result already carries a `toc_path`, a pointer straight at the relevant heading. The agent reads only that section: `note_html(pid, toc_path=[...])`.

The route is short: `search` → `note_html(toc_path)`. No tree-walking needed, because the pointer from the match leads straight to the target. When you want to inspect the structure first, [[expand]] sits in between, but for this benchmark that is an extra step.

**Questions.** Eight real queries against this documentation: webhooks, publishing to Telegram, custom domains, multiple languages, sync, templates, subscriptions, and Telegram limits. Not cherry-picked for a pretty result, just the ones people actually ask.

**Tokens.** Counted with a simple, transparent tokenizer: words plus punctuation (`\w+|[^\w\s]`). It is not Claude's tokenizer, so the absolute numbers are approximate. But both arms are measured with the same ruler, so the **ratio**, how many times cheaper, is honest.

### Results

| Question | Whole note | Focused section | Savings |
|---|--:|--:|--:|
| templates | 7668 | 206 | 37.2× |
| custom domain | 2698 | 155 | 17.4× |
| publish to Telegram | 3108 | 201 | 15.5× |
| two-way sync | 2529 | 163 | 15.5× |
| webhooks | 4654 | 302 | 15.4× |
| multilingual | 2234 | 185 | 12.1× |
| Telegram limits | 3108 | 277 | 11.2× |
| subscriptions | 920 | 248 | 3.7× |

**Median: 15.4×.** The win scales with note size. Long notes save the most: the templates page is 7668 tokens whole versus 206 in the right section, a 37× cut. Short notes save less: the subscriptions note is only 920 tokens to begin with, so just 4×. The logic is simple: the fatter the note, the more dead weight you skip.

One thing to be clear about: the baseline. 15× is against the naive dump of the whole note, not against a well-tuned RAG setup. If the agent already has good chunk retrieval, the gap is smaller. We measure what an agent does with no hints: it finds the note and reads all of it.

The numbers are live and taken from the current vault, so they drift over time: the docs grow, notes get fatter, individual rows shift. The chart above is a snapshot from 22 June 2026. The median stays around 15×.

### Why not just grep the files?

A fair question: why use MCP at all if you can search the notes with plain `grep`? We measured that too. A separate AI agent answered each question by grepping the local `.md` files and reading the hits.

Same 8 questions, same token count. Here is how it landed:

| Method | Tokens to answer | Tool calls |
|---|--:|--:|
| MCP, focused section | ~200 | 2 |
| MCP, whole note | ~2,700 | 2 |
| grep files + read | ~4,600 | 3 |

Grep over files costs **23× more** than the focused section, and even 1.7× more than pulling the whole note through MCP. The gap is not in the number of calls, which are nearly even, 3 vs 2. It is in volume. `grep` finds *files*, not *section boundaries*, so the agent reads whole files. `dev/obsidian_sync.md` alone is 6,144 tokens. Every MCP call returns the answer; every `grep` returns a list of files still to be read.

Honest about the measurement itself: this is naive `grep` that reads whole files, a single run, over a local repo with en+ru duplicates. Not lab-clean. A skilled person would narrow it with `grep -n`, then read the relevant lines instead of the whole file, and the gap would shrink. We are not claiming `grep` is bad.

It comes down to who is paying. For you, `grep` over local notes is fast and needs no setup. But an agent pays for every token, and a 23× gap is real money and a cluttered context. And when the base is large, shared, or remote, `grep` cannot reach it at all. There, MCP is the only way in. More in [[expand]].

### The easy half and the hard half

Saving the tokens is the easy half, almost self-evident: read less, pay less. The hard, interesting half is **landing on exactly the right section**.

You cannot guess the section blindly: a literal string match often sits somewhere other than where the answer lives. That is why every chunk in the trip2g index carries a breadcrumb, a chain of headings that doubles as a precise `toc_path`. Search returns a concrete path in the tree, not just "somewhere around here".

How precise? We checked: of the six questions where there is something to verify, the focused section contained the highlighted answer terms in four. The two misses were a highlighted stop word ("set" from "set up") and a case where the pointer landed on a neighbouring section. The answer was not lost, but this is the hard half: landing on exactly the right section, not next to it.

How that pointer works is in [[Fuzzy Pointer]]. When you want to inspect a note's structure and descend by hand, that is [[expand]]. Why pull a slice instead of the whole file at all is in [[Token Economy]].

### Check it yourself

No need to clone the repo and hunt for a script. Copy the block below into a file called `te_check.py`. All you need is Python 3 and an internet connection, no `pip install`:

```python
#!/usr/bin/env python3
# Token-economy check: focused section vs whole note, against a live trip2g MCP endpoint.
# Pure Python 3 standard library — no pip install. Run: python3 te_check.py
import json, re, statistics, urllib.request

ENDPOINT = "https://trip2g.com/_system/mcp"   # point this at any trip2g instance
QUERIES = [
    "what templates are available",
    "how do webhooks work",
    "how do i publish a post to telegram",
    "telegram post types and limits",
    "set up a custom domain for my site",
    "two way sync between obsidian and the site",
    "how to use multiple languages on my site",
    "accept paid subscriptions and monetization",
]

TOK = re.compile(r"\w+|[^\w\s]", re.UNICODE)
def tokens(text):                       # rough, tokenizer-independent — same count for both arms
    return len(TOK.findall(text or ""))

def mcp(name, args):                    # one MCP tool call over JSON-RPC 2.0
    body = json.dumps({"jsonrpc": "2.0", "id": 1, "method": "tools/call",
                       "params": {"name": name, "arguments": args}}).encode()
    req = urllib.request.Request(ENDPOINT, data=body,
        headers={"Content-Type": "application/json"}, method="POST")
    with urllib.request.urlopen(req, timeout=30) as r:
        return json.loads(r.read())["result"]

def note_html(res):
    content = res.get("content") or []
    return content[0]["text"] if content else ""

ratios = []
print(f"{'question':42}{'whole':>7}{'section':>9}{'saved':>8}")
for q in QUERIES:
    results = (mcp("search", {"query": q}).get("structuredContent") or {}).get("results") or []
    if not results:
        print(f"{q[:42]:42}  (no match)"); continue
    top = results[0]
    pid = top["note_id"]
    toc_path = (top.get("matches") or [{}])[0].get("toc_path")   # search points straight at the section
    whole = tokens(note_html(mcp("note_html", {"pid": pid})))
    section = tokens(note_html(mcp("note_html", {"pid": pid, "toc_path": toc_path}))) if toc_path else whole
    ratios.append(whole / section)
    print(f"{q[:42]:42}{whole:7d}{section:9d}{whole / section:7.1f}x")

print(f"\nmedian: {statistics.median(ratios):.1f}x   (whole note / focused section)")
```

Run it:

```sh
python3 te_check.py
```

What you'll see is a table with two token columns and a savings column, plus a median at the end:

```
question                                    whole  section   saved
what templates are available                 7668      206   37.2x
how do webhooks work                         4654      302   15.4x
...
median: 15.4x   (whole note / focused section)
```

If every row prints `(no match)` or the script fails with a network error, check that `ENDPOINT` opens in a browser and needs no login.

For your own instance, change `ENDPOINT` to your MCP server's address. For your own questions, edit the `QUERIES` list.

This script measures token economy only. The full version also checks that the section actually contains the answer and can walk the heading tree via `expand`. It lives in the repo: [`scripts/expand_check.py`](https://github.com/trip2g/trip2g/blob/main/scripts/expand_check.py).

### Related

- [[Token Economy]]: why pull a section instead of the whole file
- [[Fuzzy Pointer]]: how search finds the exact `toc_path`
- [[expand]]: layer-by-layer table-of-contents navigation
- [[mcp|MCP server]]: all methods and access
