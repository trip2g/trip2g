---
free: true
title: Datachart — inline source
---

# Datachart: inline source

Data bundled right in the block via `data.source = "inline"`. No fetch, no asset
— the simplest case for small static charts.

```datachart
{
  "data": {
    "source": "inline",
    "rows": [
      { "month": "Jan", "sales": 120 },
      { "month": "Feb", "sales": 200 },
      { "month": "Mar", "sales": 150 },
      { "month": "Apr", "sales": 280 },
      { "month": "May", "sales": 240 }
    ]
  },
  "config": {
    "xAxis": { "type": "category" },
    "yAxis": { "type": "value" },
    "series": [{ "type": "bar", "encode": { "x": "month", "y": "sales" } }]
  }
}
```
