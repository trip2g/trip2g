---
free: true
title: Datachart — multiple types & custom layout
charts: custom
---

# Datachart: several charts, custom placement

`charts: custom` in frontmatter suppresses inline auto-render — a custom template
iterates `ctx.Charts()` and positions them. Three types below exercise the widget,
each from a different `data.source`.

## Line (internal)

```datachart
{
  "data": {
    "source": "internal",
    "sql": "SELECT strftime('%Y-%W', created_at) AS week, count(*) AS n FROM note_versions GROUP BY week ORDER BY week"
  },
  "config": {
    "xAxis": { "type": "category" },
    "yAxis": { "type": "value" },
    "series": [{ "type": "line", "encode": { "x": "week", "y": "n" } }]
  }
}
```

## Bar (url)

```datachart
{
  "data": {
    "source": "url",
    "url": "http://localhost:8087/v1/query",
    "body": "{\"sql\":\"SELECT product, revenue FROM sales ORDER BY revenue DESC LIMIT 5\"}"
  },
  "config": {
    "xAxis": { "type": "category" },
    "yAxis": { "type": "value" },
    "series": [{ "type": "bar", "encode": { "x": "product", "y": "revenue" } }]
  }
}
```

## Pie (inline)

```datachart
{
  "data": {
    "source": "inline",
    "rows": [
      { "source": "direct", "hits": 30 },
      { "source": "search", "hits": 50 },
      { "source": "social", "hits": 20 }
    ]
  },
  "config": {
    "series": [{ "type": "pie", "encode": { "itemName": "source", "value": "hits" } }]
  }
}
```
