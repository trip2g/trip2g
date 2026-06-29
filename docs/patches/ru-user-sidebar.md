---
title: "Patch: sidebar for Russian user docs"
type: frontmatter-patch
include: ["ru/user/**/*.md"]
exclude: []
priority: 0
---

The Russian counterpart of the user-docs sidebar patch: every page under `ru/user/` gets the shared Russian sidebar.

See [[frontmatter-patches]] for matching by path glob.

```jsonnet
{ left_sidebar: "ru/user/_sidebar.md" }
```
