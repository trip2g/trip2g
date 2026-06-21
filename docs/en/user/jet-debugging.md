---
title: Debugging Jet templates
free: true
lang: en
lang_redirect: "[[ru/user/jet-debugging]]"
---

When a layout condition doesn't fire or a frontmatter field comes out blank, guessing is slow. Two built-in functions let you see exactly what the template engine sees: `debug()` and `note.M().Debug()`.

These are for use inside custom Jet layouts in `_layouts/`. If you use the default template without custom HTML, you don't need this page.

### `note.M().Debug()` — dump all frontmatter

`note.M()` returns the note's metadata object. Calling `.Debug()` on it prints all frontmatter keys as a JSON string, inline in the rendered page:

```jet
{{ note.M().Debug() }}
```

Output:

```
{"extra_content":["channels","prices"],"layout":"my-page","title":"My Page"}
```

This answers the question "what frontmatter does the template actually receive?" — useful when a `{{ if note.M().Has("featured") }}` block never renders and you want to confirm the key is present.

### `debug()` — inspect any expression

The global `debug()` function takes any expression and returns its Go type, current value, and full list of available methods:

```jet
{{ debug(note.M()) }}
```

Output (rendered as inline text):

```
*templateviews.Meta: &{raw:map[title:My Page extra_content:[a b]]}
      methods: [Debug Get GetBool GetInt GetString GetStrings Has Raw]
```

You can pass any expression — a string, a number, the result of a method call:

```jet
{{ debug(note.Title()) }}
{* → string: My Page *}

{{ debug(note.M().GetStrings("tags", nil)) }}
{* → []string: [go obsidian] *}
```

**Always call methods with parentheses.** `debug(note.Title())` inspects the string "My Page". `debug(note.Title)` inspects the function reference itself — not what you want.

### Before and after: why a condition fails

Suppose you have a layout that shows a "Featured" badge on certain notes:

```jet
{{ if note.M().Has("featured") }}
  <span class="badge">Featured</span>
{{ end }}
```

The badge never appears. You add a debug call to check what the template sees:

```jet
{{ note.M().Debug() }}
{{ if note.M().Has("featured") }}
  <span class="badge">Featured</span>
{{ end }}
```

The output shows:

```
{"layout":"article","title":"Top Post"}
```

The key `featured` is missing. Back in Obsidian, the note has `Featured: true` — capital F. Frontmatter keys are case-sensitive. Rename the property to `featured: true`, sync, and the badge appears.

Without `debug()`, you'd be checking sync status, cache, or template logic. With it, you see the real data in one render.

### Workflow: debug without uploading

You don't have to sync a file to the vault every time you add a debug call. Use `/_system/renderlayout` to render any template against any note content from the terminal — see [[renderlayout]] for the full API.

Quick example — check what frontmatter a note exposes:

```bash
python3 ../scripts/renderlayout.py \
  --layout-src '{{ note.M().Debug() }}' \
  --layout-path '/_debug.html' \
  --note-path /my-post \
  --fetch
```

The `--fetch` flag prints the rendered HTML directly to stdout. No browser needed, no file sync.

### Remove debug calls before publishing

`debug()` and `.Debug()` output raw Go internals directly into your HTML. Readers see this text on the page. Remove all debug calls before syncing a final version.

A safe pattern during development: put debug calls inside Jet comments so they render but are easy to find and delete:

```jet
{* DEBUG: {{ note.M().Debug() }} *}
```

Jet comments don't appear in the output — but that also means you won't see the result. Use a plain `{{ }}` block while debugging, switch to a `{* *}` comment when you're done checking, then delete the line entirely before publishing.

### Related

- [[templates]] — how Jet layouts work, available variables, and template syntax
- [[renderlayout]] — render a layout against any note without uploading files
