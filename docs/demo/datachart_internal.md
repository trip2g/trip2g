---
free: true
title: Datachart — internal source
chart_ttl: 30m
---

# Datachart: internal source

A dashboard over trip2g's **own content**. `data.source = "internal"` runs the
SQL against the filtered read replica (only allowlisted tables exist there, so it
can never reach `secrets`/`users`). Here: notes grouped by their frontmatter
`status`, via the `note_version_frontmatters` JSON index.

```datachart
{
  "data": {
    "source": "internal",
    "sql": "SELECT json_extract(data,'$.status') AS status, count(*) AS n FROM note_version_frontmatters GROUP BY status"
  },
  "config": {
    "series": [{ "type": "pie", "encode": { "itemName": "status", "value": "n" } }]
  }
}
```
