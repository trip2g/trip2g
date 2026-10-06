---
title: "Template components"
free: true
wide: true
lang: en
lang_redirect: "[[ru/user/components]]"
---

Build custom layouts from components. A component is a Jet `block` in its own file under `_layouts/`, called from a page with `{{ yield name(param="value") }}`. Name its block with `@lid` and its CSS classes with BEM and `@did`, and let trip2g import it automatically: no `{{ import }}` needed. Collect the components' CSS with `yield_blocks("_style_")`. The [[#Best practices]] list below is the short version of this page.

New to custom layouts? Start with [[en/user/templates|Templates]]. Every function and construct is in the [[en/user/jet-functions|Jet functions reference]].

### What a component is

A component is one file in `_layouts/` that defines:

- an HTML block, named after the file: `{{ block @lid(...) }}`
- optionally a CSS block, `{{ block _style_@lid() }}`, and a JS block, `{{ block _js_@lid() }}`

A page calls the HTML block with `yield`. trip2g finds the file that defines the block, imports it, and puts the CSS of every component the page uses into one `<style>` tag.

### File layout and block names

```
_layouts/
├── page.html
└── components/
    ├── button.html
    └── card.html
```

Every `.html` file under `_layouts/` can be a component, in any folder. `components/` is a convention, not a requirement. Blocks are looked up across **all** layout files of the site, so a page in one folder can yield a block defined in another.

Before parsing a file, trip2g replaces two placeholders with names derived from the file's path relative to `_layouts/`:

| File | `@lid` (block names) | `@did` (CSS classes) |
|---|---|---|
| `button.html` | `button` | `button` |
| `components/card.html` | `components_card` | `components-card` |
| `site/components/card.html` | `site_components_card` | `site-components-card` |
| `my-theme/card.html` | `my_theme_card` | `my-theme-card` |

A page calls `components/card.html` as `{{ yield components_card(...) }}`. The full placeholder reference, including `@@lid` for a literal `@lid`, is in [[en/user/yield_blocks|yield_blocks]].

`@lid` turns every character that can't be part of a Jet name (`/`, `-`, `.`, a space) into `_`, and puts `_` before a leading digit: `2col/card.html` gives `_2col_card`. `@did` only turns `/` into `-`. So start folder and file names with a letter and keep `.` and spaces out of them: `@did` keeps those characters, and CSS can't select a class like `2col-card` or `v1.2-card` as written.

### Calling a component

```jet
{{ yield components_button(label="All posts", url="/blog") }}
```

- **Pass parameters by name.** A positional argument, `yield components_button("All posts")`, is ignored: the parameter keeps its default, or prints `false` when it has none.
- **Give every parameter a default** in the definition: `{{ block @lid(label="", url="", featured=false) }}`. A parameter you don't pass then has a known value.
- **Wrap content** with `content` after the call. The component prints it with `{{ yield content }}`, and prints nothing there when the caller passes none:

  ```jet
  {{ yield components_card(title="Docs", url="/docs") content }}
    <p>Anything here becomes the card body.</p>
  {{ end }}
  ```

- `content` is reserved. A parameter named `content` stops the component file from loading.
- A value after the call becomes `.` inside the block: `{{ yield components_list() items }}`.

### Auto-import

A page doesn't need `{{ import }}` to use a component. Every component the page calls with `{{ yield name(...) }}` is found and connected automatically, together with the components those call in turn: a card that yields a button brings in the button file too. Just call the block by name.

The lookup reads the template source, not the rendered output, so a component is connected even when its `yield` sits inside an `if` that is false on this request. How the loader does this is described for trip2g developers in [[dev/layouts]] (section "Автоимпорт компонентов"); layout authors don't need it.

**Collisions.** Two files that define a block with the same name don't stop the load:

- Between component files, the file whose path sorts first wins. `a/card.html` beats `b/card.html`. A page that calls `yield_blocks` shows the warning `block "card" defined in both /a/card and /b/card` in the layout preview. A page without `yield_blocks` gets no warning. `@lid` names make this collision impossible, because two files never share a path.
- A block the page defines itself wins over a component's block of the same name. Remember that a `{{ block }}` written in a page also renders where it is defined.

**When an explicit `{{ import }}` is still needed.** Auto-import runs for the page being rendered. A page that starts with `{{ extends }}` gets it too. It doesn't reach into other templates the page pulls in:

