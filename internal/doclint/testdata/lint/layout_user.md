---
title: Layout User Fixture
layout: bad_layout
---

This note uses bad_layout. The smoke renderer will try to render it and
fail because bad_layout.html accesses note.NoSuchField, producing a
"smoke render error" warning on the layout.
