---
type: note-query
free: true
---
How many tasks are in each status, for a dashboard.

```jsonnet
local tasks = [n for n in notes if std.startsWith(n.path, 'tasks/')];
local status(n) = std.get(n.meta, 'status', 'none');

{
  [s]: std.length([n for n in tasks if status(n) == s])
  for s in std.set([status(n) for n in tasks])
}
```
