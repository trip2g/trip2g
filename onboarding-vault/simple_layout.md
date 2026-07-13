---
free: true
layout: demo/simple
title: Custom layout example
---

This page is rendered through a custom layout: `_layouts/demo/simple.html`, which imports a reusable Jet block from `_layouts/demo/_blocks.html`.

Copy this pattern — a `_blocks.html` defining `{{ block name(args) }}...{{ end }}`, a layout that `{{ import "_blocks" }}`s and `{{ yield }}`s it, and a note that points to the layout via `layout:` frontmatter — to build your own page designs.
