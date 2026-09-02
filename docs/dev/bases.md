# Obsidian Bases (.base) on published sites

## TL;DR (вывод)

A `.base` is a **saved database-view over the vault**: YAML that declares filters
on frontmatter properties, the columns to show, sort, group, and a view type
(`table` / `cards`). Think Dataview / Notion DB, expressed as a file.

The big reframe: **`.base` files are already ingested end-to-end.** They sync
(plugin/CLI/browser whitelists + `pushNotes` `allowedExtensins`,
`internal/case/pushnotes/resolve.go:31`), get stored verbatim in
`note_versions.content`, get a URL that keeps the extension (`/tasks.base`,
`internal/mdloader/loader.go:295-297`), and reach the page endpoint. The **only**
missing piece is rendering — today the render path short-circuits every raw file
to a "Bases are not supported yet" stub (`endpoint.go:96-103` →
`views.html:536`). **This is a render-layer feature, not a greenfield build.**

The plan, in one breath:

1. **Parse** the `.base` YAML into a typed spec with a new package
   `internal/obsidianbase` (mirror `internal/obsidiancanvas`), parse it in
   `registerRawFile`, store it on `model.NoteView.Base` next to `.Canvas`. The spec
   is **two-level**: top-level `filters`/`formulas`/`properties` plus a `views[]`
   array, where each view *inherits* the base-level query and adds its own.
