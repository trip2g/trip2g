---
free: true
title: Datachart — url source (live query)
chart_ttl: 1h
---

# Datachart: url source

Live data from an HTTP-JSON endpoint. trip2g fetches it on the backend, caches it
by note version, and refreshes on the `chart_ttl`. `config` carries a `"$data"`
placeholder where the fetched rows are injected.

```datachart
{
  "data": {
    "source": "url",
    "url": "http://localhost:8090/v1/query",
    "body": "{\"sql\":\"SELECT day, revenue FROM stats ORDER BY day\"}"
  },
  "config": {
    "xAxis": { "type": "category" },
    "yAxis": { "type": "value" },
    "series": [{ "type": "line", "encode": { "x": "day", "y": "revenue" } }]
  }
}
```

