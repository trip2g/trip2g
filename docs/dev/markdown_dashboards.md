# Markdown Dashboards (`datachart` blocks)

Status: **partially implemented** (2026-06-10) — inline/frontmatter/url work; see "Implementation status & remaining tasks" below.

Render charts directly from a fenced ` ```datachart ` code block in a note. trip2g
fetches the chart's data from an HTTP-JSON endpoint, caches the result in SQLite
keyed by the note version, and the published page draws it with ECharts. No
admin UI, no drag-and-drop in the MVP — charts render inline, in document order,
exactly like any other fenced block renders today.

## Implementation status & remaining tasks (2026-06-10)

**Done & live:**
- Parsing: `extractCharts` → typed `data:{source}` model (`internal/model/note_chart.go`).
- Renderer: `chartRenderer` emits `<div class="chart">` + `<script class="chart__data">`; non-datachart code blocks unchanged (`internal/mdloader/chart_renderer.go`).
- Sources end-to-end: **inline** (baked), **frontmatter** (vault `[[link]]` → presigned URL → browser fetch; assets marked in `findAssets`), **url** (server-side fetch + cache).
- Cache: `chart_data_cache` table + sqlc `GetChartData`/`UpsertChartData`.
- Orchestration: `internal/chartdata` service (embedded in `app`), `refresh_chart_data` backjob (`internal/case/backjob/refreshchartdata/`), live-update via debounced re-render.
- Widget: `assets/chart/` (lazy-loads echarts), injected via `UserJSURLs`.
- Local demo: mock adapter `scripts/data_mock_server.py` + `data_mock` compose service.
- User docs: `docs/{en,ru}/user/chartdata.md`.

### Task A — Scheduled refresh (`chart_ttl` cron) — NOT DONE

**Problem.** `chart_ttl` is documented but **not enforced**. `fetched_at` is written to `chart_data_cache` but never compared to a TTL, and there is no cron. So `url`/`internal` data is fetched once (at note load, on a cache miss) and never refreshes on a schedule — only when the note's `version_id` changes (edit). Wrong for live metrics.

**Already correct:** the initial fetch happens at **render time** (`noteloader.Load` → `chartRenderer.renderChart` → `chartdata.ChartRows` → cache miss → `EnqueueChartDataRefresh` → job → cache → debounced reload), **not** on reader request (the page serves cached HTML).

**Build:**
1. Cronjob `internal/case/cronjob/refreshstalecharts/{job.go,resolve.go}` (pattern: `internal/case/cronjob/cleanupapikeylogs/`), ~every minute; register in `internal/cronjobs/jobs.go`.
2. Each run: iterate the loaded notes' url/internal charts (`NoteView.Charts` + the note's `RawMeta["chart_ttl"]`). For each: read the cache row's `fetched_at`, parse `chart_ttl` (Go duration; absent = no auto-refresh), and if `now - fetched_at > ttl` → `EnqueueChartDataRefresh`. Job caches → `reloadLoop` re-renders.
3. Put the staleness logic in `chartdata` (e.g. `func (c *ChartData) RefreshStale(...)`), don't duplicate in the cron. Extend `chartdata.Env` so it can read `fetched_at` (today `CachedChartData` returns only the data string). Add a compile-time check.
4. Per-chart TTL: read `RawMeta["chart_ttl"]` per note, or attach the parsed TTL to each `NoteViewChart` in `extractCharts`.

**Accept:** url chart with `chart_ttl: 1m` re-fetches ~every minute (new `fetched_at`); without it, no scheduled refresh.

### Task B — `internal` SQL source — NOT DONE

**Problem.** `data.source = "internal"` renders a loader only. `chartdata.ChartRows` enqueues a refresh only for `ChartSourceURL`; the job only does HTTP. Full design is the **Internal SQL source** section below — implement it.

**Build (per that section):**
1. `note_version_frontmatters(version_id, data json)` — persist `RawMeta` as JSON at push/sync (queryable via SQLite JSON1; shared with Obsidian Bases).
2. `internal_sql_tables(table_name, min_sync_interval_secs, synced_at)` — admin-managed allowlist + cadence.
3. `internal/internalsql/` — non-durable filtered **read replica**: lazy per-table sync (`ATTACH main read-only` → `CREATE TABLE AS SELECT`), debounced by `synced_at`. Security by construction (only allowlisted tables exist).
4. Internal fetch path: run `data.sql` against the replica (SELECT-only + `LIMIT` + timeout), cache rows.
5. Extend `chartdata.ChartRows` miss-enqueue to cover `ChartSourceInternal`.
6. Admin CRUD for `internal_sql_tables`.

**Accept:** `internal` chart renders notes-by-status; queries can't reach non-allowlisted tables (e.g. `secrets`).

## Authoring format

````markdown
```datachart
{
  "title": "Daily revenue",
  "data": {
    "source": "url",
    "url": "http://clickhouse-adapter:8123/query",
    "body": "SELECT day, revenue FROM stats ORDER BY day"
  },
  "config": {
    "xAxis": { "type": "category" },
    "yAxis": { "type": "value" },
    "series": [{ "type": "line", "encode": { "x": "day", "y": "revenue" } }]
  }
}
```
````

| Field | Meaning |
|-------|---------|
| `data` | Typed data source — `{ "source": "url" \| "frontmatter" \| "internal" \| "inline", … }`. See **Data sources**. The `url` body/SQL is server-side only — never shipped to the browser. |
| `config` | An ECharts [option object](https://echarts.apache.org/en/option.html). Fetched rows are auto-injected into `config.dataset.source` (no placeholder needed; `"$data"` optional for custom placement). |

Per-note frontmatter:

```yaml
chart_ttl: 1h     # optional; refresh cadence (Go duration). Absent = refresh only on version change.
charts: custom    # optional; suppress inline rendering, place charts via ctx.Charts() in a custom template.
```

## Data sources

A chart is always a ` ```datachart ` block: **config in the block**, and a typed
`data` object that says where the rows come from. One syntax, one widget; the
`data.source` discriminator selects the path.

