---
type: note-query
---

A health report over docs/: how many notes are typed, which lack a
description, which folders have no README, and which superseded notes are
still active.

```jsonnet
local docs = [n for n in notes if std.startsWith(n.path, 'docs/')];
local parts(p) = std.split(p, '/');
local dir(p) = std.join('/', parts(p)[:std.length(parts(p)) - 1]);
local name(p) = std.strReplace(parts(p)[std.length(parts(p)) - 1], '.md', '');
local target(link) = std.split(std.stripChars(link, '[]'), '|')[0];
local byName = { [name(n.path)]: n for n in docs };
local paths = [n.path for n in docs];
local done = ['deprecated', 'archived'];

{
  docs: std.length(docs),
  typed: std.length([n for n in docs if std.objectHas(n.meta, 'node_type')]),
  no_description: [
    n.path for n in docs
    if std.objectHas(n.meta, 'node_type')
    && n.meta.node_type != 'index'
    && !std.objectHas(n.meta, 'description')
  ],
  no_readme: [
    d for d in std.set([dir(p) for p in paths])
    if !std.member(paths, d + '/README.md')
  ],
  superseded_but_active: [
    { note: byName[t].path, replaced_by: n.path }
    for n in docs
    for t in [
      target(l)
      for l in std.get(std.get(n.meta, 'links', {}), 'supersedes', [])
    ]
    if std.objectHas(byName, t)
    && !std.member(done, std.get(byName[t].meta, 'status', ''))
  ],
}
```
