---
title: "Token economy, measured on this site"
free: true
lang_redirect: "[[ru/user/token-economy-bench]]"
chart: "[[token_economy_bench.datachart.csv]]"
---

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

Across these questions the focused read costs about **11× fewer tokens** (median) than dumping the whole note. The win scales with note size: long notes (webhooks, multilingual) save the most, short notes (Telegram limits, subscriptions) save little — a small note is already cheap.

One honest caveat. Saving the tokens is the easy half. The hard, interesting half is **navigating to the exact section** — and that is what the note's table of contents is for: you read the top-level structure, then drill down, instead of searching every chunk. See [[Token Economy]] and [[Fuzzy Pointer]] for the mechanism.

**Reproduce it yourself.** No dependencies, just Python 3:

```sh
python3 scripts/token_economy_check.py
```

The script asks `https://trip2g.com/_system/mcp` the same questions and prints the table.