| Case | What happens | Fix |
|---|---|---|
| A template pulled in with `{{ include "partials/x" }}` yields a component | Render stops with `unresolved block "components_card"` | Add `{{ import "components/card" }}` at the top of the included template, or turn the partial into a component and yield it |
| A parent template used through `{{ extends "base" }}` yields a component | Render stops with `unresolved block …` | Add `{{ import "components/card" }}` at the top of the parent template |
| A file run with `exec("lib/x")` yields a component | Render stops with `unresolved block …` | Add `{{ import "components/card" }}` at the top of that file |

An explicit `{{ import }}` together with auto-import of the same file works, so adding one never breaks a page.

A `yield` of a block that no file defines fails at render time with `unresolved block "name"`. Admins see the error on the page and in `/_system/renderlayout`.

### Component CSS: BEM and `yield_blocks`

Each component keeps its CSS in a `_style_@lid` block, with class names in BEM form built from `@did`:

```jet
{{ block _style_@lid() }}
.@did { display: block; }
.@did--featured { border-color: #0070f3; }
.@did__title { font-size: 1.25rem; }
{{ end }}
```

| BEM part | Form | Example for `components/card.html` |
|---|---|---|
| Block | `.@did` | `.components-card` |
| Element | `.@did__name` | `.components-card__title` |
| Modifier | `.@did--name` | `.components-card--featured` |

The page writes the CSS once, in `<head>`:

```jet
<style>{{ yield_blocks("_style_") }}</style>
```

`yield_blocks` includes the `_style_` blocks of only the components this page reaches, so a page without a card carries no card CSS. Keep a `_style_` block free of parameters and of `yield`: it is called without arguments, and a style block that yields another style block prints that CSS twice. JS blocks work the same way with `<script>{{ yield_blocks("_js_") }}</script>`. Details: [[en/user/yield_blocks|yield_blocks]] and [[en/user/bem|BEM naming in templates]].

The CSS lands in an inline `<style>` tag. If your site sends a Content-Security-Policy header, it needs `style-src 'unsafe-inline'`, or the browser drops the component CSS.

### Base layer

Put the site-wide CSS (theme variables, a reset, typography) in one component that every page calls, for example `components/base.html`. Make it the page frame as well, so a page can't forget it:

```jet
{{ block _style_@lid() }}
:root { --text: #1a1a1a; --accent: #0070f3; }
*, *::before, *::after { box-sizing: border-box; }
body { margin: 0; font: 16px/1.5 system-ui, sans-serif; color: var(--text); }
a { color: var(--accent); }
{{ end }}

{{ block @lid(title="") }}
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>{{ title }}</title>
  <style>{{ yield_blocks("_style_") }}</style>
</head>
<body>
  {{ yield content }}
</body>
</html>
{{ end }}
```

A page wraps its content in the base:

```jet
{{ yield components_base(title=note.Title()) content }}
  {{ note.HTMLString() }}
  {{ yield components_button(label="All posts", url="/blog") }}
{{ end }}
```

The base holds only `:root` variables and element selectors (`body`, `a`, `h1`), never a class. Components read the variables, `.@did { color: var(--accent); }`, so changing the theme means editing one file.

### The order of styles in `<style>`

`yield_blocks("_style_")` prints style blocks in the order trip2g reaches the component files:

1. First the components that other components call: the header the base yields, the button inside a card.
2. Then the components the page calls itself, in the order of their first `yield` in the page.

A page that yields `base` (which yields `header`) and then `card` (which yields `button`) gets the CSS of `header`, `button`, `base`, `card`, in that order. So the base's CSS isn't guaranteed to come first.

Don't depend on the order. With BEM, specificity decides instead. The base uses element selectors, and each component rule uses its own class, so a component rule beats a base rule wherever the two land. Two components never style the same class. Order matters only between two rules with the same specificity that match the same element, which with BEM means one component styling another one's classes. Rule 4 below forbids that.

### Worked example

Three files: a card, a button and a page that lists the three newest public posts.

```
_layouts/
├── page.html
└── components/
    ├── button.html
    └── card.html
```

`components/card.html`:

```jet
{{ block _style_@lid() }}
.@did {
  display: block;
  padding: 16px;
  border: 1px solid #ddd;
  border-radius: 8px;
}
.@did--featured { border-color: #0070f3; }
.@did__title { margin: 0 0 8px; font-size: 1.25rem; }
{{ end }}

{{ block @lid(title="", url="", featured=false) }}
<a class="@did{{ if featured }} @did--featured{{ end }}" href="{{ url }}">
  <h3 class="@did__title">{{ title }}</h3>
  {{ yield content }}
</a>
{{ end }}
```

`components/button.html`:

