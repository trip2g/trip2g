---
title: "Patch: Russian section"
type: frontmatter-patch
include: ["ru/**/*.md"]
exclude: []
priority: 0
---

Everything under `ru/` is Russian and shares the Russian header and footer. One patch can carry several `jsonnet` blocks — here the first sets the language, the second wires the shared header/footer notes.

See [[frontmatter-patches]] for the patch model.

```jsonnet
{ lang: "ru" }
```

```jsonnet
{
  header: "[[ru/_header]]",
  footer: "[[ru/_footer]]",
}
```
