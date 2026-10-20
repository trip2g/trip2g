---
type: frontmatter-patch
include: ["docs/**"]
---

A note's folder gives its default node_type, and every note under docs/ is
checked by the ontology validator.

```jsonnet
local byFolder = {
  decisions: 'decision', ops: 'runbook', plans: 'plan', reference: 'reference',
};
local parts = std.split(path, '/');
local file = parts[std.length(parts) - 1];
local folder = if std.length(parts) > 2 then parts[1] else '';
local own = std.get(meta, 'validate', []);

{
  validate: (if std.isString(own) then [own] else own) + ['[[ontology]]'],
} + (
  if std.objectHas(meta, 'node_type') then {}
  else if file == 'README.md' then { node_type: 'index' }
  else if std.objectHas(byFolder, folder) then { node_type: byFolder[folder] }
  else {}
)
```
