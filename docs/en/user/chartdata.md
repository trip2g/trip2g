---
title: "Charts in notes"
free: true
lang_redirect: "[[ru/user/chartdata]]"
chart_sales: "[[sales-demo.datachart.csv]]"
---

A `datachart` fenced code block in any note becomes an interactive chart on the published page. No admin panel, no drag-and-drop — you write the chart the same way you write any other code block in Obsidian.

## How it works

Add a fenced block with the language identifier `datachart`. The block contains a single JSON object with two keys:

- `data` — where the rows come from
- `config` — an [Apache ECharts](https://echarts.apache.org/en/option.html) option object that controls the chart type and appearance

```jsonc
{
  "data": { "source": "inline", "rows": [ ... ] },
  "config": {
    "xAxis": { "type": "category" },
    "yAxis": { "type": "value" },
    "series": [{ "type": "bar", "encode": { "x": "month", "y": "sales" } }]
  }
}
```

trip2g reads the `data` object, fetches or reads the rows, and injects them into `config.dataset.source` automatically. You do not write a placeholder for the data — it just appears. The ECharts widget on the page then draws the chart from the result.

One block = one chart. Place as many blocks as you need in a note; they render in document order.

---

## Live examples

The charts below are rendered from `datachart` blocks on this very page — each is just a few lines of markdown with the data bundled in.

```datachart
{
  "data": { "source": "inline", "rows": [
    { "month": "Jan", "sales": 120 },
    { "month": "Feb", "sales": 200 },
    { "month": "Mar", "sales": 150 },
    { "month": "Apr", "sales": 280 },
    { "month": "May", "sales": 240 }
  ]},
  "config": {
    "title": { "text": "Monthly sales (bar)" },
    "xAxis": { "type": "category" },
    "yAxis": { "type": "value" },
    "series": [{ "type": "bar", "encode": { "x": "month", "y": "sales" } }]
  }
}
```

```datachart
{
  "data": { "source": "inline", "rows": [
    { "day": "Mon", "visitors": 40 },
    { "day": "Tue", "visitors": 65 },
    { "day": "Wed", "visitors": 52 },
    { "day": "Thu", "visitors": 88 },
    { "day": "Fri", "visitors": 71 }
  ]},
  "config": {
    "title": { "text": "Visitors per day (line)" },
    "xAxis": { "type": "category" },
    "yAxis": { "type": "value" },
    "series": [{ "type": "line", "encode": { "x": "day", "y": "visitors" } }]
  }
}
```

```datachart
{
  "data": { "source": "inline", "rows": [
    { "source": "direct", "hits": 30 },
    { "source": "search", "hits": 50 },
    { "source": "social", "hits": 20 }
  ]},
  "config": {
    "title": { "text": "Traffic sources (pie)" },
    "series": [{ "type": "pie", "encode": { "itemName": "source", "value": "hits" } }]
  }
}
```

## Data sources

The `data.source` field tells trip2g where to get the rows. There are four options.

### `inline` — data bundled in the block

The simplest case. All rows live right inside the block. Use this for small static charts that don't change.

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

No fetch happens. The rows are part of the note itself.

### `frontmatter` — data from a vault file

The data lives in a `.csv` or `.json` file inside your vault. You reference it from the note's **frontmatter**, not inside the code block.

Why frontmatter? Because Obsidian gives frontmatter `[[links]]` real link support — autocomplete, graph edges, highlighting in the editor. A link buried inside a code fence gets none of that.

Add the link as a frontmatter property, then point the block at that property by name:

```yaml
---
chart_sales: "[[sales-demo.datachart.csv]]"
---
```

The chart below is live — this very page declares `chart_sales` in its own
frontmatter and ships a small `sales-demo.datachart.csv` next to it:

```datachart
{
  "data": { "source": "frontmatter", "ref": "chart_sales" },
  "config": {
    "title": { "text": "Monthly sales (from a vault CSV)" },
    "xAxis": { "type": "category" },
    "yAxis": { "type": "value" },
    "series": [{ "type": "bar", "encode": { "x": "month", "y": "sales" } }]
  }
}
```

When a reader opens the page, the browser fetches the data file directly — trip2g resolves the wikilink to the asset's URL and puts it in the page HTML. The file is served with the same access control as the note itself (a free note serves it publicly; a paid note gates it behind subscription).

Both `.csv` and `.json` are supported. A JSON file must contain an array of row objects. A CSV file must have a header row; the widget parses it and coerces numeric columns automatically.

### `url` — data from an external HTTP endpoint

Live data from any HTTP-JSON endpoint you control. trip2g fetches the URL on the **server side**, caches the result, and embeds it in the rendered HTML. The endpoint URL and request body never appear in the page source.

The chart below is live and **copy-pasteable**: it points at a tiny demo endpoint
on this site. Copy the block into your own vault and it renders the same chart —
your trip2g fetches the same public URL.

```datachart
{
  "data": {
    "source": "url",
    "url": "https://trip2g.com/en/user/sales_demo_api"
  },
  "config": {
    "title": { "text": "Daily revenue (from a URL)" },
    "xAxis": { "type": "category" },
    "yAxis": { "type": "value" },
    "series": [{ "type": "line", "encode": { "x": "day", "y": "revenue" } }]
  }
}
```

The endpoint must return a flat JSON array of row objects — exactly what
`https://trip2g.com/en/user/sales_demo_api` returns:

```json
[
  { "day": "2026-06-01", "revenue": 1234 },
  { "day": "2026-06-02", "revenue": 1891 }
]
```

Need to send a query? Add a `body` and trip2g POSTs it as JSON instead of a plain
GET. This is a syntax example — point it at your own endpoint:

```jsonc
{
  "data": {
    "source": "url",
    "url": "https://api.example.com/query",
    "body": "{\"sql\":\"SELECT day, revenue FROM stats ORDER BY day\"}"
  },
  "config": {
    "xAxis": { "type": "category" },
    "yAxis": { "type": "value" },
    "series": [{ "type": "line", "encode": { "x": "day", "y": "revenue" } }]
  }
}
```

**When the data is fetched.** trip2g fetches the URL server-side when the note is **published or rebuilt** (during the note load) — not when a reader opens the page. Readers always get pre-rendered HTML. While the first fetch is in flight the chart shows a loading indicator; once it lands, the page re-renders with the data. The cached result is then reused until the note is re-published.

**Auth headers** (coming soon). Endpoints that need an auth header — an API key or token — will be supported through encrypted server-side secrets, so the credentials never reach the browser.

### `internal` — query your site's own content (coming soon)

`data.source = "internal"` with a `sql` field will let you chart over your own published content — notes by status, publications over time, and similar. This data source is not yet available; the chart renders a loading indicator on the live site.

---

## Frontmatter options

Two optional frontmatter properties control chart behavior at the note level.

**`chart_ttl`** (refresh cadence — coming soon)

The intended refresh interval for `url`/`internal` data, as a Go duration: `30m`, `1h`, `24h`. Scheduled re-fetch is not live yet — for now the data is fetched once when the note is published and refreshes only when the note is re-published.

```yaml
chart_ttl: 1h
```

**`charts: custom`**

Suppresses inline chart rendering. The charts are still parsed and available to a custom template via `ctx.Charts()`, so you can position them yourself within the page layout.

```yaml
charts: custom
```

This is an advanced option for custom templates. Most authors do not need it.

---

## Chart types

The chart type comes from `config.series[].type`. The three most common:

| Type | `series[].type` | Notes |
|------|-----------------|-------|
| Bar | `"bar"` | Use `encode: { x: "col", y: "col" }` to map columns |
| Line | `"line"` | Same encode pattern as bar |
| Pie | `"pie"` | Use `encode: { itemName: "col", value: "col" }` |

All ECharts chart types and options are available — see the [ECharts option reference](https://echarts.apache.org/en/option.html) for the full list. The `config` object is passed directly to ECharts, so anything that works in ECharts works here.

---

## Worked example: bar, line, and pie in one note

The following note uses all three types from different data sources:

```yaml
---
free: true
title: Traffic overview
chart_ttl: 1h
---
```

```datachart
{
  "data": {
    "source": "inline",
    "rows": [
      { "product": "Coffee", "revenue": 4200 },
      { "product": "Tea", "revenue": 3100 },
      { "product": "Cocoa", "revenue": 1800 },
      { "product": "Juice", "revenue": 1500 },
      { "product": "Water", "revenue": 900 }
    ]
  },
  "config": {
    "title": { "text": "Revenue by product (bar)" },
    "xAxis": { "type": "category" },
    "yAxis": { "type": "value" },
    "series": [{ "type": "bar", "encode": { "x": "product", "y": "revenue" } }]
  }
}
```

```datachart
{
  "data": {
    "source": "url",
    "url": "https://trip2g.com/en/user/sales_demo_api"
  },
  "config": {
    "title": { "text": "Daily revenue (line, from a URL)" },
    "xAxis": { "type": "category" },
    "yAxis": { "type": "value" },
    "series": [{ "type": "line", "encode": { "x": "day", "y": "revenue" } }]
  }
}
```

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
    "title": { "text": "Traffic sources (pie)" },
    "series": [{ "type": "pie", "encode": { "itemName": "source", "value": "hits" } }]
  }
}
```
