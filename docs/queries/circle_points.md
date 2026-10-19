---
title: "Query: positioning wheel points"
type: note-query
free: true
search: false
noindex: true
---

The points of a [[en/circle/_index|positioning wheel]]: the notes directly in the folder `params.dir` that have a `label`, sorted by `order`. The layout `mesh/circle` gets the rest of each note through `.Note()`.

```jsonnet
local inDir(n) =
  std.startsWith(n.path, params.dir)
  && std.length(std.split(n.path[std.length(params.dir):], '/')) == 1;
local points = [
  n for n in notes
  if inDir(n) && std.get(n.meta, 'label', '') != ''
];

[
  { version_id: n.version_id }
  for n in std.sort(points, function(n) std.get(n.meta, 'order', 0))
]
```
