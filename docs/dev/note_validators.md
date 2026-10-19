# Note validators (`validate:`) — MVP

**Summary:** a note lists validator notes in its `validate` frontmatter. Each validator is a note with one or more ```` ```jsonnet ```` blocks that return messages. `error` becomes a `Critical` note warning and `warning` becomes a `Warning`. Nothing is blocked: the note is saved and published as usual. The sync CLI already exits 1 on `Critical`, and the admin dashboard lists the warnings.

## Contract

```yaml
validate: "[[article_lint]]"        # one target
validate:                           # or a list: every validator runs
  - "[[session_has_image]]"
  - validators/seo.md
```

Targets resolve with `NoteViews.ResolveWikilinkTarget`, after dropping a `#fragment`: a bare name, `./x`, a path with `/`, with or without `[[ ]]` and `.md`. An unquoted `[[x]]` also works: YAML reads it as a nested list, and the loader unwraps it. A target listed twice runs once.

Inside a jsonnet block the validator sees:

| Name | Value |
|---|---|
| `meta` | frontmatter **after** frontmatter patches (same local as in patches) |
| `path` | note path |
| `nodes` | flat document-order list of AST nodes |
| `std.native('regexMatch')(pattern, str)` | Go `regexp` match; jsonnet's stdlib has no regex. Each pattern compiles once per load |

A node looks like `{type, depth, line, ...}`:

| `type` | Extra fields |
|---|---|
| `heading` | `level`, `text` |
| `paragraph` | `text`; a tight list item's `TextBlock` is emitted as a `paragraph` too |
| `image` | `src`, `alt` (both `![](x)` and `![[x.png\|alt]]`) |
| `link` | `url`, `text` |
| `wikilink` | `target`, `embed` |
| `code_block` | `lang` (`""` when missing), `line` is the opening fence |
| `list` | `ordered` |
| `list_item`, `blockquote`, `table` | — (table cell text is not emitted) |

`line` is the line of the enclosing block in the source file (frontmatter included). Inline nodes (`link`, `image`, `wikilink`) carry their paragraph's line.

The block must evaluate to `[{level: 'error'|'warning', msg: string, line?: number}]`. Each message becomes a note warning: `validate <validator path>: line N: <msg>`.

A missing `validate`, `null`, and a blank string all mean no validators, so a patch returning `{ validate: null }` switches validation off for a note. A blank item inside a list is still an error.

`std.trace` output is discarded (`jsonneteval.NewVM`, shared with frontmatter patches and webhook transforms): the author can't read the server's stderr, and anyone who can edit a note could otherwise write to it on every load.

These cases are all `Critical`: a `validate` value that isn't a note or a list of notes, a target that can't be found, a target that isn't a markdown note (`.canvas`, `.base`, `.excalidraw`), a validator without jsonnet blocks, a runtime error, a result that isn't such an array (`null` included), a message without `msg`, or an unknown level. A broken validator must not silently pass everything.

Jsonnet error positions are relative to the block, not to the validator file.

### Why a flat list, not a walker

