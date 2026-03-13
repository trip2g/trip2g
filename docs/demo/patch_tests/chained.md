---
# Expected patches: "Add default layout" (priority 0), "Override blog layout" (priority 10)
# Expected meta after patches: { layout: "blog_layout", free: true }
# Original frontmatter has no layout field
free: false
title: Chained Patch Test
---

This page tests patch chaining with different priorities.

Patch 1 (priority 0): Sets default layout
Patch 2 (priority 10): Overrides with blog_layout
Patch 3 (priority 20): Sets free: true

Final result should have layout="blog_layout" and free=true.
