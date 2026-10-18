---
title: Note validators
free: true
lang: en
lang_redirect: "[[ru/user/validators]]"
---

trip2g stores every note in its database and uses it as data. Templates, lists, search and the API all read fields from the frontmatter and sections from the body. A card needs an image, a list needs a date, an SEO snippet needs a description. When a field is missing or a section has the wrong name, the page breaks quietly: an empty card, a list in the wrong order, a cut-off snippet.

Language models write markdown well, but not the same way twice. One run produces `## Steps`, the next produces `## How to`. One run puts `tags` in a list, the next writes them as a string.

A validator is a note with rules written in [Jsonnet](https://jsonnet.org). You attach it to other notes, and on every sync trip2g checks each of them against those rules. The sync still goes through. A note that breaks a rule gets a warning, and you see exactly what to fix.

### Quick start

1\. Create a note `validators/required_frontmatter.md`. The rules go in a `jsonnet` code block:

````markdown
Every article needs a title and a description, both strings.

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
````

2\. Attach it to a note through the `validate` field:

```yaml
---
title: Publishing a vault
validate: "[[required_frontmatter]]"
---
```

3\. Sync the vault. If the note has no `description`, the sync output shows:

```
notes/article.md
  [CRITICAL] validate validators/required_frontmatter.md: missing frontmatter field: description
```

The note is still saved and published. The warning marks it until you fix it.

### Where you see the result

- **Sync CLI.** Messages are listed under each note. When there is at least one `error`, the CLI exits with code 1, so a script or an AI agent knows it has to fix something. Messages come with a sync that pushes notes: when nothing changed, the CLI pushes nothing and shows nothing. See [[en/user/cli|CLI sync]].
- **Admin panel.** Dashboard → Note Warnings always lists every note with its messages.
- **Locally, without a server.** `trip2g lint <folder>` prints the same messages. It is stricter than sync: it exits with code 1 on an `error` and on a `warning` alike. Put it in CI to stop invalid notes before they reach the site. See [[en/user/lint|Linting your vault]].

### Attaching validators

`validate` takes one validator or a list:

```yaml
validate:
  - "[[required_frontmatter]]"
  - "[[seo]]"
  - validators/session_has_image.md
```

A validator can be named as a wikilink or as a path, with or without `.md`. Lookup works the same way as for wikilinks in the text; a `#heading` part is ignored. A note passes only when none of its validators returns an `error`.

#### One validator, many notes

Write each rule once and reuse it. Keep the validators in one folder, such as `validators/`, one rule per note: "required fields", "SEO", "every session has an image". Each note type then lists the set it needs. A workshop gets `session_has_image` and `seo`. An article gets `required_frontmatter`, `seo` and `sections_order`.

To check a whole folder, don't edit every note: set `validate` with a [[en/user/frontmatter-patches|frontmatter patch]]:

````markdown
---
type: frontmatter-patch
include: ["blog/**"]
---

```jsonnet
{ validate: ['[[required_frontmatter]]', '[[seo]]'] }
```
````

Validators see the frontmatter after patches are applied, so a field added by a patch counts as present.

### Writing a validator

Each `jsonnet` block of a validator has access to:

| Name | What it holds |
|---|---|
| `meta` | the note's frontmatter, after patches |
| `path` | the note's path, such as `blog/post.md` |
| `nodes` | the note's body as a list of elements, top to bottom |
| `std.native('regexMatch')(pattern, text)` | `true` when the text matches a regular expression |

Every element in `nodes` has a `type`, a `line` (its line in the file, frontmatter included) and a `depth` (how deeply it is nested):

| `type` | Extra fields |
|---|---|
| `heading` | `level` (1–6), `text` |
| `paragraph` | `text` (a tight list item's text comes as a `paragraph` too) |
| `image` | `src`, `alt` (both `![alt](src)` and `![[src\|alt]]`) |
| `link` | `url`, `text` |
| `wikilink` | `target`, `embed` |
| `code_block` | `lang`, empty when the language is not given |
| `list` | `ordered` |
| `list_item`, `blockquote`, `table` | — |

The block returns a list of messages. An empty list means the note is fine:

```jsonnet
[
  {level: 'error', msg: 'what is wrong', line: 12},
  {level: 'warning', msg: 'worth a look'},
]
```

- `error` marks the note as invalid: `CRITICAL` in the CLI, and the CLI exits with code 1.
- `warning` is advice: `WARNING`, and the sync CLI exit code stays 0. `trip2g lint` still fails on it.
- `line` is optional. When given, the message shows it.

A validator that is itself broken never passes silently. A typo in Jsonnet, a result that is not a list, an unknown `level` or a validator that can't be found all produce a `CRITICAL` message on the note.

A validator can have several `jsonnet` blocks. Each one runs, and their messages add up.

### Examples

These validators are taken from a demo vault that the trip2g test suite runs on every build.

#### Required fields

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

#### Sections in a set order

An article has exactly these second-level sections, in this order:

```jsonnet
local want = ['Intro', 'Steps', 'FAQ'];
local got = [n.text for n in nodes if n.type == 'heading' && n.level == 2];

if got == want then []
else [{
  level: 'error',
  msg: 'sections must be %s, got %s' % [
    std.join(' → ', want),
    std.join(' → ', got),
  ],
}]
```

#### Every session has an image

The note is split into sections at each `##`, and the images in each section are counted. `std.foldl` walks the list of elements and carries the sections along:

```jsonnet
local last(xs) = xs[std.length(xs) - 1];
local init(xs) = xs[:std.length(xs) - 1];

local sections = std.foldl(
  function(acc, n)
    if n.type == 'heading' && n.level == 2 then
      acc + [{title: n.text, line: n.line, images: 0}]
    else if n.type == 'image' && std.length(acc) > 0 then
      init(acc) + [last(acc) + {images: super.images + 1}]
    else acc,
  nodes,
  [],
);

[
  {level: 'error', line: s.line, msg: '"%s" has no image' % s.title}
  for s in sections
  if std.startsWith(s.title, 'Session') && s.images == 0
]
```

#### Images and code

A missing alt text is a warning. A code block without a language is an error:

```jsonnet
[
  {level: 'warning', line: n.line, msg: 'image %s has no alt text' % n.src}
  for n in nodes
  if n.type == 'image' && n.alt == ''
] + [
  {level: 'error', line: n.line, msg: 'code block without a language'}
  for n in nodes
  if n.type == 'code_block' && n.lang == ''
]
```

#### Description length and date format

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

### What else to check

- **The contract with a template.** A layout reads `cover`, `description` and a `## Summary` section. The validator makes sure every note that uses the layout has them.
- **Notes written by agents.** An agent writes a note, runs sync, gets exit code 1 with a list of messages, and fixes the note. This loop catches what the model got wrong in a given run.
- **Data in notes.** Events, products, people: `price` is a number, `status` is one of `draft`, `review`, `published`, and the date is in ISO format. Lists, charts and the API then get clean data.
- **SEO and accessibility.** A single `#` heading, no skipped levels (`##` followed by `####`), alt text on images, a description under 160 characters.
- **House style.** Banned words, link text like "click here", sentences that run too long. A paragraph's text is in the `text` field of `paragraph` elements. Text inside table cells is not in `nodes`.
- **Course structure.** Every lesson has a video and a `## Quiz` section.
- **Telegram posts.** The text fits the limit, and there is no more than one image.

### Limits

- **A validator sees one note.** It can't check that a slug is unique across the vault, or that some other note links here.
- **Validators don't block sync.** An invalid note is saved and published with a warning. To block it, run `trip2g lint` in CI before syncing.
- **A validator note is published like any other note.**
- **Results are remembered between syncs.** A note is checked again only when it changes, when one of its validators changes or starts pointing to another note, or when any frontmatter patch changes. A server start, saving a patch in the admin panel, or a sync that brings a patch note checks every note that has `validate` again. With seven validators on about 850 notes, that pass takes about 4 seconds.

How it works inside, and what it costs: [[dev/note_validators|note validators (dev)]].
