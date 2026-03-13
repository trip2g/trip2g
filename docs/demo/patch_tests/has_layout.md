---
# Expected patches: "Conditional default layout" (priority 0)
# Expected meta: { layout: "custom", free: true } - layout NOT overridden
# Tests: conditional should return {} (no-op) when layout already exists
layout: custom
free: true
title: Has Layout Test
---

This page already has `layout: custom` in frontmatter.

The conditional patch should detect this and return `{}` (no-op).

Final layout should remain "custom", not changed to "default".
