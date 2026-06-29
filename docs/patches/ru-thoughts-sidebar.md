---
title: "Patch: sidebar for Russian thoughts"
type: frontmatter-patch
include: ["ru/thoughts/**/*.md"]
exclude: []
priority: 0
---

The Russian counterpart: essays under `ru/thoughts/` get the Russian thoughts sidebar.

See [[frontmatter-patches]] for how a more specific path glob layers on top of the site-wide patches.

```jsonnet
{ left_sidebar: "[[ru/thoughts/_sidebar]]" }
```
