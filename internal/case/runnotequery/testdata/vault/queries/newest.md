---
type: note-query
free: true
---
The three newest blog posts, as note ids: the page gets each post through `rows { note }` or `.Note()`.

```jsonnet
local posts = [n for n in notes if std.startsWith(n.path, 'blog/')];
local byDate = std.sort(posts, function(n) std.get(n.meta, 'date', ''));

[{ version_id: n.version_id } for n in std.reverse(byDate)][:3]
```
