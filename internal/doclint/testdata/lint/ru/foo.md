---
title: Foo (Russian)
---

This Russian note uses a bare wikilink [[bar]]. Both en/bar.md and ru/bar.md
exist at the same path depth, so the link is ambiguous (two candidates).
The resolver picks one non-deterministically; the linter flags the ambiguity.