```jet
{{ block _style_@lid() }}
.@did {
  display: inline-block;
  padding: 8px 16px;
  border-radius: 6px;
  background: #0070f3;
  color: #fff;
}
{{ end }}

{{ block @lid(label="", url="") }}
<a class="@did" href="{{ url }}">{{ label }}</a>
{{ end }}
```

`page.html`, with no `{{ import }}`:

```jet
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>{{ note.Title() }}</title>
  <style>{{ yield_blocks("_style_") }}</style>
</head>
<body>
  {{ note.HTMLString() }}

  {{ blog := nvs.ByGlob("blog/*.md").Public() }}
  {{ sorted := blog.SortByMeta("date").Desc().SortBy("Title") }}
  {{ range i, post := sorted.Limit(3).All() }}
    {{ yield components_card(
      title=post.Title(),
      url=post.PermalinkEncoded(),
      featured=i == 0
    ) content }}
      {{ if summary := post.M().GetString("summary", ""); summary }}
        <p>{{ summary }}</p>
      {{ end }}
    {{ end }}
  {{ end }}

  {{ yield components_button(label="All posts", url="/blog") }}
</body>
</html>
```

The rendered `<style>` holds the CSS of `.components-card` and `.components-button` only. The newest post gets `components-card--featured`, and a post with a `summary` in its frontmatter gets a paragraph under its title.

This example is rendered by a test in the trip2g repository (`internal/layoutloader/components_doc_example_test.go`), so it stays correct as the code changes.

### Best practices

Follow these when you write or change a custom layout:

1. **Build from components, don't copy markup.** When the same markup appears twice, move it into a component file and `yield` it.
2. **One component per file.** The file holds the component's HTML block and, when needed, its `_style_` and `_js_` blocks. Nothing else.
3. **Name blocks with `@lid` and classes with `@did`**, never by hand. The names then follow the file path, and two files can't collide.
4. **Use BEM for every class:** `.@did`, `.@did__element`, `.@did--modifier`. Don't style bare tags or another component's classes.
5. **Put component CSS in `_style_@lid` blocks**, and print it once with `<style>{{ yield_blocks("_style_") }}</style>` in `<head>`.
6. **Rely on auto-import in pages.** Write `{{ import }}` only in a template reached through `include`, `extends` or `exec`: auto-import doesn't reach those (see [[#Auto-import]]).
7. **Keep theme CSS in one base component** that every page calls: `:root` variables and element selectors only. Don't rely on the order of styles (see [[#Base layer]]).
8. **Give every block parameter a default and pass arguments by name.**
9. **Check optional lookups in the `if` that uses them:** `{{ if about := nvs.ByPermalink("/about"); about }}…{{ end }}`. A lookup that finds nothing is `nil`, so `if x` and `x == nil` both work.
10. **Let the template escape text.** Output is escaped by default, so titles, frontmatter strings and anything an author or visitor wrote need no filter. Methods that return HTML print as is. Use `| unsafe` only for markup the template builds itself (see [[#Escaping]]).
11. **Use `exec` for data and components for markup.** An exec'd file `return`s a list or a map that several layouts share. Wrap only the part that can legitimately fail in `try`/`catch`, and show the error to admins.
12. **List other notes with `nvs` queries and `.Public()`** on any page anonymous visitors see: `nvs.ByGlob("blog/*.md").Public()`. Without `.Public()`, paid, sign-in-only and `_` notes are listed too. Sort with a tie-breaker, because an unsorted query comes back in random order.
13. **Keep content in notes, not in templates.** Settings and lists go in frontmatter, read with `note.M()`. Text goes in the note body, read with `note.HTMLString()` or `note.PartialRenderer()`. Structured data such as a chart goes in a code block in a note. A template holds markup, not content.

### Escaping

`{{ value }}` escapes HTML, in text and inside attributes alike: `{{ post.Title() }}` prints `Q&A` as `Q&amp;A`. Methods that return HTML the server built print as is: `{{ note.HTMLString() }}`, `TitleHTML`, `ContentHTML`, `asset()`. `| html` from older templates is harmless: it escapes once. `| unsafe` prints a plain string without escaping; use it for markup the template builds itself, never for text someone else wrote. Details: [[en/user/jet-functions#Output is escaped by default|Jet functions reference]].

### See also

- [[en/user/templates|Templates]] — how layouts work
- [[en/user/jet-functions|Jet functions reference]] — every function, filter and construct
- [[en/user/yield_blocks|yield_blocks]] — CSS and JS per page, `@lid` / `@did`, `asset()`
- [[en/user/bem|BEM naming in templates]]
- [[en/user/jet-debugging|Debugging Jet templates]]
- [[en/user/renderlayout|Layout preview endpoint]] — try a template without uploading it
