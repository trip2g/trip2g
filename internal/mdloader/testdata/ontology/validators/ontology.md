Every note under docs/ is an object with a type. Load-bearing types need a
title, a description, a status and an updated date. Values come from fixed
vocabularies, and each verification names who confirmed the note: a person,
a process or a tool with its version.

```jsonnet
local re = std.native('regexMatch');
local nodeTypes = [
  'service', 'reference', 'runbook', 'gotcha', 'decision',
  'plan', 'guide', 'report', 'index',
];
local bearing = ['service', 'reference', 'runbook', 'plan', 'decision'];
local statuses = ['active', 'draft', 'deprecated', 'archived'];
local linkTypes = [
  'documents', 'depends_on', 'supersedes',
  'relates_to', 'implemented_by', 'part_of',
];
local err(msg) = {level: 'error', msg: msg};
local warn(msg) = {level: 'warning', msg: msg};
local type = std.get(meta, 'node_type', '');

if type == '' then [err('missing node_type')]
else if !std.member(nodeTypes, type) then [
  err('node_type "%s" is not one of: %s' % [type, std.join(', ', nodeTypes)]),
] else [
  err('a %s needs %s' % [type, f])
  for f in ['title', 'description', 'status', 'updated']
  if std.member(bearing, type) && !std.objectHas(meta, f)
] + [
  warn('status "%s" is not one of: %s' % [s, std.join(', ', statuses)])
  for s in [std.get(meta, 'status', 'active')]
  if !std.member(statuses, s)
] + [
  warn('unknown link type "%s"' % k)
  for k in std.objectFields(std.get(meta, 'links', {}))
  if !std.member(linkTypes, k)
] + [
  err('verified.by "%s" must be human:<id>, process:<id> or tool/version'
      % v.by)
  for v in std.get(meta, 'verified', [])
  if !re('^((human|process):[a-z0-9_-]+|[a-z0-9_-]+/[a-z0-9._-]+)$',
         std.get(v, 'by', ''))
]
```
