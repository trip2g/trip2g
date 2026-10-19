---
type: note-query
free: true
---
The latest blog posts. Params: `{"limit": 2}`, 3 by default.

```jsonnet
local limit = std.get(params, 'limit', 3);
local posts = [n for n in notes if std.startsWith(n.path, 'blog/')];
local byDate = std.sort(posts, function(n) std.get(n.meta, 'date', ''));

[n.title for n in std.reverse(byDate)][:limit]
```