```datachart
{
  "data": { "source": "frontmatter", "ref": "chart_sales" },
  "config": { "series": [{ "type": "bar" }] }
}
```

| `data.source` | Shape | Data origin | Who fetches |
|---|---|---|---|
| `frontmatter` | `{source, ref}` | a frontmatter key holding a vault `[[link]]` → asset | **client** (resolve link → URL → fetch) |
| `url` | `{source, url, body?}` | external HTTP-JSON endpoint | **backend** (fetch + cache + secrets) |
| `internal` | `{source, sql}` | trip2g's own content (filtered replica) | **backend** (replica + cache) |
| `inline` | `{source, rows}` | bundled in the block | none |

**Data injection.** By default trip2g/the widget injects the fetched rows into
`config.dataset.source` automatically — you do **not** write a placeholder. If
`config` has no `dataset`, one is added as `{ "source": <rows> }`; the series'
`encode` then references the row columns. The rows may be an array-of-objects or
a 2D array (both are valid ECharts `dataset.source`).

`"$data"` is **optional**: write it explicitly anywhere in `config` only when you
need the rows somewhere other than `dataset.source` — trip2g replaces every
`"$data"` it finds. `config` stays a plain ECharts option object either way (the
trip2g-specific "where from" lives in `data`, never mixed into `config`).

### `frontmatter` source — data from a vault file (client-fetched)

