# Skill: Check Template Rendering

Use this to verify a layout renders without errors and inspect template variable values.

## How it works

`/_system/renderlayout` compiles and executes a Jet template server-side against a real note from the vault. It returns a `previewURL` — a direct link to the rendered HTML. No sync needed: share this link with the user while they're actively editing, it's faster than waiting for vault sync.

## Tool: scripts/trip2g-preview.mjs

The repo ships `scripts/trip2g-preview.mjs` — a CLI wrapper that reads the API key automatically from `.obsidian/plugins/trip2g/data.json` (walks up from cwd).

```bash
# run from vault dir (where .obsidian/ lives), e.g. docs/
# smoke test — no files needed
node scripts/trip2g-preview.mjs \
  --layout-src "{{ note.HTMLString() }}" --layout-path "/_debug.html" \
  --note-src "hello"
# → http://localhost:8081/_system/renderlayout?preview_id=abc123

# test a layout file against a real note
node scripts/trip2g-preview.mjs \
  --layout-file _layouts/mesh/index.html \
  --note-path /en/user/templates

# fetch rendered HTML directly
node scripts/trip2g-preview.mjs \
  --layout-src "{{ note.M().Debug() }}" --layout-path "/_debug.html" \
  --note-path /demo/template_meta_test \
  --fetch
```

Warnings and errors go to stderr. Exit code 1 on failure.
Share the printed URL with the user — they can open it in browser.

## Inspect variables with debug()

```jet
{{ debug(note.M()) }}
{* → *templateviews.Meta: &{raw:map[extra_content:[channels prices]]}
      methods: [Debug Get GetBool GetInt GetString GetStrings Has Raw] *}

{{ note.M().Debug() }}
{* → {"extra_content":["channels","prices"],"title":"Sales"} *}
```

> Always use parentheses when passing methods to `debug()`:
> `{{ debug(note.Title()) }}` ✓ → `string: My Title`
> `{{ debug(note.Title) }}` ✗ → `func() string: 0x...` (method reference, not value)

## Preview vs production: known differences

| Feature | Preview | Production |
|---------|---------|------------|
| autoimport components | ✅ works | ✅ works |
| `yield_blocks()` CSS | ✅ works | ✅ works |
| `{{ asset("file.css") }}` | ❌ returns path as-is | ✅ resolves to CDN URL |
| `htmlInjectionsHead` | ✅ empty slice (no error) | ✅ real injections |
| `note.M()` frontmatter | ✅ works (both `note.path` and `note.src`) | ✅ works |

`{{ asset() }}` doesn't resolve in preview because assets are tied to note version IDs in the production loader. Use `note.path` to get real asset URLs, or inline CSS/JS directly in the template during development.

## Rendering a single BEM component

To preview one component in isolation, explicitly import its file and yield the block.
See [[en/user/bem]] for BEM conventions and `@lid`/`@did` naming.

**Important caveats:**

- Import paths must be **relative** to the layout path (after `/_layouts/` prefix is stripped).
  From `/_layouts/mesh/_preview.html` → import `"bar"` resolves to `/mesh/bar` ✓
- `yield_blocks()` returns **empty** in preview (wire phase is skipped). Yield style blocks directly instead.
- Dependencies must be imported manually — autoimport doesn't work in preview.

```bash
# Render mesh/bar.html component with its styles
node scripts/trip2g-preview.mjs \
  --layout-path "/_layouts/mesh/_preview.html" \
  --layout-src '{{ import "_blocks" }}{{ import "bar" }}{{ import "button" }}<style>{{ yield _style_mesh_bar() }}{{ yield _style_mesh_button() }}</style>{{ yield mesh_bar() }}' \
  --note-src "hello"
```

Pattern:
1. `{{ import "_blocks" }}` — shared layout wrapper (if needed)
2. `{{ import "component" }}` — the component file (and its dependencies)
3. `<style>{{ yield _style_component() }}</style>` — inline CSS (instead of `yield_blocks`)
4. `{{ yield component() }}` — the HTML block

**Tip:** `/_system/renderlayout` without params always serves the **latest** render — keep it open in a browser while iterating.

## Jet range gotcha

Single-variable range gives the **index** (0, 1, 2…), not the value:

```jet
{* WRONG — item = 0, 1, 2 *}
{{ range item := note.M().GetStrings("extra_content") }}{{ item }}{{ end }}

{* CORRECT *}
{{ range i, item := note.M().GetStrings("extra_content") }}{{ item }}{{ end }}
```

