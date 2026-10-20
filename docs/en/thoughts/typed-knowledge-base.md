---
title: A typed knowledge base, the ontoship way
free: true
lang_redirect: "[[ru/thoughts/typed-knowledge-base]]"
---

[ontoship](https://github.com/vakovalskii/ontoship) by Valerii Kovalskii is a knowledge base made of three things: markdown files, a `README.md` index in every folder, and git. Its CLI, `gitmark`, builds a search index, an HTML graph and reports from the same files. On top of that sits an optional ontology, and it is the part worth borrowing: every document gets a type, and a linter checks that the base keeps its shape as it grows. This essay shows how much of it a trip2g vault does today, and where it stops.

## The idea

The ontology comes from Palantir's: an organization is modelled as objects, their properties and the links between them. ontoship applies the same three entities to documentation about code:

- **The object** is the document. Its `node_type` comes from a fixed list: `service`, `reference`, `runbook`, `gotcha`, `decision`, `plan`, `guide`, `report`, `index`.
- **The properties** are the frontmatter: `title`, `description`, `service`, `status`, `updated`, plus the trust fields of Google's Open Knowledge Format: `generated` (who wrote it), `verified` (who checked it against reality) and `stale_after`.
- **The links** are typed, under `links:`: `documents`, `depends_on`, `supersedes`, `relates_to`, `implemented_by`, `part_of`.

`gitmark lint` checks eight invariants:

| | Invariant |
|---|---|
| I1 | every load-bearing document has a valid `node_type` |
| I2 | `node_type`, `service` and `status` come from their vocabularies |
| I3 | no orphans: a document has at least one link, in or out |
| I4 | no broken links |
| I5 | every folder under `docs/` has a `README.md` |
| I6 | the target of `supersedes` is `deprecated` or `archived` |
| I7 | a `documents` or `implemented_by` target exists in the code |
| I8 | `stale_after` hasn't passed |

The point is the same as with any schema. A typed document tells a person and an agent what it is before they read it, and a base that is checked doesn't decay into a pile of files.

## The same on trip2g

A trip2g vault is a folder of markdown too, so the pieces map almost one to one. The examples below come from a small demo vault that the trip2g tests load: six notes under `docs/`, one patch, one validator and one query.

### The type from the folder

ontoship puts each type in its own folder: decisions in `docs/decisions/`, runbooks in `docs/ops/`. Then the folder can set the type, and nobody has to write it by hand. A [[en/user/frontmatter-patches|frontmatter patch]] does it for every note under `docs/`:

````
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
````

A `node_type` written in the note wins. A `README.md` becomes an `index`. The same patch attaches the validator below to every note under `docs/`, added to whatever validators the note already lists.

### I1 and I2: a validator

The rules themselves go into a [[en/user/validators|validator]], `validators/ontology.md`:

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

It checks that the note has a type from the list, that a load-bearing type has a title, a description, a status and an updated date, that `status` and the link types come from their vocabularies, and that each `verified` entry names a person, a process or a tool with its version. A wrong status or an unknown link type is a warning; a missing type or field is an error.

In the demo, a runbook with no description, `status: live`, a `see_also` link and `verified: [{ by: claude }]` gets four messages on sync:

```
[CRITICAL] validate validators/ontology.md: a runbook needs description
[WARNING] validate validators/ontology.md: status "live" is not one of: active, draft, deprecated, archived
[WARNING] validate validators/ontology.md: unknown link type "see_also"
[CRITICAL] validate validators/ontology.md: verified.by "claude" must be human:<id>, process:<id> or tool/version
```

ontoship keeps the vocabularies in the frontmatter of its `ontology.md`. Here they live in the validator, because a validator sees only the note it checks and can't read another one. The `service` vocabulary is specific to each project, so the demo leaves it out; it would be one more list of the same kind.

### I4: already there

trip2g resolves every wikilink in a note's text and reports one that leads nowhere as `broken link`, in the admin panel and in `trip2g lint`. No rule is needed.

### I5, I6 and the health score: a query

`gitmark health` is a scorecard of the whole base: the share of typed documents, the documents without a description, and so on. In trip2g that is a [[en/user/note-queries|note query]]: a note with a Jsonnet program that gets the path and frontmatter, after patches, of every note the caller can read, and returns any JSON. The demo runs it as the admin, who can read them all. `queries/kb_health.md`:

````
---
type: note-query
---

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
````

On the demo vault it returns:

```json
{
  "docs": 6,
  "typed": 5,
  "no_description": ["docs/ops/restart-db.md"],
  "no_readme": ["docs/decisions", "docs/ops", "docs/plans"],
  "superseded_but_active": [
    {
      "note": "docs/decisions/old-sandbox.md",
      "replaced_by": "docs/decisions/drop-firecracker.md"
    }
  ]
}
```

`no_readme` is I5, and `superseded_but_active` is I6: the old sandbox design was replaced, but nobody marked it deprecated. A page shows the report through `noteQuery("[[kb_health]]")` in a template, or a script asks for it through the `noteQuery` GraphQL query. The same mechanism answers `gitmark query`-style questions, such as "all decisions that aren't deprecated": that is a filter over `notes` and their `meta`.

I3 needs links, and a query doesn't get them. A [[en/user/templates|template]] does: `nvs.BackLinks(note)` and `nvs.OutLinks(note)` list a note's incoming and outgoing links, so a report page can list the notes where both are empty.

## Where it stops

**I3, I5 and I6 are a report, not a mark.** The validator marks the note it checks, and the sync CLI exits with code 1 on an error, so an agent sees the problem right away. A query or a template answers only when someone opens the page. To mark notes by rules over the whole vault, trip2g needs a validator that sees all notes; it is planned, and the discussion is in [issue #436](https://github.com/trip2g/trip2g/issues/436).

**I8 isn't possible yet.** Neither a validator nor a query knows the current date. For validators this is deliberate: their results are cached, and a result that depends on the clock can't be. Time is planned for the vault-wide validators of the same issue.

**I7 doesn't apply.** A trip2g vault holds notes, not the code they describe, so there is nothing to check a `documents` path against.

## What it gives

The rules live in the vault, next to the notes they check: the patch, the validator and the query are notes too, synced and versioned like the rest. A new note in `docs/decisions/` is typed and checked on its first sync, with nothing to set up. An agent that writes a runbook without a description gets an error in the sync output, and a human sees the same list in the admin panel.
