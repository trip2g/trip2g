---
title: "Patch: keep dev docs out of search"
type: frontmatter-patch
include: ["dev/**/*.md"]
exclude: []
priority: 0
---

The `dev/` notes are internal and partly out of date. This patch sets `search: false` on all of them so they never surface in the site search, without editing each file.

See [[frontmatter-patches]] for the full list of fields a patch can set.

```jsonnet
{ search: false }
```
