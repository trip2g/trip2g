---
title: "One HTML page"
free: true
home_position: 4
lang_redirect: "[[ru/user/One HTML Page]]"
---

Sometimes a page needs to be pure HTML — a product landing, a custom homepage, or a demo. trip2g supports this without any special mode: write the HTML in the note body, point the note at a bare template, and the template renders it as-is.

### How it works

**Step 1.** Create a minimal template in `_layouts/html-page.html`:

```jet
<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <title>{{ note.Title() }}</title>
</head>
<body>
  {{ note.HTMLString() | unsafe }}
</body>
</html>
```

The `| unsafe` filter tells the template engine to output the content without escaping it. Without it, angle brackets are escaped and the HTML renders as text.

**Step 2.** Create your note (e.g. `my-page.md`) and assign the template:

```yaml
---
title: My landing page
layout: html-page
free: true
---
```

**Step 3.** Write raw HTML in the note body:

```html
<section class="hero">
  <h1>Hello, world</h1>
  <p>This is a fully custom HTML page.</p>
  <a href="/signup" class="btn">Get started</a>
</section>
```

After sync, `my-page.md` is published as a page whose body is exactly your HTML.

### Good use cases

- Custom site homepage (`slug: /`)
- Product landing page
- Demo or portfolio page
- Any page where markdown layout is too limiting

### Notes

The note filename becomes the URL unless you override it with a `slug` property. To make the page the site root, add `slug: /` to the frontmatter.

CSS and JS files go in `_assets/` and are referenced from the template with `asset()`:

```jet
<link rel="stylesheet" href="{{ asset("landing.css") }}">
```

For more on templates — variables, asset loading, and multi-file layouts — see [[en/user/templates|Templates]].
