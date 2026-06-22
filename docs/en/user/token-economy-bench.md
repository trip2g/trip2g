---
title: "Token economy, measured on this site"
free: true
lang_redirect: "[[ru/user/token-economy-bench]]"
chart: "[[token_economy_bench.datachart.csv]]"
---

**In short:** reading only the section that holds the answer costs far fewer tokens than dumping the whole note — about **15× less** here (median). The agent lands on that section deterministically (the search match points straight at it), so the same question always reads the same section.

These numbers are live. We asked the trip2g MCP server real questions and compared two ways for an agent to read the answer: dump the **whole note**, or pull only the **focused section** that holds it. The gap is the token economy.

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

Across these questions the focused read costs about **15× fewer tokens** (median) than dumping the whole note. The win scales with note size: long notes (webhooks, multilingual) save the most, short notes (subscriptions) save less — a small note is already cheap.

The section is located deterministically: the search match already carries a `toc_path` pointing directly to the relevant heading, so the same question always lands on the same section. Drill-down goes `search` → `expand` → `note_html` — no tree-walking needed when the match pointer is present.

One honest caveat. Saving the tokens is the easy half. The hard, interesting half is **navigating to the exact section** — and that is what the note's table of contents is for: you read the top-level structure, then drill down, instead of searching every chunk. See [[Token Economy]] and [[Fuzzy Pointer]] for the mechanism.

**Reproduce it yourself.** No dependencies, just Python 3:

```sh
python3 scripts/expand_check.py
```

The script asks `https://trip2g.com/_system/mcp` the same questions and prints the table.
