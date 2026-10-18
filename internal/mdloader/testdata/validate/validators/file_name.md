Notes are named "{project} {type} title – YYYY-MM-DD – tool.md": project
and type in braces, the date always near the end, the tool that wrote the
note last. A frontmatter date, when present, must match the one in the name.

```jsonnet
local re = std.native('regexMatch');
local parts = std.split(path, '/');
local file = parts[std.length(parts) - 1];
local name = std.substr(file, 0, std.length(file) - 3);
local chunks = std.split(name, ' – ');
local tools = ['Claude Code', 'Cursor', 'ChatGPT', 'Manual'];

local problems =
  if std.length(chunks) != 3 then [
    'name must be "{project} {type} title – YYYY-MM-DD – tool"',
  ] else
    local head = chunks[0], date = chunks[1], tool = chunks[2];
    (
      if re('^\\{[a-z0-9]+\\} \\{[a-z]+\\} .+$', head) then []
      else ['name must start with {project} {type} and a title']
    ) + (
      if re('^\\d{4}-\\d{2}-\\d{2}$', date) then []
      else ['date "%s" must be YYYY-MM-DD' % date]
    ) + (
      if std.member(tools, tool) then []
      else ['tool "%s" must be one of: %s' % [tool, std.join(', ', tools)]]
    ) + (
      local fmDate = std.toString(std.get(meta, 'date', date));
      if fmDate == date then []
      else ['frontmatter date %s differs from %s in the name' % [fmDate, date]]
    );

[{level: 'error', msg: p} for p in problems]
```