The data file is referenced from **frontmatter**, not inside the code block,
because Obsidian gives frontmatter `[[links]]` real link support — highlighting,
autocomplete, graph edges — which a link buried in a ```code fence does not
(verified in Obsidian). The block just names the key:

```yaml
---
chart_sales: "[[sales.datachart.csv]]"      # Obsidian-native link property
---
```
```datachart
{ "data": { "source": "frontmatter", "ref": "chart_sales" }, "config": { "series": [{ "type": "bar" }] } }
```

At render trip2g reads `RawMeta["chart_sales"]`, extracts the wikilink target,
resolves it to the asset's presigned URL via `currentPage.AssetReplaces[target]`
(the same `ResolveWikilink` mechanism, `internal/mdloader/link_resolver.go:33`),
and emits `<div class="chart" data-src="<url>" data-format="csv|json">` + the
inline config. **The browser fetches the data** — no server-side read, no
`GetAssetContent`, no memory/streaming concerns (a `.csv`/`.json` is served like
any other asset, gated by the same presigned URL as the note's images).

- **`.json`** referenced file → array used directly as `dataset.source`.
- **`.csv`** → the widget parses it (header row + numeric coercion → 2D source);
  chart type comes from the block's `config.series[].type`.
- **Dependency tracking:** the asset must be marked as a note dependency so it
  gets a presigned URL. Body `[[links]]` are marked by `findAssets`; **frontmatter
  links need an explicit mark** (open item — verify whether frontmatter wikilinks
  are already registered as asset deps; if not, add it during `extractCharts`).

> Dropped: the standalone `![[x.datachart.json]]` embed and the server-side
> `GetAssetContent` read. Superseded by `data.source = "frontmatter"` +
> client-fetch — simpler, Obsidian-friendly, and avoids reading asset bytes
> through the app.

## Data adapter contract

`dataUrl` may be any HTTP-JSON endpoint. The reference adapter is the sibling
project `trip2g_agent_queue` (`POST /v1/query`, `internal/api/handlers.go:568`):

- **Request body**: `{"sql": "SELECT ..."}` (SELECT-only; non-SELECT → `400`).
- **Response**: a flat JSON **array of row objects** (`[]map[string]any`), e.g.
  `[{"day":"2026-06-01","revenue":1234}, ...]`; empty → `[]`; errors →
  `{"error":"..."}`.

This array maps **directly** onto an ECharts `dataset.source` — no transform
layer. The fetcher auto-injects the fetched array into `config.dataset.source`
(no placeholder needed). `data.body` is sent verbatim as the POST body (for this
adapter it is the JSON `{"sql":...}`, not raw SQL).

## Auth: secrets as request headers

Some endpoints need an auth header. Reuse the existing **secrets** subsystem
rather than inventing storage — it already provides encrypted storage,
prefix-namespacing, CRUD, and an admin widget.

**Naming** — follow the established `SecretPrefix` convention
(`internal/model/secret_prefix.go`, format `<entity>:<id>:`). For charts, key by
**domain**:

```
chart_secret:{domain}:{HEADERNAME}   →   value = header value
e.g.  chart_secret:trip2g_agent_queue:Authorization  →  "Bearer …"
      chart_secret:api.example.com:X-Api-Key         →  "…"
```

Add `ChartSecretPrefix(domain string) SecretPrefix` to `secret_prefix.go`
returning `chart_secret:{domain}:`. Its `.Like()` lists all headers for a domain.

**Fetch-time** (in the `refresh_chart_data` backjob, before the HTTP request):
1. Extract the host from `dataUrl`.
2. `ListSecretKeys("chart_secret:{host}:%")` → for each key, decrypt the value.
3. Set HTTP header `HEADERNAME: value` for each.

**Admin UI** — the secrets widget is already prefix-driven and reusable:
`$trip2g_admin_secrets` (`assets/ui/admin/secrets/secrets.view.tree`) exposes a
`key_prefix` property; `changewebhook/show` embeds it via `secrets_prefix()`
(`assets/ui/admin/changewebhook/show/show.view.ts:165`). New admin page:
- List **domains** — extracted from `chart_secret:*` keys (`ListSecretKeys`,
  split on `:`), optionally unioned with domains discovered in note chart blocks.
- For each domain, embed `$trip2g_admin_secrets` with
  `key_prefix = "chart_secret:{domain}:"` — gives add/edit/delete of header
  secrets per domain with zero new widget code.

Secrets are server-side only; like `dataBody`, they never reach the browser.

## Internal SQL source (content dashboards)

A `datachart` may target trip2g's **own data** instead of an HTTP endpoint —
turning the platform into a Grafana for its own content: notes by status,
signups over time, most-read pages, Telegram delivery stats. The chart declares
`"dataSource": "internal"` and a `dataSql` SELECT instead of `dataUrl`.

### Security by construction: a filtered read replica

Raw SELECT against the main DB is unacceptable — it holds `secrets`, `users`,
payment data for the whole instance. Instead of sandboxing queries (SQL parsing,
view whitelists — fragile), **isolate by data**: internal SQL runs against a
**separate replica SQLite file that contains only allowlisted tables**. A query
physically cannot reach `secrets` because that table isn't in the replica.

The allowlist lives in a DB table (admin-manageable at runtime, no env/restart):

```sql
CREATE TABLE internal_sql_tables (
  table_name             text    primary key,
  min_sync_interval_secs integer not null default 300,
  synced_at              integer not null default 0,   -- unix secs, last replica sync
  created_at             datetime not null default current_timestamp
);
```

Empty table → nothing queryable → feature off. No separate enable flag needed:
the rows *are* both the allowlist and the on-switch.

### Lazy sync (no timer — rides the chart refresh)

The replica is rebuilt inside the `refresh_chart_data` job, just before an
internal query, debounced per table by `synced_at`:

```
for row in internal_sql_tables:
  if now - row.synced_at > row.min_sync_interval_secs:
     BEGIN;
       replica: DROP TABLE IF EXISTS t; CREATE TABLE t AS SELECT * FROM main.t;  -- main ATTACHed read-only
     COMMIT;                                                                      -- atomic swap, no torn reads
     row.synced_at = now
