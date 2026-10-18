---
title: A file name that explains the file
free: true
lang_redirect: "[[ru/thoughts/file-naming-convention]]"
---

The [AI Mindset](https://aimindset.org/) team names the notes in its vaults by one pattern:

```
{project} {type} title – YYYY-MM-DD – tool.md
```

For example, `{blog} {article} Weekly summary – 2026-10-18 – Cursor.md`. The project and the content type go first, in braces. The date always comes near the end, after a dash. The last part names the tool that wrote the note: `Cursor`, `Claude Code`, `ChatGPT`, or `Manual` for a note written by hand.

## Why it works

The name answers three questions before you open the file: what it belongs to, what kind of text it is, and when it was written. A person scanning a folder sees it. So does an agent that gets a list of files and has to pick one, because a file list is often all an agent has before it spends a read.

The tool at the end tells you where a text came from. A note written in Claude Code and one written by hand carry different weight when you decide what to trust.

## Why it breaks

The convention's own rules call a date in the middle of the name a critical mistake, and come back to it in both of their wrong examples. A rule gets written that loudly when it keeps being broken.

Agents break it, because a model writes a name a little differently every time: `{2026-10-18}` in braces, an ordinary hyphen instead of a dash, underscores instead of spaces. My own vault has four notes named by this convention, and three of them came out as `{rule}_…_–_2025_12_18_–_Cursor.md`. A convention that lives only in a note is a wish.

## Checking it in trip2g

In trip2g a rule like this becomes a [[en/user/validators|validator]]: a note with a Jsonnet block that checks each note's name on every sync. Put it in `validators/file_name.md`:

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

The validator splits the name at ` – `, a dash with a space on each side, so a plain hyphen fails right away. When the note has a `date` in its frontmatter, it must match the date in the name.

To make the whole vault follow the convention, don't add `validate` to every note. One [[en/user/frontmatter-patches|frontmatter patch]] attaches the validator to every note, new ones included:

````markdown
---
type: frontmatter-patch
include: ["**/*.md"]
exclude: ["validators/**", "patches/**"]
---

Every note in the vault follows the naming convention.

```jsonnet
local own = std.get(meta, 'validate', []);
{
  validate: (if std.isString(own) then [own] else own)
    + ['[[file_name]]'],
}
```
````

The patch adds `file_name` to whatever validators a note already lists, instead of replacing them. `exclude` keeps out the validators and patches themselves, along with any other service notes whose names follow a different rule.

On the next sync, a badly named note gets a message such as:

```
[CRITICAL] validate validators/file_name.md: name must be "{project} {type} title – YYYY-MM-DD – tool"
```

The sync CLI exits with code 1, so an agent that wrote the note sees that it has to rename it. The admin panel lists every such note, and `trip2g lint` reports them in CI before a sync.

## What it doesn't do

The validator marks a note; it doesn't rename it and doesn't block the sync. The list of tools is yours to edit. Notes that live by other rules, such as daily notes named `2026-10-18.md`, go into `exclude`, or get a validator of their own.
