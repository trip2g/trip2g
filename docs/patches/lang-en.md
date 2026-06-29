---
title: "Patch: default language English"
type: frontmatter-patch
include: ["**/*.md"]
exclude: ["demo/**/*.md"]
priority: -1
---

Sets `lang: en` on every note that does not set its own. The negative `priority` makes this the weakest patch, so the Russian section below can override it for `ru/` notes.

See [[frontmatter-patches]] for how `priority` resolves overlapping patches.

```jsonnet
{ lang: "en" }
```
