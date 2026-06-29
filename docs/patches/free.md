---
title: "Patch: publish every note"
type: frontmatter-patch
include: ["**/*.md"]
exclude: ["demo/**/*.md"]
priority: 0
---

This whole documentation site is public. Rather than add `free: true` to the frontmatter of every note by hand, one patch marks all Markdown free — except the `demo/` vault, which keeps its own access rules.

This note IS the patch: the rule below is read from its `jsonnet` block, and the file is itself a live example of the feature. See [[frontmatter-patches]] for how patches work.

```jsonnet
{ free: true }
```