Jsonnet can't pass a function into a native function: arguments are manifested as JSON, and a function fails with `couldn't manifest function as JSON`. A stateful `next()` iterator breaks too, because jsonnet is lazy. `[a(0), a(0), a(0)]` read as `xs[2], xs[0], xs[1]` returns `1, 2, 3`. A stateless `next(id)` cursor works only with `tailstrict`; without it, 20 000 nodes exceed `MaxStack = 500`. A flat list plus `std.foldl` gives a walker with a callback without either trap. See `session_has_image` in the demo vault for a fold that splits a note into sections.

## Where it runs

`mdloader.Load` → `ldr.validateNotes()` right after `resolveLangRedirects`, only when `Options.Validate` is set. At that point all notes are registered (validators can sit anywhere in the vault), frontmatter patches are applied and every `NoteView` still holds its AST. `NoteView`s are rebuilt on each load, so warnings don't pile up between syncs.

`noteloader.Loader.EnableValidation` sets it. Two loaders call it:
- the latest loader (`createNoteLoaders` in `cmd/server/boot.go`). Every reader of note warnings reads the latest notes: the admin Note Warnings list (`LatestNoteViews`), `pushNotes` and `commitNotes` (`PrepareLatestNotes`), and Telegram publish (`handletgpublishviews`);
- `trip2g lint` (`internal/doclint`).

The live loader skips validation: nothing reads its warnings. So does the single-note loader behind `LoadNoteViewByVersionID`. It loads one version without the rest of the vault, so every validator would come out as `target not found`.

`noteloader.Loader.Load` does all its database reads (site config, patches, notes, assets, subgraphs, chunks) before `mdloader.Load`. Parsing, validating and rendering a large vault can use up the caller's deadline, such as the 60 s startup budget in `loadAllNotes`, and none of that time is spent before a database read.

Code: `internal/mdloader/validate.go`.

## Demo vault

`internal/mdloader/testdata/validate/` (8 validators in `validators/`, valid and invalid notes in `notes/`). It sits outside `docs/` on purpose: CI runs `trip2g lint docs`, and the deliberately invalid notes would fail it. `TestValidateDemoVault` pins the expected messages for each note.

## Cache

`ValidateCache` (`internal/mdloader/validate_cache.go`) keeps each note's validator messages between loads. `noteloader.Loader` owns one per loader, next to the frontmatter-patch cache. It is in memory: a restart validates the whole vault once.

An entry is stored per note path, under a key that hashes:
- the note's path and content;
- the patch-set hash, so **any** patch change misses every entry. Working out which notes a patch touches is not worth it;
- the path and content of each validator the note resolved to, so an edited validator reruns only the notes that list it. A validator that now resolves to a different note changes the key too.

After each load the cache keeps only the notes that load validated, so deleted or renamed notes don't linger.

Messages about a `validate` value that is broken or a target that can't be resolved are recomputed on every load. They are cheap and never cached.

## Cost

`BenchmarkValidateDocs` loads the repo's `docs/` vault (857 notes) with every note listing the demo validators (12 threads, `-benchtime=5x -count=2`). The `cached` rows warm one `ValidateCache` with a `Load` before measuring, so each iteration is a sync where nothing changed.

| Validators per note | No cache | Cached, nothing changed | Allocated (no cache) |
|---|---|---|---|
| none | ~0.52–0.57 s | ~0.51–0.56 s | 466 MB |
| `required_frontmatter` (meta only) | ~0.67 s | ~0.49–0.61 s | 634 MB |
| `session_has_image` (fold over nodes) | ~0.90 s | ~0.50 s | 951 MB |
| all 7 | ~4.3 s | ~0.49–0.53 s | 4.5 GB |

The full `Load` runs on every push batch: `pushNotes` calls `PrepareLatestNotes`, which reloads the whole vault. With the cache, only notes whose key changed run jsonnet. A full pass, the "no cache" column, still runs on server start (the latest loader only, after its database reads), when a patch is saved in the admin panel (inside that mutation), and on a sync that carries a patch note.

A CPU profile of an uncached `all 7` run shows where a full pass goes. Jsonnet evaluation takes about 90% of validation. Re-parsing the snippet on every call (`SnippetToAST`) is about 25%, rebuilding `nodes` per validator about 20%, and turning `nodes` into jsonnet values about 15%. Allocation pressure (GC) is the rest.

Not done yet, and only worth doing if a full pass turns out to hurt:
- parse each validator once (`jsonnet.SnippetToAST` + `vm.Evaluate`);
- build `nodes` once per note, not once per validator;
- validate notes in parallel, with one VM per worker;
- persist results (version id → messages) so a restart doesn't revalidate.

## Out of the MVP

Binding by `include`/`exclude` globs, `type: validator`, selectors over nodes, and the speed-ups listed above. Vault-wide rules (orphans, folder indexes, `supersedes`, `stale_after` with a clock) are planned as a separate, uncached vault-validator type: https://github.com/trip2g/trip2g/issues/436. Reading another note from a note validator (`std.native('note')`) was deferred, because the per-note cache key would have to track every note it read.

Also missing are a time limit and a memory limit for a validator. `import` is blocked and `MaxStack` stops deep recursion, but a wide loop like `std.makeArray(1e8, ...)` still runs inside the full-vault `Load` on every push. Frontmatter patches carry the same risk today. Validators multiply it, because each one runs once for every note that lists it.
