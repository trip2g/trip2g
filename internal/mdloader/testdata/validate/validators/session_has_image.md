Every "Session …" section of a workshop shows at least one picture. The fold
splits the note into sections at each `##` and counts images in each.

```jsonnet
local last(xs) = xs[std.length(xs) - 1];
local init(xs) = xs[:std.length(xs) - 1];

local sections = std.foldl(
  function(acc, n)
    if n.type == 'heading' && n.level == 2 then
      acc + [{title: n.text, line: n.line, images: 0}]
    else if n.type == 'image' && std.length(acc) > 0 then
      init(acc) + [last(acc) + {images: super.images + 1}]
    else acc,
  nodes,
  [],
);

[
  {level: 'error', line: s.line, msg: '"%s" has no image' % s.title}
  for s in sections
  if std.startsWith(s.title, 'Session') && s.images == 0
]
```
