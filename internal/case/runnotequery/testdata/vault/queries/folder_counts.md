---
type: note-query
free: true
---
How many notes the visitor can read in each top-level folder.

```jsonnet
local folder(n) =
  local parts = std.split(n.path, '/');
  if std.length(parts) > 1 then parts[0] else '(root)';

{
  [f]: std.length([n for n in notes if folder(n) == f])
  for f in std.set([folder(n) for n in notes])
}
```
