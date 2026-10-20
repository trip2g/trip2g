---
title: "Calls dashboard"
free: true
lang_redirect: "[[ru/user/calls-dashboard]]"
---

Two layouts for a vault where calls are recorded, transcribed and reviewed: a timeline of long-running threads, and a screen where you check what was proposed about one call. Both read plain notes and write back with [[en/user/update_notes|updateNotes]]. Why they are built this way is in [[en/thoughts/calls-become-threads|Calls become threads]].

- [[en/user/calls-threads-demo|▶ Threads demo]]
- [[en/user/calls-call-demo|▶ One-call demo]]

The demos run on invented data. A visitor can click everything, but changes stay in the tab; the site owner's changes are saved to the notes.

## Install

Copy three files from [`docs/_layouts/calls/`](https://github.com/trip2g/trip2g/tree/main/docs/_layouts/calls) into your vault's `_layouts/calls/`: `threads.html`, `call.html` and `frontmatter.js`. Then make a note for each page:

```yaml
---
title: Threads
layout: calls/threads
---
```

```yaml
---
title: Call
layout: calls/call
owner: Dana Whitfield
---
```

The body of the page note is shown under the title.

## The notes the pages read

| Note | Default folder | What it holds |
|---|---|---|
| Transcript | `transcripts/` | `Speaker \| MM:SS` lines, each followed by what was said. Optional `title` and `duration_min` in frontmatter |
| Review | `calls/` | `type: review`, `transcript:` path, `speakers:` map from label to name, optional `speakers_confidence:` (`unknown` marks a label), `assumed: true` when the speakers or topics were guessed. The first line of the body is the agenda |
| Topic map | `topics/` | Same file name as the review. A summary paragraph, then a `## Topics` section of `MM:SS \| topic` lines |
| Thread | `threads/` | `type: thread`, `group`, `kind`, `status`, `people`, `aliases`, optional `priority` and `hidden`. A `## Chronology` list, one line per call, and a `## Open items` list |
| Person | `people/` | One note per person; its title is the name. `confirmed: false` marks a guess |
| Category | `categories/` | `type: call-category`, `order`, optional `kind` (`client` turns on the sales score) |
| Edits | `call_edits/` | Written by the one-call page, one note per call |

A call's id is the file name of its review note. The topic map and the edits note use the same name.

A chronology line starts with a link to the call, labelled with its date:

```markdown
## Chronology

- [[calls/c8e1a5|2026-07-31]]: Tax due on the 25th.
  Decided: depreciate the roaster. Open: send the invoice.
- 2026-10-01: The accountant sent a template by email.
```

A line that links to a review or a transcript is a dot on the timeline, dated by the link's text. A line without one, such as a letter or a decision made outside a call, appears in the thread's chain but not as a dot. `Decided:` and `Open:` start new lines in the side panel.

## Threads page settings

| Key | Default | Meaning |
|---|---|---|
| `threads` | `threads/*` | Glob of thread notes |
| `reviews`, `transcripts` | `calls/*`, `transcripts/*` | Globs that tell a call link from any other link |
| `thread_type` | `thread` | The `type` a thread note carries |
| `chronology_heading` | `Chronology` | Heading of the chronology list |
| `open_heading` | `Open items` | Heading of the open items list |
| `outcome_labels` | `[Decided, Open]` | Words that start a new line in the chain |
| `groups` | empty | Order of the groups; without it, the most active group comes first |
| `stale_days` | `21` | Days without a call after which a thread is quiet |
| `span_months` | `3` | Months shown on open |
| `today` | the current date | Fixes "today", for a demo |
| `locale` | `en` | Dates, month names and plural forms |
| `ui` | English | Interface strings, see below |

Inside a group, threads are sorted by `priority`, then by activity: recent calls weigh more, and the weight halves every three weeks without a call. **Hide thread** sets `hidden: true` in the thread's frontmatter; **Hidden** in the toolbar shows hidden threads again.

## One-call page settings

| Key | Default | Meaning |
|---|---|---|
| `owner` | empty | Your name as it appears in `speakers`; that label shows "this is you" |
| `reviews`, `topics`, `transcripts` | `calls/*`, `topics/*`, `transcripts/*` | Globs |
| `people`, `threads`, `categories` | `people/**`, `threads/*`, `categories/*` | Globs |
| `edits` | `call_edits` | Folder for the edits notes |
| `people_folder` | `people` | Where **+ new person** creates a note |
| `review_type`, `thread_type`, `category_type` | `review`, `thread`, `call-category` | The `type` values the page looks for |
| `topics_heading` | `Topics` | Heading of the topic map section |
| `topic_line` | `{tc} \| {text}` | Shape of a topic line |
| `transcript_line` | `{who} \| {tc}` | Shape of a transcript speaker line |
| `pending_since` | empty | Date; transcripts from it on that have no review are listed as waiting |
| `transcripts_inline` | `false` | Put every transcript into the page instead of loading it on demand |
| `scores` | three scores | List of `key`, `label` and optional `only_kind` |
| `subgraph` | empty | Added to the notes the page creates |
| `locale`, `ui` | `en`, English | As above |

A call counts as **waiting for your check** while it has `assumed: true` and its edits note lacks any of the four `verified` marks: speakers, category, threads, topics. **All verified** sets all four and opens the next call that waits.

Edits save themselves a second after the last change. The page keeps them in `call_edits/<id>.md`, never in the review: the review belongs to whoever wrote it, and the edits note belongs to the page. Saving checks the note's hash first, so an edit made elsewhere in the meantime is not overwritten.

Transcripts are loaded one at a time with the `ReadNote` query, which needs the site owner's session. A public demo sets `transcripts_inline: true` instead.

## Interface language

Every string has an English default. To translate, list the keys under `ui` in the page note; the Russian demo pages carry the full list. A count takes plural forms:

```yaml
locale: ru
ui:
  search: "Найти нить"
  calls:
    one: "{n} встреча"
    few: "{n} встречи"
    many: "{n} встреч"
    other: "{n} встречи"
```

The forms follow `Intl.PluralRules` for the `locale`.

## Limits

- The page puts every review, thread and topic map matched by its globs into the HTML. A few hundred calls are fine; thousands are slow.
- The layouts read frontmatter in the browser with a small parser (`frontmatter.js`): maps, lists and `|` / `>` blocks, no anchors or tags.
- Nothing links a call to a thread automatically. The one-call page lists the threads whose notes link to that call, and saves the threads you pick in the edits note; whoever keeps the thread notes adds the chronology line.