```

- **Demand-driven**: tables sync only when an internal chart actually needs them;
  a burst of refreshes shares one sync (per-table `synced_at` guard).
- **Full copy is fine at target scale** (10–50k rows): `CREATE TABLE AS SELECT`
  ≈ 10–50 ms, auto-handles schema drift. Reconsider incremental only at millions
  of rows / hundreds of MB / seconds-scale cadence.
- **Replica is disposable** → open it non-durable for speed:
  `PRAGMA journal_mode=MEMORY; PRAGMA synchronous=OFF;` (rebuildable from main).
- **Isolation bonus**: heavy `GROUP BY` runs on the replica, never locking the
  main write DB — the same R/W-split principle as `docs/dev/sqlite.md`, one step
  further.

Mixed freshness: a chart joining two tables with different intervals sees one
fresher than the other (fine for dashboards; align intervals if a chart needs
cross-table consistency).

### Frontmatter index — the content substrate

To chart over note metadata (status, tags, category), frontmatter must be
**queryable**, but today `note_versions` stores only raw `content`; frontmatter
is parsed in-memory only. Persist it:

```sql
CREATE TABLE note_version_frontmatters (
  version_id integer primary key references note_versions(id) on delete cascade,
  data       text    not null   -- RawMeta serialized as JSON
);
```

Populated at push/sync (where the YAML is already parsed). Query with SQLite
JSON1: `json_extract(data,'$.status')`, `json_each(data,'$.tags')`. Add
`note_version_frontmatters` to `internal_sql_tables` and dashboards over content
work:

```sql
SELECT json_extract(f.data,'$.status') AS status, count(*) AS n
  FROM note_version_frontmatters f
  JOIN <live-version filter>
 GROUP BY status
```

**This index is shared with Obsidian Bases** (`docs/dev/obsidian_bases.md`), which
also needs a queryable frontmatter index — build it once, both features win.

### Config

| Setting | Where | Default |
|---------|-------|---------|
| allowed tables + per-table cadence | `internal_sql_tables` rows (admin CRUD) | empty (off) |
| replica file path | `internal/appconfig/config.go` | e.g. `internal_sql.sqlite` next to main DB |

## Permission model

Access to the note **is** access to all data on it. The fetched data is embedded
in the note's rendered HTML, and that HTML is only ever served to a viewer who
already passed the note's permission gate (guest / paid / admin). A guest viewing
a paid note never receives the HTML, therefore never the data, therefore never
the SQL. Different audiences → different pages. No per-block gating, no separate
data endpoint.

## Data flow

```
NOTE (markdown)            SERVER (trip2g)                       BROWSER
──────────────            ───────────────                       ───────
```datachart            parse: extractCharts()        render: chart_data_cache lookup
{ dataUrl,    ──▶   walk AST, find lang==chart    (version_id, chart_hash)
  dataBody,         hash = sha256(url\0body)            │
  config }          → NoteView.Charts[]           ┌─────┴─────┐
```                                               hit         miss / stale
                                                   │            │
                              embed config+data    │      enqueue refresh_chart_data
                              into note HTML  ◀─────┘      (goqite backjob)
                                   │                            │
                                   ▼                       HTTP fetch dataUrl/dataBody
                       <div class="chart" data-i=0>         → write cache row
                       <script class="chart__data"          → publish noteChange (update)
                        type="application/json">                  │
                        {config, data|null}</script>             ▼
                                   │                    SSE noteChanges → $trip2g_user_live
                                   ▼                    (if live-reload toggle on) → reload
                       ECharts IIFE widget draws;
                       data===null → loader
