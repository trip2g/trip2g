---
title: "Patch: sidebar for English thoughts"
type: frontmatter-patch
include: ["en/thoughts/**/*.md"]
exclude: []
priority: 0
---

Essays under `en/thoughts/` share their own sidebar, separate from the user docs.

See [[frontmatter-patches]] for how a more specific path glob layers on top of the site-wide patches.

```jsonnet
{ left_sidebar: "[[en/thoughts/_sidebar]]" }
```
