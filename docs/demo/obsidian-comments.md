---
free: true
title: Obsidian Comments Test
---

This page tests Obsidian-style `%%` comment stripping.

## Inline comment

The word BEFORE%%this comment must not appear%%AFTER is visible on both sides.

## Block comment

Text before the block comment.

%%
This entire block comment must not appear in output.
Multiple lines are hidden.
%%

Text after the block comment.

## Code block preservation

Content inside fenced code blocks must not be stripped:

```
%% this literal percent-percent inside a code block must be preserved %%
```
