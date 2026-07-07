# Call knowledge base: presentation layer on the default template

**TL;DR.** No backend changes needed. The homepage is an `index.md` with `content: [magazine]` and `magazine_include_files: "calls/**/*.md"`; terms live at `concepts/index.md` with the same mechanism sorted by a pipeline-computed `mentions` number; daily notes get their own magazine at `daily/index.md`. Navigation is a bullet list in `_header.md`. The load-bearing contract point: every note must carry `created_at` as an RFC3339 string in frontmatter. A key named `date` or `created` is silently ignored and the note falls back to DB sync time, which breaks sorting and daily bucketing.

Ready-to-use note templates live in `/home/alexes/krisp-run/vault-templates/` (personal vault, not this repo).

## Verified template behavior (against current code)

| Fact | Source |
|------|--------|
| Magazine mode = `content:` list containing the string `magazine` | `widgets.go` `parseContentRef` (line 61), `views.html` line 270 |
| Root URL with no index note renders a default magazine of everything | `rendernotepage/resolve.go:245`, `template.go:254` |
| A note named `index.md` (or `_index.md`) gets permalink `/`; `concepts/index.md` gets `/concepts` | `model/note.go:470` (trailing `index` path component dropped) |
| Include glob `magazine_include_files` (default `**/*.md`), plus `magazine_exclude_files`, `magazine_include_property` (presence only, value ignored), `magazine_exclude_property` | `template.go:336-380`, `magazine.go` |
| Sort: with `magazine_sort_property` set, notes having that property come first sorted **desc only**, then the rest by `created_at` desc; without it, everything by `created_at` desc | `magazine.go:41-48` (`.Desc()` is hardcoded) |
| `CreatedAt()` = frontmatter `created_at` (checked first) or `created_on`, **string value**, parsed as RFC3339 / `2006-01-02 15:04[:05]` / `2006-01-02`; otherwise falls back to DB creation time | `model/note_created_at.go:8-70` |
| Card renders: title, excerpt, read-more, `CreatedAt().Format("2006-01-02")`. No image in the card (the doc's `FirstImageURL()` mention is outdated) | `views.html:360-390` |
| Excerpt = `PartialRenderer().Introduce()` = all content **before the first heading** or thematic break. A body that starts with `# H1` yields an empty excerpt | `mdloader/partial_renderer.go:225-261` |
| Title = frontmatter `title` > leading H1 > filename | `model/note.go:948-970` |
| Header/footer: per-note `header:`/`footer:` key > `_layouts` glob section > `_header.md`/`_footer.md` at vault root | `template.go:296-330` |
| Header nav = first markdown list in the header note (`FirstListHTML()`); first image = logo | `views.html:392-421` |
| Footer: intro paragraph = brand block; 2+ `##` sections = link columns; else first list inline | `views.html:423-445` |
| System notes (any path component starting with `_`) are excluded from magazines and not publicly routed | `magazine.go:58`, `rendernotepage` systemRE |
| The index note excludes itself from its own magazine | `magazine.go:54` |
| Sidebars appear only if `_left_sidebar`/`_right_sidebar` notes or layout sections exist | `template.go:183-232` |
| Featured card = first item after sort; tiers: index 0 featured, 1-4 grid, 5+ list | `magazine.go:72-77` |

## Vault layout

```
index.md              → /            magazine of calls, newest first
daily/index.md        → /daily       magazine of daily notes, newest first
concepts/index.md     → /concepts    magazine of terms, most-mentioned first
calls/2026-07-02_acme-demo.md        one note per call
daily/2026-07-02.md                  one note per day, links the day's calls
concepts/hot-auth-token.md           one note per term
_header.md                           site nav (system note, not routed)
_footer.md                           footer (system note, not routed)
```

Globs map 1:1 to folders: `calls/**/*.md`, `daily/*.md`, `concepts/**/*.md`. The three index notes match their own globs but are auto-excluded as the current note.

## Magazine-mode frontmatter (exact)

`index.md` (homepage, calls):

```yaml
---
title: Звонки
content:
  - magazine
magazine_include_files: "calls/**/*.md"
---
```

No `magazine_sort_property` needed: the default sort is `created_at` desc, and `created_at` carries the call's true start instant.

`daily/index.md`:

```yaml
---
title: По дням
content:
  - magazine
magazine_include_files: "daily/*.md"
---
```

`concepts/index.md`:

```yaml
---
title: Термины
content:
  - magazine
magazine_include_files: "concepts/**/*.md"
magazine_sort_property: mentions
---
```

## Frontmatter contract (authoritative for the krisp pipeline)

### The date key (load-bearing)

- The key is **`created_at`** (fallback `created_on`). `date` and `created` are silently ignored; the note then sorts by DB sync time, which is wrong.
- The value must be a **string**. Quote it so YAML does not produce a timestamp object (the parser warns "expected string" and falls back).
- Emit **full RFC3339 with timezone offset**, decoded from the krisp UUIDv7 id: `created_at: "2026-07-02T15:30:00+03:00"`. Timezone-less forms parse in the site location, which reintroduces the wrong-local-clock problem.

### Call note (`calls/YYYY-MM-DD_slug.md`)

```yaml
---
title: "Созвон с ACME: демо продукта"
title_source: krisp          # krisp | inferred
created_at: "2026-07-02T15:30:00+03:00"
type: call
source: krisp
speakers:
  - Алексей
  - Мария
needs_review: false
---
Разбор одним сильным абзацем: о чём был звонок, что решили, что дальше.
Этот абзац становится карточкой на главной.

## Разбор
...

## Термины
- [[hot-auth-token|Hot Auth Token]]
```

Body rules:
- **No leading H1.** The title lives in frontmatter; a body starting with `# ...` makes the card excerpt empty.
- First paragraph = the summary, 1-3 sentences, plain prose (no list, no callout). Everything before the first `##` goes into the card.
- Section headings start at `##`.

### Daily note (`daily/YYYY-MM-DD.md`)

```yaml
---
title: "2026-07-02, среда"
created_at: "2026-07-02T00:00:00+03:00"
type: daily
calls_count: 3
---
3 звонка: демо ACME, синк с Марией, интервью кандидата.

## Звонки
- 15:30 [[2026-07-02_acme-demo|Созвон с ACME: демо продукта]]
- ...

[[2026-07-01|← Вчера]] · [[2026-07-03|Завтра →]]
```

- `created_at` at midnight of that day, with offset, so the daily magazine sorts by calendar day.
- Prev/next-day navigation is **plain wikilinks written by the pipeline**; the template has no automatic prev/next. A link to a not-yet-existing next day renders as a dead link, so the pipeline should add the "Завтра" link retroactively when it creates the next daily note (or omit it).

### Concept note (`concepts/slug.md`)

```yaml
---
title: Hot Auth Token
type: concept
created_at: "2026-06-14T00:00:00+03:00"   # first seen
aliases:
  - HAT
mentions: 7                    # int, NOT quoted; count of calls mentioning the term
last_mentioned: "2026-07-02"
---
Определение в одно-два предложения. Попадает в карточку на /concepts.

## Контекст
...

## Упоминания
- [[2026-07-02_acme-demo]]
```

- `mentions` must be a YAML **int** (unquoted). `compareValues` is type-strict: a quoted `"7"` compares as string against ints and sorting degrades to 0.
- The pipeline must **rewrite `mentions` on every re-process** of a call that references the term (updateNotes handles in-place frontmatter patches).
- `last_mentioned` is a spare sort key: ISO date strings compare correctly as strings, so switching the concepts index to `magazine_sort_property: last_mentioned` gives "recently discussed first" with no other changes.

## Header and footer

`_header.md` — nav is the first bullet list; header = Calls / By day / Terms:

```markdown
---
title: Header
free: true
---
- [Звонки](/)
- [По дням](/daily)
- [Термины](/concepts)
```

`_footer.md` — first paragraph = brand line, then one list (inline links):

```markdown
---
title: Footer
free: true
---
База звонков: Krisp → разбор → термины.

- [Звонки](/)
- [По дням](/daily)
- [Термины](/concepts)
```

## Honest gaps (default template cannot do this today)

1. **No ascending sort.** `magazine.go` hardcodes `.Desc()`. Alphabetical A→Z for terms is impossible; that is why concepts sort by `mentions` desc (frequency), which the pipeline must compute and keep fresh. `last_mentioned` desc is the fallback if counting proves noisy.
2. **No pinning.** The featured (big) card is always the first item after sort. You cannot pin a specific call or term.
3. **Card date is always `CreatedAt()` as `YYYY-MM-DD`.** Call time of day is not shown on cards, only inside the note.
4. **`magazine_include_property` checks presence, not value.** `type: call` vs `type: concept` cannot be distinguished by property filters; folders + globs are the real separation axis.
5. **No prev/next-day navigation primitive.** Daily notes chain via pipeline-written wikilinks only.
6. **Cards have no images.** `MagazineCard` renders title/excerpt/date only.
7. **Empty excerpt trap.** Any note whose body starts with a heading gets a card without an excerpt. The contract (frontmatter `title`, first paragraph before any `##`) exists to avoid this.

## Mapping to fleet KB-construction

The contract is exactly the "presentation half" of the fleet segment→wiki-extract pipeline:

- **Ingest per source** stays untouched; only the emit step must produce the frontmatter above.
- `created_at` from UUIDv7 makes call ordering source-authoritative, independent of sync time — the same property the KB-construction engine needs for re-processability (re-emitting a note never changes its position).
- `mentions`/`last_mentioned` on concepts are the first cross-source graph metrics surfaced in the UI; the same counters generalize to books/YouTube sources later.
- Daily notes are a pure derived view (bucketing by `created_at` date), so they can be regenerated from scratch at any time without touching call notes.