2. **Index** every note's frontmatter into a set of persistent SQL tables — five
   **per-type value tables** (`note_string_values` / `note_int_values` /
   `note_real_values` / `note_bool_values` / `note_date_values`), each keyed by
   `(version_id, prop, seq)`, mirroring the established `form_*_values` pattern.
   The index is populated when the in-memory note set is (re)loaded. Bases
   filter/sort/group/page in **SQL** against this index — `WHERE` / `ORDER BY` /
   `GROUP BY` / `LIMIT`/`OFFSET` — instead of reflecting over the whole in-memory set
   per request. This is the owner-chosen architecture; the in-memory `NoteQuery`
   approach is kept only as a possible v0 spike (see
   [Alternatives](#alternatives-considered)).
3. **Render one spec into three outputs**: an SSR page (table + cards), a JSON
   data endpoint, and `![[tasks.base]]` embeds — all from the same SQL-resolved row
   set, with pagination as plain SQL `LIMIT`/`OFFSET`. Reuse magazine card markup for
   the cards view.
4. **Re-apply access control per viewer, per row** — the raw-file short-circuit
   currently runs *before* paywall/signin, so a renderer that lists other notes
   must filter each surfaced row by `Free` / `CanReadNote`. Because the predicate
   runs per viewer, the *same* base automatically shows fewer rows to anonymous
   visitors and more to subscribers — the common "public teaser vs full member list"
   needs **no author effort and no second base**.

> **Pending owner confirmation — new SQL migration.** The property index requires
> five new tables (`note_string_values` / `note_int_values` / `note_real_values` /
> `note_bool_values` / `note_date_values`), created by a single new SQL migration. Per
> `CLAUDE.md` ("SQL migrations: always ask for confirmation before creating"), this
> doc *specifies* the schema but the migration **must not be created yet**. Treat the
> schema below as a proposal awaiting sign-off.

Bases also unlock a product cleanup: let the magazine take a `.base` as its data
source (decouple *what* data from *how* it looks), and replace the five
`magazine_*` knobs with named presets. The magazine could later migrate to the same
property index instead of its bespoke in-memory query.

---

## What a `.base` is

### Obsidian semantics

A `.base` file is YAML with a **two-level structure**: a set of top-level keys
(`filters` / `formulas` / `properties`) that apply to the whole base, and a
`views[]` array where **each view inherits the base-level query** and adds its own
`filters` (ANDed with the base ones), column `order`, `sort`, and `limit`. Real
example from `docs/demo/example_base.base` — note the **two views in one file**:

```yaml
# --- base level: shared by every view below ---
formulas:
  content: formula.content
  Untitled: ""
properties:
  formula.content:
    displayName: content
# --- per-view level: each inherits the base level, adds its own ---
views:
  - type: table
    name: Table
    filters:
      and:
        - telegram_publish_at != ""
  - type: table
    name: telegram notes
    filters:
      and:
        - "!telegram_publish_at.isEmpty()"
    order:                       # selected/projected columns, in order
      - formula.Untitled
      - file.backlinks
      - title
    sort:
      - property: file.name
        direction: DESC
```

Key parts of the schema:

| Key | Level | Meaning |
|-----|-------|---------|
| top-level `filters` | base | a boolean tree (`and`/`or`/`not`) every view inherits |
| top-level `formulas` | base | computed columns shared by views — out of scope for v1 |
| top-level `properties` | base | per-column display config (`displayName`) shared by views |
| `views[]` | view | one or more named views; each is `type: table\|cards` |
| view `filters` | view | view-specific boolean tree, **ANDed** with the inherited base filter |
| view `order` | view | the projected columns, in order (frontmatter props + `file.*` built-ins like `file.name`, `file.backlinks`) |
| view `sort` | view | list of `{property, direction}` |
| view `limit` | view | max rows for this view (becomes SQL `LIMIT`) |

So "make two tables out of one query" is the **native Obsidian model**: two entries
in `views[]`, both inheriting the base-level filter/properties. There is **no
cross-*file* inheritance** in Obsidian (no `extends: [[other.base]]`); that is a
deliberate non-goal here too (see [Inheritance](#inheritance-two-views-not-two-files)),
which keeps `.base` files portable.

### How it'll be used on a published trip2g site

- **A standalone page** at `/tasks.base` showing a live table of notes that match
  a filter (e.g. "all posts with `status: published`, sorted by date"). Static,
  server-rendered, crawlable.
- **JSON for the same data** at the same path with `Accept: application/json` (or
  `/tasks.base.json`) — the owner explicitly wants ".json access to the data".
  This also feeds pagination for large embedded tables.
- **Embedded inside an article** via `![[tasks.base]]` — an SSR table the author
  drops into a post, JS layered on top for client-side sort/filter.
- **As a magazine data source** — `magazine_source: [[tasks.base]]` so the
  magazine renders *that* query's rows as cards.

---

## Current state in trip2g

`.base` is already a first-class ingested raw file. What exists vs. the stub:

| Stage | Status | Evidence |
|-------|--------|----------|
| Sync whitelist (plugin/CLI/browser) | ✅ done | `pushnotes/resolve.go:27-33` `allowedExtensins` includes `.base` |
| Stored verbatim, no markdown pipeline | ✅ done | `mdloader/loader.go:263-266` `isRawFile`; `:270-299` `registerRawFile` |
| URL keeps extension | ✅ done | `loader.go:295-297` `Permalink = "/" + src.Path` |
| Reaches page endpoint | ✅ done | catch-all `rendernotepage` (note_rendering.md) |
| **Parsed into a spec** | ❌ **missing** | `registerRawFile` only parses `.canvas` (`loader.go:286-292`); `.base` gets `RawMeta={}`, no struct |
| **Rendered** | ❌ **stub** | `endpoint.go:96-103` → `UnsupportedFileExt=".base"` → `views.html:536` "Bases are not supported yet." |
| **Embeddable** | ❌ **skipped** | raw files have empty `HTML`; `link_renderer.go:235-238` does `WalkSkipChildren` on `len(note.HTML)==0` |
| Search index | (n/a) | raw files skipped from Bleve by design |

The render hook short-circuit:

```go
// internal/case/rendernotepage/endpoint.go:96-103
if resp != nil && resp.Note != nil {
    if ext := unsupportedFileExt(resp.Note.Path); ext != "" {
        dtCtx := buildDefaultTemplateCtx(req, layoutParams, resp, env)
        dtCtx.UnsupportedFileExt = ext           // ".base"
        defaulttemplate.WriteRender(ctx, dtCtx)
        return nil, nil
    }
}
```

**Critical:** this block runs at `endpoint.go:96`, *before* the signin-wall
(`:115`) and paywall (`:130`) checks. Raw files currently skip all access
control. A real Bases renderer that surfaces *other* notes' titles, links, and
property values **must re-apply access control to each surfaced row** (see
[Access control](#access-control-per-viewer-per-row)).

### Query engine that already exists (and why we still add an index)

`internal/templateviews/NoteQuery` (`query.go:21-249`) is a lazy in-memory query
builder over the whole note set:

- `NVS.ByGlob(pattern)` / `NVS.Query()` entry points (`nvs.go:192,200`).
- `SortBy(field)` (reflection over `Note` methods) and `SortByMeta(field)` (sort
  by **arbitrary frontmatter property**, `query.go:36-42,197-208`).
- `Limit` / `Offset` paging (`query.go:60,66`).
- `compareValues` (`query.go:213-248`) handles `string/int/int64/float64/time.Time`.

`internal/defaulttemplate/magazine.go` already renders a **glob-filtered +
property-sorted + property-included/excluded** note collection as cards
(`MagazineItems()` at `:29`), driven by `magazine_*` frontmatter (`template.go:305-344`).

Why this isn't enough for bases: frontmatter is **opaque** in SQLite —
`note_versions.content` (`db/schema.sql:83-91`) is raw text; the frontmatter is only
parsed into an in-memory `RawMeta map[string]interface{}` (`model.NoteView.RawMeta`,
populated at `loader.go:756`). There is **no frontmatter column and no way to SQL
filter, sort, group, or page by property**. The in-memory `NoteQuery` does all of
this in Go by reflecting over the whole note set on every request, with no
`WHERE`-style predicate. The owner-chosen architecture closes that gap with a
**persistent property index** (next section), so bases run as real SQL queries with
cheap `LIMIT`/`OFFSET` pagination. `NoteQuery` stays as-is for magazine and other
in-memory consumers; bases get a new SQL-backed query path (we are **not** rewriting
all of `NoteQuery`).

---

## Chosen architecture: one spec, three outputs (SQL-backed)

```
   note (re)load                      .base YAML  (note_versions.content, raw)
   (PrepareLatest/Live → loader)           │
        │ index frontmatter         registerRawFile │ obsidianbase.Parse (NEW)
        ▼                                            ▼
  note_{string,int,real,bool,date}_values  model.NoteView.Base
  (5 NEW per-type tables)           ──►   *obsidianbase.Base {
  (version_id, prop, value, seq)           Filters, Properties, Formulas,
        │                                   Views[]{Filters,Columns,Sort,Limit,Type} }
        │                                            │
        └──────────────► SQL query path (NEW) ◄──────┘
              base spec → SELECT … FROM note_<type>_values
              (pick the table by the filter literal's value type)
              WHERE <base filters AND view filters>
              ORDER BY <sort>  [GROUP BY <group>]  LIMIT/OFFSET
              + per-viewer access-control predicate (Free / CanReadNote)
                            │
        ┌───────────────────┼────────────────────────────┐
        │                   │                            │
        ▼                   ▼                            ▼
  (1) SSR PAGE        (2) JSON ACCESS              (3) EMBED ![[x.base]]
  table + cards       resolved rows as JSON        SSR table (first page)
  reuse magazine      Accept: application/json     + <script> JSON island
  card markup/CSS     or /x.base.json              + glue for sort/filter,
  static, crawlable   SQL LIMIT/OFFSET source       lazy-load more from (2)
```

The unification is the point: **one SQL query resolves filter+sort+project+page**;
the three outputs differ only in serialization (HTML table, HTML cards, JSON) and in
delivery (full page, content-negotiated, embedded). Pagination is plain SQL
`LIMIT`/`OFFSET` against the index, so large embedded tables stay cheap.

### Data flow for the standalone page

1. Request `/tasks.base` hits `rendernotepage`.
2. `Resolve` finds the note by path (extension preserved).
3. `endpoint.go` detects `.base`, but instead of the stub it now: reads
   `resp.Note.Base`, builds a SQL query from the (base-level ∧ view-level) spec
   against the type-appropriate per-type value table(s), applies the per-viewer
   access-control predicate, runs it (with `LIMIT`/`OFFSET`), and renders a Bases
   template func (`WriteBase`) into the default-template `Ctx`.
4. The page emits a static `<table>` + a cards section + a JSON data-island
   `<script type="application/json">` and a conditionally-injected glue script.

---

## Frontmatter property index (per-type value tables)

This is the **owner-chosen query architecture**: a persistent SQL index of every
note's frontmatter, so bases run as real SQL queries instead of in-memory
reflection. It mirrors the project's existing precedent of splitting arbitrary
key/value data into typed storage — see `form_string_values` / `form_int_values` /
`form_bool_values` (`db/schema.sql:737-754`), each keyed by `(submit_id, field_name)`
with a typed `value` column. The index follows that house pattern exactly: **one
table per value type**, no discriminator.

> **Pending owner confirmation.** The schema below is a **proposal**. It introduces a
> new SQL migration, and per `CLAUDE.md` ("SQL migrations: always ask for
> confirmation before creating") **the migration must not be created until the owner
> signs off**. Nothing here should be applied to `db/schema.sql` yet.

### Chosen schema — five per-type tables (mirror `form_*_values`)

Two shapes matched the `form_*` precedent. The owner chose **(A), per-type tables**,
to mirror the established house pattern:

- **(A) per-type tables** (`note_string_values` / `note_int_values` / …) — direct
  extension of `form_*_values`. **Chosen.** `string`/`int`/`bool` are the exact mirror
  of `form_string_values`/`form_int_values`/`form_bool_values` (`db/schema.sql:737-754`);
  `real` (floats) and `date` (a unix-epoch integer, for clean date sort/range — the
  magazine "sort by date" use case) are the frontmatter additions. The new column the
  `form_*` tables lack is `seq`, a list index for multi-valued props (`tags` etc.):
  one row per element, so `contains` becomes an equality probe.
- **(B) one table with a `value_kind` discriminator + typed columns** — one row per
  (note version, property), the kind tells which typed column is live. **Rejected.**
  Its one merit was co-locating a property's heterogeneous values in a single row, so
  a value that compares as string *or* number *or* date lives in one place. It was
  reconsidered for that flexibility, then dropped: the owner prefers to mirror the
  established `form_*_values` house pattern rather than introduce a one-off
  discriminator shape. See the trade-off this choice accepts below.

```sql
-- PROPOSED — DO NOT CREATE YET (pending owner confirmation)
-- A single migration file (db/migrations/<ts>_create_note_property_values.sql)
-- creates all five tables + five indexes.
CREATE TABLE note_string_values (
  version_id integer not null references note_versions(id) on delete cascade,
  prop       text    not null,
  value      text    not null,
  seq        integer not null default 0,
  primary key (version_id, prop, seq)
);
CREATE TABLE note_int_values (
  version_id integer not null references note_versions(id) on delete cascade,
  prop       text    not null,
  value      integer not null,
  seq        integer not null default 0,
  primary key (version_id, prop, seq)
);
CREATE TABLE note_real_values (
  version_id integer not null references note_versions(id) on delete cascade,
  prop       text    not null,
  value      real    not null,
  seq        integer not null default 0,
  primary key (version_id, prop, seq)
);
CREATE TABLE note_bool_values (
  version_id integer not null references note_versions(id) on delete cascade,
  prop       text    not null,
  value      integer not null,            -- 0 | 1
  seq        integer not null default 0,
  primary key (version_id, prop, seq)
);
CREATE TABLE note_date_values (
  version_id integer not null references note_versions(id) on delete cascade,
  prop       text    not null,
  value      integer not null,            -- unix epoch seconds
  seq        integer not null default 0,
  primary key (version_id, prop, seq)
);
CREATE INDEX idx_note_string_values_prop on note_string_values(prop, value);
CREATE INDEX idx_note_int_values_prop    on note_int_values(prop, value);
CREATE INDEX idx_note_real_values_prop   on note_real_values(prop, value);
CREATE INDEX idx_note_bool_values_prop   on note_bool_values(prop, value);
CREATE INDEX idx_note_date_values_prop   on note_date_values(prop, value);
```

Keying by `version_id` (referencing `note_versions(id)`) matches the established
per-version pattern of `note_version_assets`, `note_version_embeddings`, etc. — the
index is content-addressed by note version, so a content change naturally produces a
new key set.

Why these indexes: each `(prop, value)` composite index makes `WHERE prop = ? AND
value <op> ?` an index range scan over the type-appropriate table, and serves
`ORDER BY value` / `GROUP BY value` for the queried property. `LIMIT`/`OFFSET`
pagination then walks the index without a full scan. Multi-valued props (lists like
`tags`) get one row per element via `seq`, so `contains` is an equality probe.

### Accepted trade-off — type lives in the table, not in the row

The per-type split has a cost the single-table design avoided: **the query path must
pick the table by the filter's value type, and one property can land in different
tables across notes.** If `priority: 2` is an int in one note but `priority: "2"` is a
string in another, the two values sit in `note_int_values` and `note_string_values`
respectively — so `priority > 2` queries `note_int_values` and **misses** the
string-typed row. (The discriminator design sidestepped this by keeping a property's
values co-located.)

Resolution:

- **Classify each value deterministically at ingest** — decide its type once when the
  index is populated, and write it to exactly one of the five tables.
- **Pick the table by the comparison literal's type** in the query path: a numeric
  literal (`> 2`) targets `note_int_values`/`note_real_values`, a quoted literal
  targets `note_string_values`, a date literal targets `note_date_values`, etc.
- **Warn on type-mismatched props** — when the same property is observed with
  inconsistent types across notes, surface it as an indexing/parse warning so authors
  can normalize their frontmatter, rather than silently dropping rows.

### Where the index is populated

Frontmatter is parsed at note load into the in-memory `RawMeta` map
(`loader.go:756`, `pp.RawMeta = rawMeta`). The in-memory note set itself is rebuilt
on every reload via `noteloader.Load()`, driven by the app's `PrepareLatestNotes` /
`PrepareLiveNotes` (`cmd/server/main.go:492,508`) — which are called after writes
(`pushnotes` → `insertnote` → `PrepareLatestNotes`, `pushnotes/resolve.go:83`) and
after every admin mutation that touches notes.

**Populate the index in the same reload, treating it as derived data** — rebuilt
from the in-memory set, never hand-maintained:

- After `Load()` produces the `NoteViews`, write each note's `RawMeta` into the
  appropriate per-type table — classify each value (string/int/real/bool/date) and
  route it to `note_string_values` / `note_int_values` / `note_real_values` /
  `note_bool_values` / `note_date_values`, keyed by its `version_id` (one batch per
  note, fanning out across the five tables).
- Keep it in sync the way other derived data is: a reload replaces the rows for the
  affected versions (delete-by-version across all five tables, then re-insert), so the
  index can never drift from the loaded content. A full reload rebuilds the whole
  index; a partial reload rebuilds only the changed versions.
- The `.base` *files themselves* go through `registerRawFile` (`loader.go:270`) and
  carry `RawMeta = {}` — they are query *definitions*, not query *subjects*, so they
  contribute nothing to the index. The index is built from the **markdown notes**
  that `finishPage` produces.

This is the realistic, low-risk population point: it reuses the existing reload
lifecycle (the same one the magazine already depends on for fresh data) rather than
threading index writes through every individual mutation case.

### sqlc workflow

Queries are split read/write per `CLAUDE.md`: read queries go in `queries.read.sql`,
writes in `queries.write.sql` (both at repo root; `sqlc.yaml` points at them), then
`make sqlc` regenerates `internal/db/queries.read.sql.go` / `queries.write.sql.go`.
For bases:

- **write** (`queries.write.sql`): per-type delete + insert pairs, e.g.
  `DeleteNoteStringValuesByVersion` / `InsertNoteStringValue` (and the same for int,
  real, bool, date) — batch-friendly.
- **read** (`queries.read.sql`): the base query is **dynamic** (filters/sort/columns
  come from the spec, and the target table depends on the value type), so it can't be
  a single static sqlc query. Build it as parameterized SQL in the new query path, or
  express the common predicates as a few composable sqlc queries the path stitches
  together. Static helpers like per-type counts can still live in `queries.read.sql`.

The query layer that becomes SQL-backed is a **new path for bases** — not a rewrite
of `templateviews.NoteQuery`. The magazine (`MagazineItems()`) could later migrate to
the same index, but that is out of scope for v1.

---

## Alternatives considered

| Alternative | Why rejected |
|-------------|--------------|
| **In-memory query over `model.NoteViews`** (extend `templateviews.NoteQuery` with `WhereMeta`/`GroupBy`, reflect per request — *the previously recommended approach*) | Rejected by the owner in favor of the persistent property index. It needs no migration and matches the magazine precedent, so it is **kept as a possible v0 spike** to prove the renderer/template/JSON layers before the index lands — but it has no real `WHERE`/`LIMIT` pushdown (every request reflects over the whole set), and `GROUP BY` / paging are awkward. The SQL index is the chosen v1 target. |
| **Cross-*file* base inheritance** (`extends: [[other.base]]`, a shared base file other `.base` files inherit) | Not an Obsidian feature, and a deliberate non-goal: it would make `.base` files non-portable (a file's meaning would depend on another file). The native in-file `views[]` model already covers "share one query across several tables". |
| **Client-only render** (ship the raw `.base` + a JS engine) | Bad SEO/first-paint; duplicates the Go query engine in TS; no JSON-for-crawlers. The mermaid doc already argues against client loaders (request waterfall, preload scanner blind to injected scripts). |
| **New dedicated `/base/...` route** | The note already has a URL with the extension. Adding a route fragments the model; content-negotiation on the existing path is simpler and matches `.json` access. |
| **Parse `.base` lazily at render time** | `.canvas` is parsed at load (`loader.go:286-292`) and consumed elsewhere (the Telegram canvas-bot, `internal/case/handletgcanvasupdate`). Mirror it: parse once at load, store on the note. |

---

## Backend changes

### 1. New parser package `internal/obsidianbase`

Mirror `internal/obsidiancanvas/canvas.go` structurally: pure parsing, no
business logic, no query execution. Crucially, the parser must model the **native
two-level Obsidian structure** — base-level `filters`/`formulas`/`properties` plus a
`views[]` array, where each view inherits the base level (see
[What a `.base` is](#obsidian-semantics)).

```go
// internal/obsidianbase/base.go
package obsidianbase

type ViewType string
const (
    ViewTable ViewType = "table"
    ViewCards ViewType = "cards"
)

// Base is the two-level spec: top-level keys shared by every view, plus Views.
type Base struct {
    Filters    *FilterNode               // top-level filter, inherited by all views
    Properties map[string]PropertyConfig // top-level displayName etc., inherited
    Formulas   map[string]string         // top-level raw exprs; not evaluated in v1
    Views      []View
}

type View struct {
    Name    string
    Type    ViewType
    Filters *FilterNode   // view-specific; effective filter = Base.Filters AND View.Filters
    Columns []string      // from `order`: projected props + file.* built-ins
    Sort    []SortKey     // {Property, Desc}
    GroupBy string
    Limit   int           // becomes SQL LIMIT; 0 = unlimited
}

type FilterNode struct {
    And  []*FilterNode
    Or   []*FilterNode
    Not  *FilterNode
    Expr *FilterExpr      // leaf: property op value
}

type FilterExpr struct {
    Property string
    Op       FilterOp     // Eq, Ne, Exists, NotEmpty, Contains, Gt, Lt, ...
    Value    string
}

func Parse(raw []byte) (*Base, error) { /* yaml.Unmarshal + normalize */ }

// EffectiveFilter ANDs the base-level filter with a view's own filter.
func (b *Base) EffectiveFilter(v View) *FilterNode { /* AND(b.Filters, v.Filters) */ }
```

Inheritance is the parser's job: `EffectiveFilter` (or the query path) combines
`Base.Filters` with each `View.Filters` via AND, and base-level `Properties` apply to
every view unless a view overrides a column's config. There is **no `extends:`
key** — cross-file inheritance is a non-goal (see
[Inheritance](#inheritance-two-views-not-two-files)).

Parsing notes:
- Use the YAML lib already in the tree (frontmatter uses `goldmark-meta`; check
  `go.mod` for `yaml.v3` before adding a dep).
- The expression mini-language (`!prop.isEmpty()`, `prop != ""`,
  `status == "done"`) needs a small tokenizer. v1: support the common shapes
  (`==`, `!=`, `prop != ""` → NotEmpty, `prop.isEmpty()` → empty/exists). Document
  unsupported expressions as a parse warning, not a hard error.
- `file.*` built-ins (`file.name`, `file.backlinks`, `file.mtime`) map to `Note`
  methods, not `RawMeta`.

### 2. Store the spec on the note

```go
// internal/model/note.go (next to Canvas at :282-284)
// Base holds the parsed Obsidian .base spec for .base files.
Base *obsidianbase.Base `json:"-"`
```

```go
// internal/mdloader/loader.go:286-293  (add a sibling branch)
if strings.HasSuffix(strings.ToLower(src.Path), ".base") {
    base, err := obsidianbase.Parse(src.Content)
    if err != nil {
        ldr.log.Warn("base parse failed", "path", src.Path, "error", err)
    } else {
        nv.Base = base
    }
}
```

### 3. SQL query path for bases (against the per-type value tables)

The base spec compiles to SQL over the property index (see [Property
index](#frontmatter-property-index-per-type-value-tables)), **not** to in-memory
`NoteQuery` predicates. Add a small query builder (e.g. `internal/basesquery` or a
helper in the renderer's package) that takes `*obsidianbase.Base` + a view + the
viewer and produces a parameterized SQL statement. Each predicate selects the table
that matches the comparison literal's type (see [the trade-off
above](#accepted-trade-off--type-lives-in-the-table-not-in-the-row)):

- **Filter → `WHERE`** — walk the `EffectiveFilter` tree (base ∧ view) into a
  `WHERE` clause, using `EXISTS`/`JOIN` subqueries per property predicate against the
  type-appropriate table. Map operators: `Eq`/`Ne` → `=`/`<>`, `Exists`/`NotEmpty` →
  `EXISTS (SELECT 1 FROM note_string_values … WHERE prop = ? AND value <> '')`,
  `Contains` → equality probe against the multi-valued `seq` rows, ordered comparisons
  → `note_int_values`/`note_real_values`/`note_date_values` depending on the literal.
- **Sort → `ORDER BY`** on `value` in the type-appropriate table for each sort key;
  the table choice makes dates and numbers order correctly (the index avoids the
  in-memory `compareValues` coercion problem — types are decided once at ingest, not
  per comparison).
- **GroupBy → `GROUP BY`** on `value` in the grouped property's type table.
- **Limit/Offset → `LIMIT ? OFFSET ?`** straight from the view's `Limit` and the
  page request — cheap paging for both the page and embeds.
- **Projection** — select the row identity (`version_id` → resolve to `*Note`) plus
  the projected columns named in `order`; `file.*` built-ins resolve from the
  `*Note` after the query, not from the index.

Type coercion that the old in-memory plan had to do at compare time
(`compareValues`, `query.go:213-248`, only handled same-type branches) now happens
**once at ingest**: when a property is indexed, classify its value (date →
`note_date_values` as unix epoch, float → `note_real_values`, int →
`note_int_values`, etc.) and write it to one table, so the SQL comparison is already
type-correct. If the in-memory v0 spike is built first, keep the `coerce(a, b any)`
helper note from that path; it disappears once the index lands.

### 4. Render-hook wiring (the insertion point)

Replace the `.base` arm of the stub at `endpoint.go:96-103`. Keep the `.canvas` /
`.excalidraw` stub behavior; branch `.base` to a real renderer:

```go
if ext := unsupportedFileExt(resp.Note.Path); ext != "" {
    if ext == ".base" && resp.Note.Base != nil {
        // resolveBaseRows runs the SQL query against the per-type value tables
        // with the per-viewer access-control predicate, paged by LIMIT/OFFSET.
        rows := resolveBaseRows(ctx, env, resp)
        dtCtx := buildDefaultTemplateCtx(req, layoutParams, resp, env)
        dtCtx.Base = &defaulttemplate.BaseView{Spec: resp.Note.Base, Rows: rows}
        defaulttemplate.WriteRender(ctx, dtCtx)
        return nil, nil
    }
    // unchanged stub for .canvas / .excalidraw (and unparsed .base)
    ...
}
```

Add `Ctx.Base` + a `WriteBase` quicktemplate func (see Frontend) and gate the JSON
content-negotiation here (see JSON access).

### 5. Narrow accessor for conditional glue (copy the chart/mermaid pattern)

Add `Note.HasBaseEmbed()` / `Note.IsBase()` accessors on `templateviews/note.go`
(next to `HasCharts()` at `:169`). In `buildDefaultTemplateCtx`
(`endpoint.go:475`; the `chart.js` / `mermaid.js` / `codeblock.js` appends are at
`:507-513`), append `/assets/base.js` only when the page is a `.base` page or
contains a `![[*.base]]` embed:

```go
// endpoint.go ~:513 (after the codeblock.js append)
if note.IsBase() || note.HasBaseEmbed() {
    jsURLs = append(jsURLs, env.AssetURL("/assets/base.js"))
}
```

---

## Frontend changes

### Glue bundle `assets/base/` (copy `assets/mermaid/`)

Each widget bundle in the repo is the same shape: `src/` + `esbuild.browser.mjs`
+ a dedicated npm script + an entry in `embed.go`. `assets/base/` follows suit:

| File | Role |
|------|------|
| `assets/base/src/index.ts` | thin glue: finds `<table data-base>`, wires client-side column sort + text filter; for embeds, lazy-loads more rows from the JSON endpoint when "show more" is clicked |
| `assets/base/esbuild.browser.mjs` | builds `assets/base.js` (IIFE) |
| `assets/base.js` | committed artifact |

v1 is intentionally light — most work is server-side. The `.base` table is **fully
rendered server-side**; JS only enhances (sort/filter/pagination). No heavy lib,
so no `lib.ts` global like mermaid's `echarts.min.js`. Theme: follow
`window.trip2g_theme_listeners` only if the widget needs theme-reactive styling
(table is CSS-themed, so likely unnecessary).

### Build wiring

```jsonc
// package.json scripts (next to "mermaid", "codeblock" at :18-20)
"base": "node assets/base/esbuild.browser.mjs"
```

```go
// assets/embed.go:8 — add base.js to the //go:embed list
//go:embed ... mermaid.js mermaid.min.js codeblock.js base.js ...
```

`npm run base` builds the artifact. Commit `assets/base.js`.

### Templates + CSS

- `internal/defaulttemplate/views.html`: add `WriteBase` / `Base(ctx)`
  quicktemplate funcs that render the `<table data-base>` (thead from
  `Columns`, tbody from `Rows`) and a cards section. **Reuse the magazine card
  markup** (`magazine.go` items → same `.magazine`/card classes) for the cards
  view so they share styling. Emit a JSON data-island
  `<script class="base__data" type="application/json">…</script>` exactly like the
  chart widget data-island (`<script type="application/json" class="widget__data">`,
  generated at `views.html.go:839`).
- After editing `views.html`, **run `go generate ./internal/defaulttemplate/...`**
  and commit the regenerated `views.html.go` (the stub `UnsupportedFile` lives at
  `views.html:527`, generated `StreamUnsupportedFile` at `views.html.go:2314`).
- CSS for the table goes in `assets/defaulttemplate/src/index.scss`; compile with
  `npm run defaulttemplate-css`. Cards reuse existing magazine SCSS.

---

## Access control: per viewer, per row

The raw-file short-circuit (`endpoint.go:96-103`) runs **before** the signin-wall
(`:115`) and paywall (`:130`) checks — so raw files currently skip access control
entirely. For a `.canvas`/`.excalidraw` stub that shows no real content this is
harmless. A Bases renderer surfaces **other notes'** titles, permalinks, and property
values — so it must re-apply access control to each surfaced row.

**The decision: locked rows are HIDDEN (dropped) by default for viewers who can't
access them — and the predicate is per viewer.** That last part is the product win:
the *same* base automatically surfaces fewer rows to anonymous visitors and more to
subscribers. The common "public teaser list vs full members-only list" needs **no
author effort and no second base** — one base, evaluated against whoever is looking.

Rule for `resolveBaseRows`:

1. The `.base` page itself stays public by current behavior — but the **rows it
   lists** are filtered per viewer.
2. For each candidate row, include it only if it would be readable: `note.Free ==
   true`, or the viewer passes `env.CanReadNote(ctx, note)` — the same predicates
   that gate direct note access in `resolve.go` (the free/guest gate at `:288-290`,
   `CanReadNote` at `:309-316`; the `CanReadNote` Env method is declared at `:50`),
   respecting subgraph membership.
3. Default (`hide`): anonymous/unauthorized viewers see only the rows they can read;
   locked rows are **dropped**, not shown as teasers, so a base can't leak the titles
   of paywalled notes.

Apply the access predicate as part of the SQL query (or a post-filter on the SQL
result) **before** projection — so the page, the JSON endpoint, and the embed all
inherit the same per-viewer filtered set automatically.

**Optional future knob (not v1):** `paywall_rows: hide | show-locked` on the `.base`
file (default `hide`). `show-locked` would surface locked rows as teasers (title +
lock badge, no protected property values) for authors who explicitly want a
"members-only" preview. v1 ships `hide` only; the knob is recorded so the data model
leaves room for it.

---

## Inheritance: two views, not two files

When an author wants "two tables off one query" — e.g. a compact recent-posts table
and a full archive table — the answer is the **native Obsidian two-level model**, not
any custom inheritance:

- A single `.base` file declares base-level `filters` / `formulas` / `properties`.
- Its `views[]` array holds two (or more) views; **each view inherits the base level**
  and adds its own `filters` (ANDed), column `order`, `sort`, and `limit`.

`docs/demo/example_base.base` already demonstrates this: one file, two `table` views
("Table" and "telegram notes"), both inheriting the base-level `formulas`/`properties`.
The `internal/obsidianbase` parser must model this — base-level fields plus a
per-view slice, with `EffectiveFilter` ANDing the two levels (see
[parser package](#1-new-parser-package-internalobsidianbase)).

**Cross-*file* inheritance is a deliberate non-goal.** Obsidian has no
`extends: [[other.base]]`, and trip2g will not add one: it would make a `.base` file's
meaning depend on another file, breaking portability (drop the file into another
vault and it stops resolving). The in-file `views[]` model already covers the
"shared query, multiple tables" need.

**When do you actually need two views?** Only for a different **layout, column set, or
filter**. You do *not* need a second view to show "public vs members" rows — per-viewer
access control already differentiates **row visibility** automatically (see
[Access control](#access-control-per-viewer-per-row)). One view,
two audiences.

---

## JSON data access

The owner wants `.json` access to the resolved rows. Two delivery options:

| Option | Pros | Cons |
|--------|------|------|
| **Content negotiation** on the base path (`Accept: application/json`) | one URL, RESTful | needs a non-HTML Content-Type from the endpoint |
| **`/tasks.base.json` suffix** | trivial routing, cache-friendly, easy to link | second URL shape |

**Recommendation:** support both, but the suffix form is the pragmatic v1.

Dependency to call out: serving a non-`text/html` response currently needs the
**unimplemented `ResponseWriter` design** in `docs/dev/template_content_type.md`
(Content-Type is hardcoded to `text/html` in `rendernotepage/endpoint.go`).
Minimal version for v1: in the `.base` branch of `endpoint.go`, detect JSON intent
(suffix or `Accept`), set `ctx.Response.Header.SetContentType("application/json")`
directly, marshal `rows`, write, return — bypassing the default-template `Ctx`
entirely. This sidesteps the full ResponseWriter work for the single JSON case
while we wait for that design to land.

JSON shape (stable, projection-aware):

```json
{
  "view": "telegram notes",
  "columns": ["title", "telegram_publish_at"],
  "rows": [
    {"path": "/post-1", "permalink": "/post-1", "title": "...", "telegram_publish_at": "2026-06-25"}
  ],
  "total": 128,
  "offset": 0,
  "limit": 50
}
```

This endpoint doubles as the **pagination source** for large embedded tables.

---

## Embeds: `![[tasks.base]]`

Decision: **SSR the table server-side, layer JS on top** — not client-only. Better
SEO and first-paint, consistent with the mermaid argument.

The gap to close: raw-file embeds are currently dropped. In `link_renderer.go`
(`renderEmbed` at `:216`), `len(note.HTML) == 0 → WalkSkipChildren` (`:235-238`) —
and raw files always have empty `HTML`. Add a `.base`-specific branch *before* that
check: if the embedded note `IsBase()`, render its first view's table as a first SQL
page (capped, e.g. `LIMIT 25`) plus a JSON data-island and a "show more" hook,
instead of skipping. The capped SSR + lazy-load-more (SQL `OFFSET`) from the JSON
endpoint keeps large tables cheap on first paint.

This is analogous to how a normal embed emits
`<div class="embedded-note">…</div>` (`link_renderer.go:245`), but emits a
`<table data-base data-src="/tasks.base.json">` + `<script type="application/json">`
island instead.

---

## Magazine integration

### (a) `magazine_source: [[tasks.base]]`

Let the magazine take a `.base` as its data source — decouple **what** data
(the base query) from **how** it looks (magazine cards). Today
`MagazineItems()` (`magazine.go:29`) reads `magazine_include_files`,
`magazine_sort_property`, etc. (`template.go:305-344`) and builds its own in-memory
query.

Plan: if `magazine_source` resolves to a `.base` note, build the magazine's item
list from **that base's resolved rows** instead of the `magazine_*` knobs. The
base owns the filter/sort; the magazine owns the card layout. The existing
`magazine_*` properties remain as the no-base fallback. (Longer term, the magazine's
own query could move to the property index too — out of scope for v1.)

### (b) Magazine presets (product cleanup)

The owner finds the current magazine config "too for enthusiasts" — the
`magazine_*` knobs (`template.go:305-344`) are too much tuning. Propose **named
presets**:

```yaml
magazine_layout: blog   # blog | grid | list | masonry
```

Each preset sets sensible defaults (featured-first vs. uniform grid, card density,
image ratio). Authors pick a preset instead of hand-tuning. Combined with (a):
**bases = data, preset = presentation**. The granular knobs stay available for
power users but stop being the default surface.

---

## Formulas / computed columns

Out of scope for v1. When added, **reuse the jsonnet evaluator from
`internal/frontmatterpatch`** (`evaluate.go` `NewVM`, `patch.go` `WrapSource`) —
it already evaluates per-note expressions with `meta` and `path` external
variables and safe stack limits. A base formula column would compile each formula
once and evaluate per row, injecting the row's `RawMeta` as the `meta` external
var. Document this as a phase-3 item.

---

## Caching / storage

v1 needs **no result cache**. The base query runs as indexed SQL against the per-type
value tables with `LIMIT`/`OFFSET` — the property index *is* the optimization, so a
per-request query is already cheap. The index itself is the only new persistent
storage, rebuilt on note reload (see [Property
index](#frontmatter-property-index-per-type-value-tables)). Add a *result* cache only
if a base later proves expensive (formula columns, very wide projections) — and then
follow the **chartdata Service Package Pattern** (`internal/chartdata`):

- minimal `Env interface` the package needs (`cmd/server/chartdata.go:15`
  `var _ chartdata.Env = (*app)(nil)`),
- embedded anonymously in `app` so methods promote (`main.go:208`
  `*chartdata.ChartData`, constructed `main.go:404`),
- enqueue-on-miss + a debounced reload loop (`chartdata` `reloadDebounce = 2s`),
- a cache table keyed by `(version_id, base_hash)` storing `data_json TEXT`.

No asset storage is involved — a base produces no binary output (unlike charts'
echarts or future excalidraw SVGs).

---

## Open decisions

### Resolved by the owner

- **Query architecture → persistent SQL property index, five per-type value tables**
  (`note_string_values` / `note_int_values` / `note_real_values` /
  `note_bool_values` / `note_date_values`), mirroring the `form_*_values` house
  pattern — *not* a single `value_kind` table and *not* in-memory `NoteQuery`. The
  in-memory approach is kept only as a possible v0 spike. See [Property
  index](#frontmatter-property-index-per-type-value-tables). *(Carries a new SQL
  migration — pending owner confirmation before creation.)*
- **Locked rows → hidden by default**, per viewer. Anonymous/unauthorized viewers
  see only readable rows; locked rows are dropped (no teaser, no title leak). An
  optional `paywall_rows: hide | show-locked` knob (default `hide`) is recorded for
  later but is not v1. See [Access control](#access-control-per-viewer-per-row).
- **"Two tables" → two `views` in one `.base`** via native in-file inheritance; no
  cross-file `extends`. See [Inheritance](#inheritance-two-views-not-two-files).

### Still open (with recommendations)

1. **Draft vs. live note-set semantics for the index/query.**
   The page already picks `LatestNoteViews()` vs `LiveNoteViews()` per request
   (`resolve.go:196-211`, set on `response.Notes` at `:235`/`:259`). The index is
   keyed by `version_id`, so it can hold both draft and live versions. *Recommendation:
   a base queries the **same version set as the page it renders on*** — filter the
   SQL by the same version ids the page would expose, so a published base never
   surfaces unpublished drafts to visitors, and admins previewing `?version=latest`
   see drafts.

2. **Filter expression coverage in v1.**
   *Recommendation:* support `==`, `!=`, exists/`isEmpty()`/`!…isEmpty()`,
   `contains` (list props), and ordered comparison (typed at ingest). Emit a parse
   warning for unsupported expressions (e.g. regex, arbitrary formulas) rather
   than failing the whole base.

3. **JSON delivery shape.**
   *Recommendation:* ship `/tasks.base.json` (suffix) first; add `Accept`-based
   negotiation once `template_content_type.md`'s `ResponseWriter` lands.

4. **Dynamic vs. static base query in sqlc.**
   The base query is spec-driven, so it can't be one static sqlc query.
   *Recommendation:* build parameterized SQL in the new query path (static sqlc
   queries only for the index write/delete and simple counts). Revisit if the
   dynamic SQL gets unwieldy.

---

## Phased implementation plan

**Phase 0 — property index (gated on owner sign-off).**
- **Get owner confirmation for the new SQL migration** — a single migration file that
  creates the five per-type tables (`note_string_values` / `note_int_values` /
  `note_real_values` / `note_bool_values` / `note_date_values`) + their five indexes.
  Until then, Phase 0 is design-only — do not touch `db/schema.sql`.
- On sign-off: add the five tables + indexes to `db/schema.sql`, write per-type index
  read/write queries in `queries.read.sql` / `queries.write.sql`, run `make sqlc`.
- Populate the index on note reload (after `noteloader.Load()` /
  `PrepareLatestNotes`/`PrepareLiveNotes`, `cmd/server/main.go:492,508`):
  delete-by-version across all five tables + re-insert each note's classified
  `RawMeta`, keyed by `version_id`.
- If the migration is *not* yet confirmed, build the in-memory v0 spike instead
  (extend `NoteQuery` with `WhereMeta` + `compareValues` coercion) so Phase 1's
  renderer/template/JSON layers can proceed, then swap the query path to SQL.

**Phase 1 — minimal SSR page (the core).**
- `internal/obsidianbase` parser: two-level spec (base-level `filters`/`properties`
  + `views[]`), `Filter` tree (and/or/not + eq/ne/exists), `order` columns, `sort`,
  per-view `limit`, `EffectiveFilter` (base ∧ view). Tests with the
  `docs/demo/example_base.base` shapes (it has **two views**).
- `model.NoteView.Base` field + parse in `registerRawFile`.
- SQL query path: spec → `WHERE`/`ORDER BY`/`GROUP BY`/`LIMIT`/`OFFSET` over the
  per-type value tables, picking the table by the comparison literal's type (or the
  v0 in-memory `NoteQuery` spike if the index is not yet confirmed).
- Replace the `.base` stub arm in `endpoint.go:96-103` with a real renderer.
- `WriteBase` quicktemplate (table + cards reusing magazine markup) →
  `go generate` → commit `views.html.go`. SCSS for the table.
- Per-viewer access-control filtering in `resolveBaseRows`.
- `assets/base/` glue (client sort/filter) + npm script + `embed.go` + `base.js`
  conditional injection.
- Demo: `/example_base.base` renders a real table (both views); e2e fixture.

**Phase 2 — JSON access + embeds.**
- `/tasks.base.json` endpoint (minimal direct Content-Type write).
- `![[tasks.base]]` embed renderer in `link_renderer.go` (capped SSR + island +
  lazy-load-more from the JSON endpoint).
- `group` support → grouped table sections.

**Phase 3 — magazine + presets + formulas.**
- `magazine_source: [[x.base]]` wires the magazine to a base's rows.
- `magazine_layout` presets (blog/grid/list/masonry) over the five `magazine_*`
  knobs.
- Formula columns via the `frontmatterpatch` jsonnet evaluator.
- Caching (chartdata pattern) only if measured slow.

---

## File touchpoints

| Path | Change |
|------|--------|
| `db/schema.sql` | **NEW migration — PENDING OWNER CONFIRMATION.** One migration file creates **5 per-type tables** (`note_string_values` / `note_int_values` / `note_real_values` / `note_bool_values` / `note_date_values`) + **5 `(prop, value)` indexes** (mirror `form_*_values` at :737-754, keyed by `(version_id, prop, seq)`). Do **not** create until confirmed. |
| `queries.read.sql` | **NEW** — per-type index reads (e.g. counts); dynamic base query built in code over the per-type tables. `make sqlc` after editing |
| `queries.write.sql` | **NEW** — per-type delete + insert pairs (`Delete{String,Int,Real,Bool,Date}ValuesByVersion`, `Insert{String,Int,Real,Bool,Date}Value`). `make sqlc` after editing |
| `internal/db/queries.read.sql.go`, `queries.write.sql.go` | regenerated by `make sqlc` — commit |
| `cmd/server/main.go` | populate the per-type value tables on reload after `PrepareLatestNotes`/`PrepareLiveNotes` (:492,508) |
| `internal/mdloader/loader.go` | population hook reads each note's `RawMeta` (set at :756); parse `.base` in `registerRawFile` (sibling of the `.canvas` branch, :286-293) |
| `internal/basesquery/` (or renderer pkg) | **NEW** — spec → parameterized SQL over the per-type value tables (pick table by literal type; WHERE/ORDER BY/GROUP BY/LIMIT/OFFSET + per-viewer access predicate) |
| `internal/obsidianbase/base.go` | **NEW** — parse `.base` YAML → two-level `Base` spec (base-level + `views[]`, `EffectiveFilter`; mirror `obsidiancanvas`) |
| `internal/obsidianbase/base_test.go` | **NEW** — table-driven parse tests using `docs/demo/example_base.base` (two-view) shapes |
| `internal/model/note.go` | add `Base *obsidianbase.Base` field (next to `Canvas` at :282-284) |
| `internal/templateviews/query.go` | (v0 spike only) `WhereMeta`/`GroupBy` + `compareValues` coercion (:213-248); not needed once the SQL path lands |
| `internal/templateviews/note.go` | `IsBase()` / `HasBaseEmbed()` accessors (next to `HasCharts()` :169) |
| `internal/case/rendernotepage/endpoint.go` | replace `.base` stub arm (:96-103); JSON branch; append `/assets/base.js` in `buildDefaultTemplateCtx` (:475; appends at :507-513) |
| `internal/case/rendernotepage/resolve.go` | `resolveBaseRows` with per-viewer `Free`/`CanReadNote` filtering (reuse :288-290 + :309-316; `CanReadNote` Env at :50); pick note-set per request (:196-211) |
| `internal/mdloader/link_renderer.go` | `.base` embed branch before the empty-HTML skip (`renderEmbed` :216; skip at :235-238; normal embed write :245) |
| `internal/defaulttemplate/views.html` | replace `.base` stub ("Bases are not supported yet." at :536, in `UnsupportedFile` :527) with `WriteBase`/`Base(ctx)`; table + cards (reuse magazine markup) |
| `internal/defaulttemplate/views.html.go` | regenerated via `go generate ./internal/defaulttemplate/...` (`StreamUnsupportedFile` at :2314) — commit it |
| `internal/defaulttemplate/template.go` | `Ctx.Base` field; `magazine_layout` preset accessor (near magazine knobs :305-344) |
| `internal/defaulttemplate/magazine.go` | `magazine_source` → use a base's rows; preset defaults (`MagazineItems()` :29) |
| `assets/defaulttemplate/src/index.scss` | base table CSS; `npm run defaulttemplate-css` |
| `assets/base/src/index.ts` | **NEW** — glue: client sort/filter, lazy-load-more |
| `assets/base/esbuild.browser.mjs` | **NEW** — build `assets/base.js` |
| `assets/base.js` | **NEW** — committed artifact |
| `assets/embed.go` | add `base.js` to `//go:embed` (:8) |
| `package.json` | add `"base"` build script (near :18-20) |
| `docs/demo/example_base.base` | existing fixture (two views) — make it render; e2e |
| `e2e/vault.spec.js` | "Bases" tests (table renders, JSON endpoint, embed, per-viewer access control) |
