External links may point only at domains we trust.

```jsonnet
local allowed = ['https://trip2g.com', 'https://github.com/trip2g'];
local external(url) =
  std.startsWith(url, 'http://') || std.startsWith(url, 'https://');
local trusted(url) =
  std.length([a for a in allowed if std.startsWith(url, a)]) > 0;

[
  {level: 'error', line: n.line, msg: 'link to untrusted domain: ' + n.url}
  for n in nodes
  if n.type == 'link' && external(n.url) && !trusted(n.url)
]
```
