---
type: note-query
free: true
---
Blog posts with a tag, newest first. Params: `{"tag": "go"}`.

```jsonnet
local tagged = [
  n for n in notes
  if std.startsWith(n.path, 'blog/')
  && std.member(std.get(n.meta, 'tags', []), params.tag)
];
local byDate = std.sort(tagged, function(n) std.get(n.meta, 'date', ''));

[
  { title: n.title, url: n.permalink, date: n.meta.date }
  for n in std.reverse(byDate)
]
```
