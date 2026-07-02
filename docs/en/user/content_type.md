---
title: "Custom Content-Type notes"
description: "Serve a note as plain text, JSON, CSV, or any MIME type by setting content_type: in its frontmatter. The note body is delivered as-is, without an HTML wrapper."
free: true
lang_redirect: "[[ru/user/content_type]]"
---

Set `content_type:` in a note's frontmatter to serve it as any MIME type instead of the usual HTML page. The note body is delivered as-is, with frontmatter stripped. No HTML wrapper, no template chrome.

### Basic example

```yaml
---
slug: /data.csv
content_type: text/csv; charset=utf-8
free: true
---
name,value
alpha,1
beta,2
```

A request to `/data.csv` returns `Content-Type: text/csv` and the CSV rows above.

### What it does

When `content_type` is set, the render path skips the HTML wrapper and serves the note body directly:

1. Sets the HTTP `Content-Type` header to whatever you put in `content_type`.
2. Strips the frontmatter block (the `---` delimited section).
3. Returns the remaining text verbatim.
4. Skips the page cache (the cache stores only `text/html` responses).

The note is still subject to normal access rules: `free: true` makes it public, paid notes require a token.

### Supported values

Any valid MIME type works:

| Value | Use case |
|-------|----------|
| `text/plain; charset=utf-8` | robots.txt, llms.txt, any plain text file |
| `text/csv; charset=utf-8` | CSV data exports |
| `application/json` | Simple JSON endpoints |
| `application/xml` | XML feeds or sitemaps |

### Combining with a custom template

If the note has both `content_type` and a `layout`, the Jet layout renders the body and the frontmatter `content_type` sets the header. This is useful when you need template logic to build the output (for example, listing note titles in a feed format).

For static content with no template logic, just leave `layout` out and write the body directly in the note.

### Notes

- The `search: false` frontmatter key prevents the note from appearing in site search results, which makes sense for machine-readable files.
- Markdown is not rendered to HTML. The body is served as raw text, so headings, lists, and links stay as markdown characters.
- See [[en/user/robots_and_llm|robots.txt and llms.txt]] for a practical example of two common plain-text files.
