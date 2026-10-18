---
title: Jsonnet in trip2g
free: true
lang: en
lang_redirect: "[[ru/user/jsonnet]]"
---

Four trip2g features take a short program instead of a fixed setting: a frontmatter patch decides which fields to add, a validator decides what is wrong with a note, a note query decides which notes a page gets, a webhook transform decides what to send. These programs are written in [Jsonnet](https://jsonnet.org), a small language for producing JSON. This page covers the part of Jsonnet you need for trip2g and where trip2g runs it.

### What Jsonnet is

Any JSON document is already valid Jsonnet. On top of JSON, Jsonnet adds variables, conditions, functions and loops over lists, and the program's result is always JSON: an object, a list, a string or a number. Jsonnet has no side effects. A program can't change files, call the network or read the clock; it only computes a value.

trip2g runs every program in a sandbox: `import` is blocked, and recursion deeper than 500 calls stops with an error instead of hanging.

### Where trip2g uses it

| Feature | The program gets | It returns |
|---|---|---|
| [[en/user/frontmatter-patches\|Frontmatter patches]] | `meta` (the note's frontmatter), `path` | an object whose fields are merged into the frontmatter |
| [[en/user/validators\|Note validators]] | `meta` (after patches), `path`, `nodes`, `std.native('regexMatch')` | a list of messages `{level, msg, line?}` |
| [[en/user/note-queries\|Note queries]] | `notes` (the notes the caller can read), `params` | any JSON, returned to the page |
| [[en/user/webhooks\|Webhook]] body transform | `std.extVar('change')`, `std.extVar('attached_notes')` as JSON strings | the request body to send instead of the default one |

In patches, validators and note queries the program sits in a ```` ```jsonnet ```` block inside an ordinary note. The webhook transform is the `transformJsonnet` field of a change or cron webhook in the GraphQL API; it replaces the whole body and can't be combined with **Pass API key**. `attached_notes` exists only when the webhook attaches notes; reading a missing one stops the program with `Undefined external variable`.

### Syntax in ten minutes

#### Values and comments

```jsonnet
// A comment. JSON as is, plus trailing commas and unquoted keys.
{
  title: 'Hello',
  "tags": ['a', 'b'],
  draft: false,
  views: 3,
}
// → {"draft": false, "tags": ["a", "b"], "title": "Hello", "views": 3}
```

Strings take single or double quotes. Keys don't need quotes when they are plain words.

#### Variables

`local` names a value. It ends with `;` and is visible in everything that follows:

```jsonnet
local site = 'trip2g';
local year = 2026;
{ footer: site + ' © ' + year }
// → {"footer": "trip2g © 2026"}
```

`+` joins strings, adds numbers, concatenates lists and merges objects. A number added to a string becomes text.

#### Fields of an object

```jsonnet
local meta = { title: 'Post', tags: ['go'] };
{
  title: meta.title,
  hasTags: std.objectHas(meta, 'tags'),
  author: std.get(meta, 'author', 'unknown'),
}
// → {"author": "unknown", "hasTags": true, "title": "Post"}
```

`meta.author` fails when the field is missing. `std.get(meta, 'author', default)` returns the default instead, and `std.objectHas` asks whether the field is there. In patches and validators, use these two for fields a note may not have.

#### Conditions

```jsonnet
local meta = { status: 'draft' };
{
  visible: if meta.status == 'published' then true else false,
  label: if meta.status == 'draft' then 'Draft' else '',
}
// → {"label": "Draft", "visible": false}
```

`if` is an expression with a value, like a ternary operator. **Always write `else`:** without it, a false condition gives `null`. A validator that returns `null` instead of a list gets a `CRITICAL` message.

#### Lists and comprehensions

```jsonnet
local tags = ['go', 'Draft', 'web'];
{
  lower: [std.asciiLower(t) for t in tags],
  noDrafts: [t for t in tags if t != 'Draft'],
  count: std.length(tags),
}
// → {"count": 3, "lower": ["go", "draft", "web"], "noDrafts": ["go", "web"]}
```

`[expression for x in list if condition]` builds a new list. Validators are mostly written this way: one message for each element that breaks a rule.

#### Text

```jsonnet
local name = 'Notes – 2026-10-18 – Cursor';
local parts = std.split(name, ' – ');
{
  parts: parts,
  msg: 'date %s, tool %s' % [parts[1], parts[2]],
  isNotes: std.startsWith(name, 'Notes'),
}
// → {"isNotes": true, "msg": "date 2026-10-18, tool Cursor",
//    "parts": ["Notes", "2026-10-18", "Cursor"]}
```

`'…%s…' % value` fills placeholders, like `printf`: `%s` for text, `%d` for a whole number. With several values, pass a list.

#### Functions

```jsonnet
local err(msg) = { level: 'error', msg: msg };
local missing(meta, f) = !std.objectHas(meta, f);
local meta = { title: 'Post' };
[err('missing ' + f) for f in ['title', 'description'] if missing(meta, f)]
// → [{"level": "error", "msg": "missing description"}]
```

A function is a `local` with parameters. Small helpers like `err` keep a validator short.

#### Walking a list with state

When each element depends on the ones before it, for example "which section is this image in", use `std.foldl`. It walks the list and carries an accumulator along:

```jsonnet
local nodes = [
  { type: 'heading', text: 'Intro' },
  { type: 'image' },
  { type: 'heading', text: 'Steps' },
];
std.foldl(
  function(acc, n)
    if n.type == 'heading' then acc + [{ title: n.text, images: 0 }]
    else acc[:std.length(acc) - 1]
      + [acc[std.length(acc) - 1] + { images: super.images + 1 }],
  nodes,
  [],
)
// → [{"images": 1, "title": "Intro"}, {"images": 0, "title": "Steps"}]
```

`acc[:n]` is a slice: all but the last section. `obj + { images: super.images + 1 }` copies the object with one field changed.

### Functions you will use most

| Function | What it does |
|---|---|
| `std.length(x)` | length of a list, string or object |
| `std.objectHas(obj, 'f')`, `std.get(obj, 'f', default)` | is the field there; the field or a default |
| `std.isString(x)`, `std.isArray(x)`, `std.isNumber(x)` | the type of a value |
| `std.member(list, x)` | is `x` in the list |
| `std.startsWith(s, p)`, `std.endsWith(s, p)` | does the text start or end with `p` |
| `std.split(s, sep)`, `std.join(sep, list)` | split text into a list; join a list into text |
| `std.asciiLower(s)`, `std.asciiUpper(s)` | change case (latin letters) |
| `std.substr(s, from, len)` | part of a text |
| `std.toString(x)` | any value as text |
| `std.foldl(f, list, init)` | walk a list with an accumulator |
| `std.parseJson(s)` | turn a JSON string into a value |

Jsonnet has no regular expressions of its own. In validators trip2g adds `std.native('regexMatch')(pattern, text)`; backslashes in the pattern are doubled, as in `'^\\d{4}$'`.

### Common mistakes

- **`if` without `else`** gives `null` for a false condition.
- **A missing field read as `meta.x`** stops the program with `Field does not exist: x`. Use `std.get`.
- **`local` without `;`.** Each `local` ends with `;` (or `,` when several are declared in a row).
- **A single backslash in a pattern.** `'\d'` is not a valid escape; write `'\\d'`.

### Where to read more

The full language reference and an online playground are on [jsonnet.org](https://jsonnet.org): the [tutorial](https://jsonnet.org/learning/tutorial.html) and the [standard library](https://jsonnet.org/ref/stdlib.html). To try a program on your own notes, write it as a validator and run `trip2g lint` on the folder: see [[en/user/lint|Linting your vault]].
