---
title: Hello from Demo Vault
---

# Hello from Demo Vault

This is a **demo note** used for testing the `/_system/renderlayout` preview endpoint.

## Usage

Send a POST request referencing this file as `note.content`:

```bash
curl -s -X POST \
  -H "X-API-Key: $TRIP2G_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "layout": {"content": "'"$(cat docs/demo/_layouts/article.html | jq -Rs .)"'"},
    "note":   {"content": "# Hello\n\nRendered from markdown."}
  }' \
  http://localhost:8080/_system/renderlayout
```

## Sections

### Features

- Renders Jet templates instantly
- Returns `preview_url` you can open in a browser
- Reports Jet compile/runtime errors as `warnings`
- Supports `extra_layouts` for component overrides

### Frontmatter

The `title` field from frontmatter is used as `note.Title()` in the layout.
