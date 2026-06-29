# `trip2g lint <dir>` — DB-free doc/note linter

**Вывод:** `trip2g lint <dir>` runs trip2g's **real** note-loading pipeline over a filesystem directory — no database, no server, no browser — and prints the warnings the loader produces (broken/ambiguous wikilinks incl. cross-language leaks, broken images, frontmatter-patch failures, layout render errors). It replaces the fragile `scripts/check-doc-lang-links.sh` heuristic with the engine's actual link resolution, so CI and authors catch real problems locally.

## Usage

```
trip2g lint <dir>            # lint a docs/notes directory
trip2g lint docs             # e.g. the repo's docs vault
```

- Output: `path:line: <level> <message>` (one warning per line).
- Exit `0` when clean (or only baselined warnings); exit `1` when new warnings exceed the threshold.
- It's a subcommand on the existing server binary — an `os.Args[1]` switch at the top of `cmd/server/main.go`, **before** `appconfig`/DB init, so the lint path never opens SQLite.

## What it checks

Everything the loader already records as a `model.NoteWarning`, drained from two sinks after `loader.Load()`:

| Source | Examples |
|--------|----------|
| `nvs.Warnings()` (per note) | broken wikilink, broken/missing image, missing asset, embedded note not found, `lang_redirect` target issues, **frontmatter-patch failures**, vault-patch compile errors, metadata warnings |
| `layouts.Map[*].Warnings` (per layout) | Jet layout **render errors/panics** (from `smokeRenderLayouts`, built explicitly "so CLI/pushNotes flows surface them without a browser") |

Plus a lint-added check: **cross-language wikilink leaks** (a bare `[[name]]` in a `docs/ru/` note resolving to a `docs/en/` page, or vice versa).

## Why it's more accurate than the old bash script

The retired `check-doc-lang-links.sh` assumed trip2g resolves a bare `[[name]]` by **alphabetical** full-path order (so `en/` always wins). **That is wrong.** The real resolver (`internal/mdloader/loader.go`, `extractInLinks`) picks the **shortest path depth**, and breaks depth-ties by **Go map iteration order** — i.e. non-deterministic, not "en wins". The resolver also does **not** flag ambiguity at all.

`trip2g lint` instead reads each note's **`ResolvedLinks`** (raw target → resolved permalink) and flags a leak when a bare wikilink in an `en/`-prefixed note resolves under `ru/` (or vice versa) — keyed on **path prefix**, not the frontmatter `lang:` field (which most docs omit). This is exact (zero false positives) and independent of the resolver's tie-breaking subtleties. Ambiguous bare wikilinks (`>1` candidate) are detected via `nvs.BasenameMap`.

## How it runs DB-free

`internal/noteloader` + `internal/mdloader` are decoupled from the DB behind a 13-method `noteloader.Env` interface that consumes `db.*` only as struct types, never a live connection. `trip2g lint` provides a minimal **filesystem-backed Env**: `RawNotes` walks the dir and reads `*.md` (+ `_layouts/*.html`, `.canvas/.base/.excalidraw`) with synthetic ids; the rest return empty/no-op (`localstorage`'s DB-free asset methods are reused to suppress false image warnings). `loader.Load(ctx, LoadOptions{SkipSearchIndex: true})` skips bleve.

## CI

`.github/workflows/ci.yml` runs **`trip2g lint docs`** over the **whole `docs/` tree** (replacing `scripts/check-doc-lang-links.sh` + its baseline). Keep a baseline-ratchet (grandfather pre-existing warnings; fail only on new ones) so adoption is incremental.

## Test cases — `docs/demo/lint/*`

Deliberate known-bad fixtures live under `docs/demo/lint/` so the linter is tested against inputs whose warnings are known, e.g.:

- a **broken wikilink** (`[[does-not-exist]]`),
- a **cross-language leak** (`docs/demo/lint/ru/*` with a bare `[[name]]` that resolves to a `docs/demo/lint/en/*` page),
- an **ambiguous bare wikilink** (same basename in two dirs),
- a **broken image** (`![[missing.png]]`),
- a **frontmatter-patch failure** and a **layout render error**.

A Go test loads `docs/demo/lint/` through the lint pipeline and asserts the expected warnings are reported (and clean fixtures produce none) — so the linter's detection is regression-covered, not just smoke-run.

## Related

- Rejected sibling: `trip2g sync` — see `docs/dev/trip2g_sync_subcommand.md`.
- `docs/dev/obsidian_links.md` (wikilink resolution), `internal/noteloader/{loader,smoke_render}.go`, `internal/mdloader/loader.go`, `internal/frontmatterpatch/evaluate.go`, `internal/localstorage/storage.go`.
