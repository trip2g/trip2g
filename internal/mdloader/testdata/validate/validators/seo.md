Search snippets cut descriptions at about 160 characters, so a longer one is
a warning. The date must be ISO (YYYY-MM-DD).

```jsonnet
local re = std.native('regexMatch');
local desc = std.get(meta, 'description', '');
local date = std.toString(std.get(meta, 'date', ''));

(
  if std.length(desc) > 160
  then [{
    level: 'warning',
    msg: 'description is %d chars, keep it under 160' % std.length(desc),
  }]
  else []
) + (
  if !re('^\\d{4}-\\d{2}-\\d{2}$', date)
  then [{level: 'error', msg: 'date must be YYYY-MM-DD, got "%s"' % date}]
  else []
)
```
