---
title: "One HTML page"
free: true
home_position: 4
lang_redirect: "[[ru/user/One HTML Page]]"
---

Sometimes a page needs to be pure HTML — a product landing, a custom homepage, or a demo. In trip2g, this is done through layouts: all HTML lives in a template file, and the note just points to it.

> **Note:** raw HTML in a note body **does not work**. goldmark (the markdown engine) strips block-level HTML and replaces it with `<!-- raw HTML omitted -->`. Put HTML only in the layout file.

### Pattern 1 — standalone page

All HTML in the layout, note body empty. Use this for a custom homepage, landing, or demo.

**Step 1.** Create `_layouts/my-page.html` with all the HTML:

```jet
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{ note.Title() }}</title>
  <style>/* all CSS here */</style>
</head>
<body>
  <section class="hero">
    <h1>Hello, world</h1>
    <p>A fully custom HTML page.</p>
    <a href="/signup" class="btn">Get started</a>
  </section>
</body>
</html>
```

**Step 2.** Create `my-page.md` — frontmatter only, body empty:

```yaml
---
title: My page
layout: my-page
free: true
---
```

After sync, the note is published as a page rendered entirely by the layout.

### Pattern 2 — one layout, many notes

One layout, many notes — each with different content. Variability comes from frontmatter and the markdown body, **not from raw HTML in the note body**.

Data from frontmatter via `note.M()`:

```jet
{{ note.M().GetString("hero_title", "Title") }}
{{ note.M().GetString("cta_text", "Get started") }}
{{ note.M().GetBool("show_video", false) }}
```

Full markdown content of the note:

```jet
{{ note.HTMLString() | unsafe }}
```

Markdown broken into sections via `note.PartialRenderer()`:

```jet
{{ range i, section := note.PartialRenderer().Sections(2) }}
  <section>
    <h2>{{ section.TitleHTML | unsafe }}</h2>
    {{ section.ContentHTML | unsafe }}
  </section>
{{ end }}
```

Full API reference in [[en/user/templates|the Templates guide]].

### Pattern 3 — multi-file layout

For sites with a shared header, footer, and multiple page types: extract repeated parts into `blocks.html`.

```
_layouts/
└── my-theme/
    ├── blocks.html   — header, footer, base wrapper
    ├── page.html     — generic page template
    └── landing.html  — landing page template
```

`blocks.html`:

```jet
{{ block main_layout() }}
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>{{ note.Title() }}</title>
  <link rel="stylesheet" href="{{ asset("style.css") }}">
</head>
<body>
  {{ yield header() }}
  <main>{{ yield content }}</main>
  {{ yield footer() }}
</body>
</html>
{{ end }}

{{ block header() }}
<header><a href="/">Home</a></header>
{{ end }}

{{ block footer() }}
<footer><p>2025 My site</p></footer>
{{ end }}
```

`page.html`:

```jet
{{ import "blocks" }}

{{ yield main_layout() content }}
  <article>
    <h1>{{ note.Title() }}</h1>
    {{ note.HTMLString() | unsafe }}
  </article>
{{ end }}
```

See [[en/user/templates#Organizing multiple templates|Organizing multiple templates]] for more.

### Debug: preview layout before sync

`/_system/renderlayout` renders a layout with warnings **without writing to the vault**:

```
GET /_system/renderlayout?note=<slug>&layout=<layout-name>
```

Use this to catch errors before your first sync. See [[en/user/renderlayout|renderlayout]] for details.

### Notes

The note filename becomes the URL. To make the page the site root, add `slug: /` to the frontmatter.

CSS and JS go in `_assets/`, referenced via `asset()`:

```jet
<link rel="stylesheet" href="{{ asset("landing.css") }}">
```

---

See also: [[en/user/templates|Templates]] · [[en/user/templates#Jet template syntax|Jet syntax]] · [[en/user/default-template|Default template]] · [[en/user/renderlayout|renderlayout]]
