A board app reads its settings from the frontmatter. Keep their types right.

```jsonnet
local isInt(v) = std.isNumber(v) && std.floor(v) == v;
local columns = std.get(meta, 'columns');

(
  if !std.isArray(columns) || std.length(columns) == 0
  then [{level: 'error', msg: 'columns must be a non-empty list'}]
  else []
) + (
  if std.objectHas(meta, 'wip_limit') && !isInt(meta.wip_limit)
  then [{level: 'error', msg: 'wip_limit must be a whole number'}]
  else []
) + (
  if std.objectHas(meta, 'show_archive')
    && !std.isBoolean(meta.show_archive)
  then [{level: 'warning', msg: 'show_archive must be true or false'}]
  else []
)
```
