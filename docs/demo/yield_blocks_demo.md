---
free: true
layout: yield-blocks-demo
title: yield_blocks Demo
---

This page demonstrates `yield_blocks` — automatic per-page CSS/JS collection for BEM components.

The layout uses three components: `yb_hero`, `yb_card`, and `yb_button`.
Each component defines its CSS in a `_style_*` block.
`yield_blocks("_style_yb_")` emits only the CSS for components actually used on this page.
