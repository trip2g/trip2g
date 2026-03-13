---
# Expected patches: "Conditional default layout" (priority 0)
# Expected meta after patches: { layout: "default", free: true }
# Tests: if std.objectHas(meta, "layout") then {} else { layout: "default" }
free: true
title: Conditional Patch Test
---

This page tests conditional jsonnet logic.

The patch checks `if std.objectHas(meta, "layout")` and only sets layout if missing.

Since this note has no layout in frontmatter, patch should add `layout: "default"`.
