---
type: note-query
free: true
---
Open tasks whose `due` date has passed, for a task board.

```jsonnet
local overdue(n) =
  std.objectHas(n.meta, 'due') && n.meta.due < now.date;

[
  n.title
  for n in notes
  if std.startsWith(n.path, 'tasks/')
  && std.get(n.meta, 'status') != 'done'
  && overdue(n)
]
```
