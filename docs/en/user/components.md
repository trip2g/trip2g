---
title: "Components, auto-import and best practices"
free: true
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

A page calls `components/card.html` as `{{ yield components_card(...) }}`. The full placeholder reference, including `@@lid` for a literal `@lid`, is in [[en/user/yield_blocks|yield_blocks]].

**Use only lowercase letters, digits and `_` in folder and file names of components.** `@lid` keeps a `-` from the path, and a Jet block name can't contain one: `my-theme/card.html` gives `my-theme_card`, and both the component and the page that calls it fail to load with `unexpected token '-'`.

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

**When an explicit `{{ import }}` is still needed.** Auto-import runs for the page being rendered. It doesn't reach into other templates the page pulls in:

| Case | What happens | Fix |
|---|---|---|
| A template pulled in with `{{ include "partials/x" }}` yields a component | Render stops with `unresolved block "components_card"` | Add `{{ import "components/card" }}` at the top of the included template, or turn the partial into a component and yield it |
| A base template used through `{{ extends "base" }}` yields a component | Render stops with `unresolved block …` | Add `{{ import "components/card" }}` at the top of the base template |
| A page that starts with `{{ extends }}` yields a component | The page fails to load: `'extends' statements must be at the beginning of the template` | Don't combine `extends` with components. Make the shared frame a component with a content block, as in the [[#Worked example]] |

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
.@did { display: block; padding: 16px; border: 1px solid #ddd; border-radius: 8px; }
.@did--featured { border-color: #0070f3; }
.@did__title { margin: 0 0 8px; font-size: 1.25rem; }
{{ end }}

{{ block @lid(title="", url="", featured=false) }}
<a class="@did{{ if featured }} @did--featured{{ end }}" href="{{ url }}">
  <h3 class="@did__title">{{ title | html }}</h3>
  {{ yield content }}
</a>
{{ end }}
```

`components/button.html`:

```jet
{{ block _style_@lid() }}
.@did { display: inline-block; padding: 8px 16px; border-radius: 6px; background: #0070f3; color: #fff; }
{{ end }}

{{ block @lid(label="", url="") }}
<a class="@did" href="{{ url }}">{{ label | html }}</a>
{{ end }}
```

`page.html`, with no `{{ import }}`:

```jet
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>{{ note.Title() | html }}</title>
  <style>{{ yield_blocks("_style_") }}</style>
</head>
<body>
  {{ note.HTMLString() | unsafe }}

  {{ posts := nvs.ByGlob("blog/*.md").Public().SortByMeta("date").Desc().SortBy("Title").Limit(3).All() }}
  {{ range i, post := posts }}
    {{ yield components_card(title=post.Title(), url=post.PermalinkEncoded(), featured=i == 0) content }}
      {{ if summary := post.M().GetString("summary", ""); summary }}<p>{{ summary | html }}</p>{{ end }}
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
6. **Rely on auto-import in pages.** Write `{{ import }}` only in an included or base template (see [[#Auto-import]]).
7. **Don't use `extends`.** Make the shared page frame a component that wraps the page body with `{{ yield content }}`.
8. **Give every block parameter a default and pass arguments by name.**
9. **Check optional lookups in the `if` that uses them:** `{{ if about := nvs.ByPermalink("/about"); about }}…{{ end }}`. Compare with `if x`, not `x == nil`: a missing note is a typed nil.
10. **Escape text, don't escape HTML.** Text values are escaped: titles, frontmatter strings, anything an author or visitor wrote. Methods that return HTML are output as is: `note.HTMLString()`, `TitleHTML`, `ContentHTML`. How to do that today: [[#Escaping]].
11. **Never use `exec`, `return` or `try`/`catch`.** A layout with `return` or `try` fails to load, so `exec` can't return a value either. Reuse template code with components.
12. **List other notes with `nvs` queries and `.Public()`** on any page anonymous visitors see: `nvs.ByGlob("blog/*.md").Public()`. Without `.Public()`, paid, sign-in-only and `_` notes are listed too. Sort with a tie-breaker, because an unsorted query comes back in random order.
13. **Keep content in notes, not in templates.** Settings and lists go in frontmatter, read with `note.M()`. Text goes in the note body, read with `note.HTMLString()` or `note.PartialRenderer()`. Structured data such as a chart goes in a code block in a note. A template holds markup, not content.

### Escaping

> **This paragraph describes the current behaviour and will change when escaping becomes the default.** Today trip2g doesn't escape template output: `{{ value }}` prints the value as is. Write `| html` after every text value: `{{ post.Title() | html }}`, and the same inside attributes. Print HTML-returning methods without a filter, or with `| unsafe` to show the intent: `{{ note.HTMLString() | unsafe }}`. Details: [[en/user/jet-functions#Output is not escaped by default|Jet functions reference]].

### See also

- [[en/user/templates|Templates]] — how layouts work
- [[en/user/jet-functions|Jet functions reference]] — every function, filter and construct
- [[en/user/yield_blocks|yield_blocks]] — CSS and JS per page, `@lid` / `@did`, `asset()`
- [[en/user/bem|BEM naming in templates]]
- [[en/user/jet-debugging|Debugging Jet templates]]
- [[en/user/renderlayout|Layout preview endpoint]] — try a template without uploading it
