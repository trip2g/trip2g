# Note validators (`validate:`) — MVP

**Summary:** a note lists validator notes in its `validate` frontmatter. Each validator is a note with one or more ```` ```jsonnet ```` blocks that return messages. `error` becomes a `Critical` note warning and `warning` becomes a `Warning`. Nothing is blocked: the note is saved and published as usual. The sync CLI already exits 1 on `Critical`, and the admin dashboard lists the warnings.

## Contract

```yaml
validate: "[[article_lint]]"        # one target
validate:                           # or a list: every validator runs
  - "[[session_has_image]]"
  - validators/seo.md
```

Targets resolve with `NoteViews.ResolveWikilinkTarget`: a bare name, `./x`, a path with `/`, with or without `[[ ]]` and `.md`. An unquoted `[[x]]` also works: YAML reads it as a nested list, and the loader unwraps it. A target listed twice runs once.

Inside a jsonnet block the validator sees:

| Name | Value |
|---|---|
| `meta` | frontmatter **after** frontmatter patches (same local as in patches) |
| `path` | note path |
| `nodes` | flat document-order list of AST nodes |
| `std.native('regexMatch')(pattern, str)` | Go `regexp.MatchString`; jsonnet's stdlib has no regex |

A node looks like `{type, depth, line, ...}`:

| `type` | Extra fields |
|---|---|
| `heading` | `level`, `text` |
| `paragraph` | `text` |
| `image` | `src`, `alt` (both `![](x)` and `![[x.png\|alt]]`) |
| `link` | `url`, `text` |
| `wikilink` | `target`, `embed` |
| `code_block` | `lang` (`""` when missing), `line` is the opening fence |
| `list` | `ordered` |
| `list_item`, `blockquote`, `table` | — |

`line` is the line of the enclosing block in the source file (frontmatter included). Inline nodes (`link`, `image`, `wikilink`) carry their paragraph's line.

The block must evaluate to `[{level: 'error'|'warning', msg: string, line?: number}]`. Each message becomes a note warning: `validate <validator path>: line N: <msg>`.

These cases are all `Critical`: a `validate` value that isn't a note or a list of notes, a target that can't be found, a target that isn't a markdown note (`.canvas`, `.base`, `.excalidraw`), a validator without jsonnet blocks, a runtime error, a result that isn't such an array (`null` included), a message without `msg`, or an unknown level. A broken validator must not silently pass everything.

Jsonnet error positions are relative to the block, not to the validator file.

### Why a flat list, not a walker

Jsonnet can't pass a function into a native function: arguments are manifested as JSON, and a function fails with `couldn't manifest function as JSON`. A stateful `next()` iterator breaks too, because jsonnet is lazy. `[a(0), a(0), a(0)]` read as `xs[2], xs[0], xs[1]` returns `1, 2, 3`. A stateless `next(id)` cursor works only with `tailstrict`; without it, 20 000 nodes exceed `MaxStack = 500`. A flat list plus `std.foldl` gives a walker with a callback without either trap. See `session_has_image` in the demo vault for a fold that splits a note into sections.

## Where it runs

`mdloader.Load` → `ldr.validateNotes()` right after `resolveLangRedirects`. At that point all notes are registered (validators can sit anywhere in the vault), frontmatter patches are applied and every `NoteView` still holds its AST. `NoteView`s are rebuilt on each load, so warnings don't pile up between syncs.

Code: `internal/mdloader/validate.go`.

## Demo vault

`internal/mdloader/testdata/validate/` (7 validators in `validators/`, valid and invalid notes in `notes/`). It sits outside `docs/` on purpose: CI runs `trip2g lint docs`, and the deliberately invalid notes would fail it. `TestValidateDemoVault` pins the expected messages for each note.

## Cost

`BenchmarkValidateDocs` loads the repo's `docs/` vault (857 notes) with every note listing the demo validators (12 threads, `-benchtime=5x -count=3`):

| Validators per note | `Load` time | Added per note | Allocated |
|---|---|---|---|
| none | ~0.49 s | — | 464 MB |
| `required_frontmatter` (meta only) | ~0.61–0.71 s | ~0.15 ms | 630 MB |
| `session_has_image` (fold over nodes) | ~0.77 s | ~0.33 ms | 910 MB |
| all 7 | ~3.7–4.4 s | ~4 ms (~0.6 ms per validator) | 4.3 GB |

**This cost is paid on every push batch, for the whole vault.** `pushNotes` calls `PrepareLatestNotes`, which reruns the full `Load`. It isn't limited to the notes that changed.

A CPU profile of the `all 7` case shows where the time goes. Jsonnet evaluation takes about 90% of `validateNotes`. Re-parsing the snippet on every call (`SnippetToAST`) is about 25%, rebuilding `nodes` per validator about 20%, and turning `nodes` into jsonnet values about 15%. Allocation pressure (GC) is the rest.

The MVP leaves these out:
- parse each validator once (`jsonnet.SnippetToAST` + `vm.Evaluate`);
- build `nodes` once per note, not once per validator;
- cache results by (note content hash, validator hash, patch-set hash), the same way `frontmatterpatch.Cache` does;
- validate notes in parallel (one VM per worker).

## Out of the MVP

Binding by `include`/`exclude` globs, `type: validator`, selectors over nodes, vault-wide rules (uniqueness, backlinks), the caching and parallelism listed above, and user docs.

Also missing are a time limit and a memory limit for a validator. `import` is blocked and `MaxStack` stops deep recursion, but a wide loop like `std.makeArray(1e8, ...)` still runs inside the full-vault `Load` on every push. Frontmatter patches carry the same risk today. Validators multiply it, because each one runs once for every note that lists it.
