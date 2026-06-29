---
title: "Patch: sidebar for English user docs"
type: frontmatter-patch
include: ["en/user/**/*.md"]
exclude: []
priority: 0
---

Every page under `en/user/` gets the shared user-docs sidebar, without repeating `left_sidebar:` in each note's frontmatter.

See [[frontmatter-patches]] for matching by path glob.

```jsonnet
{ left_sidebar: "[[en/user/_sidebar]]" }
```
