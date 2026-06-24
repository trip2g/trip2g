---
type: frontmatter-patch
include: [en/user/**, ru/user/**]
exclude: ["**/_*.md"]
priority: 10
---

```jsonnet
if std.objectHas(meta, "content") then {} else {
  content: ["self", "similar", "backlinks"],
  right_sidebar: ["toc"],
}
```
