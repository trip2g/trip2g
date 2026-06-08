---
free: true
title: Datachart — frontmatter data ref
chart_sales: "[[sales.datachart.csv]]"
chart_visitors: "[[visitors.json]]"
---

# Datachart: data from a frontmatter link

The data lives in a vault file, referenced from **frontmatter** — so Obsidian
gives it link highlighting, autocomplete, and a graph edge (which a link inside a
` ```datachart ` code block would not). The config stays in the block; `data`
just points at a frontmatter key.

## Sales (CSV via `chart_sales`)

```datachart
{
  "data": { "source": "frontmatter", "ref": "chart_sales" },
  "config": {
    "xAxis": { "type": "category" },
    "yAxis": { "type": "value" },
    "series": [{ "type": "bar", "encode": { "x": "month", "y": "sales" } }]
  }
}
```

## Visitors (JSON via `chart_visitors`)

```datachart
{
  "data": { "source": "frontmatter", "ref": "chart_visitors" },
  "config": {
    "xAxis": { "type": "category" },
    "yAxis": { "type": "value" },
    "series": [{ "type": "line", "encode": { "x": "day", "y": "visitors" } }]
  }
}
```