```

## Cache model

New table, keyed exactly by note version as requested:

```sql
-- db/migrations/<ts>_create_chart_data_cache.sql   (via: make db-new name=create_chart_data_cache)
-- migrate:up
CREATE TABLE chart_data_cache (
  version_id integer  not null,
  chart_hash text     not null,
  data_json  text     not null,
  fetched_at integer  not null,          -- unix seconds
  primary key (version_id, chart_hash)
);

-- migrate:down
DROP TABLE IF EXISTS chart_data_cache;
```

- `chart_hash = sha256(dataUrl + "\x00" + dataBody)` — identical query under the
  same version reuses the row.
- Keying by `version_id` gives free invalidation: a new note version is a new
  key, so it starts empty (→ loader) until the refresh job fills it. No
  copy-forward.

### Render-time semantics

Look up `(version_id, chart_hash)`:

| State | Embedded `data` | Loader? | Side effect |
|-------|-----------------|---------|-------------|
| No row | `null` | yes | (refresh job already enqueued on version change) |
| Fresh (`now - fetched_at ≤ ttl`) | data | no | none |
| Stale (`now - fetched_at > ttl`) | old data | no | enqueue `refresh_chart_data` |

Refresh is triggered by **exactly two events**: a version bump (job enqueued at
publish time) and a render that observes staleness. Nobody views the page →
nothing refetches. There is no polling cron for freshness.

**No feedback loop:** the job writes `fetched_at = now` *before* publishing the
`noteChange`. The triggered reload then sees a fresh row and does not re-enqueue.
This ordering is covered by a test.

## Components & file touch-points

| # | Piece | Location | Convention to mirror |
|---|-------|----------|----------------------|
| 1 | `NoteViewChart` type + `Charts []NoteViewChart` field + `extractCharts()` | `internal/model/note.go` | `Headings` field (`:213`) and `extractHeadingsAndGenerateIDs()` (`:815`); called from `ExtractMetaData()` (`:541`). Block-walking mirrors `extractJsonnetBlocks()` in `internal/mdloader/vault_patch.go:53`. Read `chart_ttl` / `charts` frontmatter via the existing `extract*` helpers. |
| 2 | `chart_data_cache` table + read/write queries | `db/migrations/`, `internal/db/queries.read.sql`, `internal/db/queries.write.sql` | `make db-new name=...` (dbmate) then `make sqlc`. Table style: `db/migrations/20260526120000_create_secrets.sql`. |
| 3 | `refresh_chart_data` backjob (resolve domain secrets → HTTP-JSON fetch with headers → inject array into `config.dataset.source` → cache write → `noteChange` publish) | `internal/case/backjob/refreshchartdata/{job.go,resolve.go}` | `internal/case/backjob/sendformsubmit/` — `const JobID`, `QueueID = model.BackgroundDefaultQueue`, `jobs.Register(env, QueueID, JobID, Priority, Resolve)`, `Enqueue*` method, `Params` struct + `Resolve`. Register import in `cmd/server/main.go` (alongside `:47`). |
| 3b | `ChartSecretPrefix(domain)` helper + domain-header lookup | `internal/model/secret_prefix.go` | existing `ChangeWebhookSecretPrefix` etc.; read via `ListSecretKeys(prefix.Like())` + decrypt. |
| 6b | Admin secrets-by-domain page | `assets/ui/admin/chartsecrets/` + GraphQL list-domains | reuse `$trip2g_admin_secrets` widget with `key_prefix="chart_secret:{domain}:"`, as `changewebhook/show` does (`show.view.ts:165`). |
| 4 | Cron cleanup of stale-version rows | `internal/case/cronjob/cleanupchartdatacache/job.go` | `internal/case/cronjob/cleanupapikeylogs/job.go` — `Name()`, `Schedule()` cron string, `ExecuteAfterStart()`, `Execute()→Resolve()`. Register in `internal/cronjobs/jobs.go`. |
| 5 | Goldmark node renderer for `datachart` blocks (inline `<div class="chart">` + embedded JSON; suppressed when `charts: custom`) | `internal/mdloader/loader.go:112-119` | register via `renderer.WithNodeRenderers(util.Prioritized(&chartRenderer{}, <priority>))` next to `linkRenderer`/`imageRenderer`/`headingRenderer`. |
| 5b | Embed renderer for `![[*.datachart.json]]` (read `NoteAsset` bytes → emit chart container + bundled JSON; no fetch/cache) | `internal/mdloader/loader.go:114` (image/embed renderer) | suffix-detect `.datachart.json`; read via `NoteAssetByAbsolutePathAndSha256Hash`/`NoteAssetByID`. |
| 6 | `ctx.Charts()` helper returning `ctx.Note.Charts` | `internal/defaulttemplate/template.go` | existing `Ctx` methods (e.g. `ContentRefs()` `:212`). |
| 7 | ECharts client widget (vanilla IIFE) | `assets/chart/src/index.ts` + `assets/chart/esbuild.browser.mjs` + `package.json` script `chart` → `assets/chart.js`; injected via `UserJSURLs()` in `cmd/server/main.go:999` | `assets/toc/` (esbuild IIFE reading embedded `application/json`). ECharts loaded with `$mol_import.script`-style async or bundled. |
| 9 | Obsidian plugin preview — `datachart` code-block processor (+ `.datachart.json` view) for WYSIWYG in-editor, identical to the site | `obsidian-sync/src/` (`main.ts` entry) | `registerMarkdownCodeBlockProcessor("datachart", …)`, fetch via `requestUrl`, render with the shared ECharts widget from `assets/chart/`. |
| 10 | `note_version_frontmatters` table + populate at push/sync (RawMeta → JSON) | `db/migrations/`, `internal/case/pushnotes/` or `noteloader` | serialize `NoteView.RawMeta`; shared with Obsidian Bases (`obsidian_bases.md`). |
| 11 | `internal_sql_tables` table + non-durable read replica + lazy per-table sync | `db/migrations/`, `internal/internalsql/` (new pkg), `internal/appconfig/config.go` | open replica via `dbmate/sqlite` `ConnectionString` (`internal/dbmate/sqlite/sqlite.go:76`); ATTACH main read-only; `CREATE TABLE AS SELECT` per table in one txn. |
| 11b | `internal` data source in the fetch job (run `dataSql` on replica after ensuring fresh) | `internal/case/backjob/refreshchartdata/` | SELECT-only guard + `LIMIT` cap + timeout; same cache/TTL/notify as HTTP source. |
| 11c | Admin CRUD for `internal_sql_tables` | `assets/ui/admin/internalsqltables/` | reuse admin CRUD patterns (`docs/dev/frontend_crud.md`); show `synced_at` per row. |

### Cache cleanup (cron)

Delete `chart_data_cache` rows whose `version_id` is no longer the live version
of any note. Daily schedule (e.g. `"0 0 2 * * *"`). Exact "live version" join to
be finalized against the note-version source of truth during implementation
(detail flagged, not blocking).

### Live update

No new infrastructure — reuse the existing `noteChanges` GraphQL SSE subscription
and `notebus` fan-out (`docs/dev/obsidian_sse_pulls.md`; resolver
`internal/graph/*` `NoteChanges` `:289`). The public page already subscribes via
`$trip2g_user_live` (`assets/ui/user/live/live.view.ts:4`), gated by the existing
opt-in toggle `$trip2g_user_live_reload_toggler` (persisted in `$mol_state_local`
as `trip2g_live_reload`). Step 3's job publishing a `noteChange{update}` for the
note path is the entire wiring; the page reloads and re-reads the fresh cache.

## Implementation order (TDD, red → green)

1. **Model / extraction** — `extractCharts()` finds `datachart` blocks, computes the
   hash, parses `config`/`dataUrl`/`dataBody`; reads `chart_ttl`/`charts`
   frontmatter. *Tests first* (table-driven, `testify/require`).
2. **Cache** — migration + sqlc read/write queries; thin store with get/upsert.
3. **Backjob** — `refresh_chart_data`: resolve `chart_secret:{host}:*` headers →
   fetch → inject response array into `config.dataset.source` → upsert
   (`fetched_at=now`) → publish `noteChange`. Test the write-before-notify
   ordering and the header attachment.
4. **Cron cleanup** — delete non-live-version rows.
5. **Renderer** — inline `<div class="chart">` + embedded `{config, data}`;
   suppressed under `charts: custom`. Render to `template.HTML`.
5b. **Embed renderer** — `![[*.datachart.json]]` reads the asset's bundled
   `{settings, data}` and renders a chart inline (no fetch/cache/job).
6. **`ctx.Charts()`** helper.
7. **ECharts widget** — `assets/chart/`, build script, `UserJSURLs()` injection;
   `data===null` → loader; redraw on live reload.
8. **Admin secrets-by-domain page** — list domains + embed `$trip2g_admin_secrets`
   per domain (`chart_secret:{domain}:` prefix).
9. **Obsidian plugin preview** — `datachart` code-block processor + `.datachart.json`
   handler in `obsidian-sync`, reusing the shared widget for in-editor WYSIWYG.

### Internal-source track (optional, can land independently)

10. **Frontmatter index** — `note_version_frontmatters` table + populate at
    push/sync (RawMeta → JSON). Unlocks content queries; shared with Bases.
11. **Filtered replica** — `internal_sql_tables` table + non-durable replica +
    lazy per-table sync (`internal/internalsql/`). *Tests first*: allowlist
    enforcement, atomic swap, debounce by `synced_at`.
12. **`internal` data source** — wire `dataSource:"internal"` + `dataSql` into the
    refresh job (SELECT-only guard, LIMIT, timeout); same cache/TTL/notify.
13. **Admin CRUD** for `internal_sql_tables` (with `synced_at` display).

## Acceptance criteria

- A note with one ` ```datachart ` block renders an `<div class="chart">` + a sibling
  `application/json` script containing `config` and `data` (or `data:null` on
  cache miss); `dataUrl`/`dataBody` are **absent** from the HTML.
- First render of a new version embeds `data:null`; after the `refresh_chart_data`
  job completes, a fresh render embeds the fetched data.
- Editing the note (new `version_id`) makes the chart re-fetch on next view.
- With `chart_ttl: 1h`, a render past TTL serves stale data **and** enqueues a
  refresh; a render within TTL enqueues nothing.
- The refresh job writes `fetched_at` before publishing `noteChange` (no
  re-enqueue loop) — asserted by test.
- `charts: custom` suppresses inline output; `ctx.Charts()` returns the parsed
  charts for a custom template.
- Cron cleanup removes rows for versions that are no longer live.
- ECharts widget draws the chart from embedded JSON and shows a loader when
  `data` is null.
- A `chart_secret:{host}:Authorization` secret is sent as the `Authorization`
  header on the fetch for that host; secrets never appear in the page HTML.
- The admin page lists domains from `chart_secret:*` keys and edits per-domain
  header secrets via the reused `$trip2g_admin_secrets` widget.

## Risks & mitigations

| Risk | Mitigation |
|------|------------|
| Render blocking on external HTTP | Render never fetches; it only reads SQLite. All fetching is async via the goqite backjob. |
| Untrusted upstream (SSRF, huge payloads) | Allowlist/limit `dataUrl` hosts; cap response size and apply a fetch timeout in the backjob. Flag for review. |
| Cache growth across versions | Daily cron deletes rows for non-live versions. |
| Refresh feedback loop via live reload | Write `fetched_at` before `noteChange`; tested. |
| ECharts bundle size on public pages | Ship a focused esbuild build; load async so it doesn't block first paint. |

## Out of scope (MVP)

- Admin drag-and-drop / reconfigure (would reuse `$mol_drag` from
  `assets/ui/admin/layout/editor/`, persisting layout back into the note).
- Per-viewer layouts.
- Charting against trip2g's own SQLite or per-block permissions.
