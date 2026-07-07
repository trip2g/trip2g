---
free: true
title: "Include Layout: Pull Another Note In"
layout: showcase/include
---

This page's layout pulls **another note** into the page. The highlighted block below this text is not part of this note — it lives in `_snippet-cta.md` and is resolved at render time with `nvs.ByWikilink("_snippet-cta")`.

Edit the snippet note once, and every page that pulls it updates. That's how the landing page on trip2g.com is built: the layout provides structure and style, the copy lives in small hidden notes.
