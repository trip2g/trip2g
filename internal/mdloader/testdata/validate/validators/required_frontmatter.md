Every article needs a title and a description, both strings. Tags, when
present, must be a list.

```jsonnet
local required = ['title', 'description'];

[
  {level: 'error', msg: 'missing frontmatter field: ' + f}
  for f in required
  if !std.objectHas(meta, f)
] + [
  {level: 'error', msg: f + ' must be a string'}
  for f in required
  if std.objectHas(meta, f) && !std.isString(meta[f])
] + (
  if std.objectHas(meta, 'tags') && !std.isArray(meta.tags)
  then [{level: 'error', msg: 'tags must be a list'}]
  else []
)
```
