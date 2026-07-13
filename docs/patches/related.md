---
title: "Patch: backlinks + related under articles"
type: frontmatter-patch
include:
  - en/thoughts/**/*.md
  - ru/thoughts/**/*.md
  - en/user/**/*.md
  - ru/user/**/*.md
exclude:
  - "**/_*.md"
priority: 10
---

Every article under `thoughts/` and `user/` should show two blocks under its body: **backlinks** (notes that link here) and **related** (semantically similar notes). Rather than add a `content:` list to each note, this patch sets it once.

It only applies when the note does not already declare its own `content` or `widgets`. That guard leaves the `_index` magazine pages (`content: [self, magazine]`) and the few user-doc pages with custom layouts untouched. Hidden `_`-prefixed notes (sidebars, index, headers) are excluded by the glob above, so sidebars and index listings are never rewritten.

The keywords match the sidebar widget vocabulary: `self` renders the note body, `backlinks` (alias of `inlinks`) the inbound-link list, `similar` the vector-similarity "related" list. Note that `backlinks` only shows something when other notes actually link here — a fresh essay with no inbound links renders an empty backlinks block; `similar` is content-based and works as soon as vector search is enabled.

See [[frontmatter-patches]] for the patch model and [[subgraphs]] for how a folder-scoped glob layers on top of the site-wide patches.

```jsonnet
if std.objectHas(meta, "content") || std.objectHas(meta, "widgets")
then {}
else { content: ["self", "backlinks", "similar"] }
```
