Notes are named "{project} {type} title – YYYY-MM-DD – tool.md": project
and type in braces, the date always near the end, the tool that wrote the
note last. A frontmatter date, when present, must match the one in the name.
A tool missing from the list is a warning, so a new one doesn't fail syncs.

```jsonnet
local re = std.native('regexMatch');
local parts = std.split(path, '/');
local file = parts[std.length(parts) - 1];
local name = std.substr(file, 0, std.length(file) - 3);
local chunks = std.split(name, ' – ');
local tools = [
  'Claude Code', 'Cursor', 'ChatGPT', 'Gemini', 'Windsurf', 'Manual',
];
local err(msg) = {level: 'error', msg: msg};
local warn(msg) = {level: 'warning', msg: msg};

local checkTool(tool) =
  if std.member(tools, tool) then []
  else if re('^[A-Z][A-Za-z0-9 .]*$', tool) then [
    warn('new tool "%s"? add it to the list in this validator' % tool),
  ] else [
    err('tool "%s" must be a name like %s' % [tool, std.join(', ', tools)]),
  ];

if std.length(chunks) != 3 then [
  err('name must be "{project} {type} title – YYYY-MM-DD – tool"'),
] else
  local head = chunks[0], date = chunks[1], tool = chunks[2];
  (
    if re('^\\{[a-z0-9]+\\} \\{[a-z]+\\} .+$', head) then []
    else [err('name must start with {project} {type} and a title')]
  ) + (
    if re('^\\d{4}-\\d{2}-\\d{2}$', date) then []
    else [err('date "%s" must be YYYY-MM-DD' % date)]
  ) + checkTool(tool) + (
    local fmDate = std.toString(std.get(meta, 'date', date));
    if fmDate == date then []
    else [err('frontmatter date %s differs from %s in the name'
              % [fmDate, date])]
  )
```
