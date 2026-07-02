---
title: "Calls into a knowledge base"
free: true
lang_redirect: "[[ru/user/calls-knowledge-base]]"
---

Krisp records your calls. An LLM pipeline turns them into a linked, navigable knowledge base: call notes, a glossary of terms, daily notes with action checkboxes, and append-only topic logs. Not a summary and a chat bot: a graph that grows with every call.

In this article:

- [What you get](#what-you-get)
- [How it works](#how-it-works)
  - [Two note techniques](#two-note-techniques)
  - [The pipeline](#the-pipeline)
  - [Navigation](#navigation)
- [Set it up yourself](#set-it-up-yourself)

## What you get

After processing 8 real calls from two and a half weeks, the vault contains:

- **8 call notes** with an inferred title, a strong first paragraph, a topic map with timecodes, decisions, and open questions
- **220 concept notes**: people, tools, projects, terms, each with aliases and a list of mentions across calls
- **9 daily notes**: action checkboxes on top, a dated log of the day below
- **3 topic logs** for recurring themes, growing append-only with each new call

Total cost of the run: $0.42 in LLM calls, about 5 cents per call. A long 50-minute call costs around 15 cents.

The difference from "summary plus chat bot" is what happens on call number two. A summary dies after you read it. Here the same knowledge lives in one note and accumulates evidence. The note for the project "Hermes" collected three mentions from three different calls, each adding a new angle: first a testing target, then "a strong OpenClaw clone", then an agent that keeps notes in Obsidian. Nobody wrote that note. It grew.

## How it works

### Two note techniques

The vault rests on two note-taking habits, both automated.

**Daily notes: checkboxes on top, log below, no summary.** Every calendar day gets a note. The top is a task list: one checkbox per action item extracted from that day's calls, with the owner and a link to the source call. Below, under `## Лог`, a dated entry per call: time, link, one-line takeaway with links to the key concepts. No "day in review" paragraph; the log is the review.

```markdown
---
title: "2026-06-19, пятница"
created_at: "2026-06-19T00:00:00+07:00"
type: daily
calls_count: 2
---
- [ ] Настя: составить список источников поиска кандидатов ([[2026-06-19_ai-agenty-hr-sourcing-automation]])
- [ ] Speaker 3: созвониться с Юлией по автоматизации маркетинга ([[...]])

## Лог

- 15:37 [[2026-06-19_demo-prodazhnik-avtomatizatsiya|Запуск демо-продажника]] — ...
- 20:01 [[2026-06-19_ai-agenty-hr-sourcing-automation|AI-агенты для HR]] — ...
```

If someone says "I'll do it tomorrow" on the call, the checkbox lands on the next day's daily note. In the test run, June 13 has no calls at all, yet its daily note exists: one deferred task from June 12 lives there. On the published page, task checkboxes are interactive for the site admin, so ticking off the day happens right in the browser, and the state is saved back to the note.

**Topic logs: append-only, dated headings.** When a topic keeps coming back across calls, it gets a log note. Each new mention is appended under a `### [[YYYY-MM-DD]]` heading that links to the daily note. Old entries are never rewritten, only new ones added, so the note reads as the history of a thought:

```markdown
### [[2026-06-19]]
Так здесь называют вариант поставки, при котором система разворачивается
на собственных серверах компании. ([[call note]], 46:14)

### [[2026-06-26]]
На такой коробке с агентами можно строить сервисы и даже
«настоящую компанию». ([[call note]], 13:01)

### [[2026-06-30]]
Продукт, который собеседник заканчивает и планирует тестировать. (...)
```

That is the log for "коробка" from the test run: four entries, four calls, the idea visibly evolving over eleven days.

### The pipeline

Five stages, all LLM work on gpt-5.4-mini. It won a head-to-head benchmark against the cheaper nano on every stage: more stable topic boundaries (75% run-to-run match vs 67%), no junk concepts, no invented timecodes.

1. **Time from the id, not the clock.** The Krisp call id is a UUIDv7 with a millisecond timestamp in the upper bits. `019f178c` decodes to 2026-06-30 08:01 UTC, 15:01 in the user's timezone. This is the only source for `created_at`, for sorting, and for placing the call into a daily note. Local clocks and file mtimes lie; the id does not.
2. **Topic boundaries.** One call per transcript with a coarse-granularity prompt: mark only major topic changes, take timecodes verbatim from the transcript, small talk at the start is one topic. The same call infers a title and tries to name the speakers, because Krisp's calendar title is unreliable. Inferred metadata gets `title_source: inferred` and `needs_review: true`, and you confirm by editing the note.
3. **Extraction.** Each segment yields concepts (people, orgs, projects, tools, terms), decisions, open questions, and actions as JSON. Named entities only: "the new employee" is not an entity. Actions carry an owner and a today/tomorrow flag.
4. **Cross-call dedup.** The hard part. Speech-to-text mangles names: the run heard "Cloud Code" and "Клод Код" for Claude Code, and "век ромбс" for Backrooms. An alias table catches exact repeats for free; the leftovers go to one reconcile call per transcript, which sees the new candidates with context plus the existing glossary and answers MERGE or NEW. Every merge teaches the alias table new spellings.
5. **Emit.** Plain code, no LLM: write call notes, append mentions to concept notes, rebuild daily notes and topic logs, update the alias table.

### Navigation

The vault is a regular trip2g site, so the navigation is three magazine pages plus the graph:

- `/` lists calls newest first; the card is the first paragraph of the call note, which is why the pipeline writes it as a strong 2-3 sentence takeaway
- `/daily` lists days; the card shows the day's checkboxes
- `/concepts` lists terms sorted by `mentions`, so the most discussed concepts float to the top
- `/log` lists topic logs by number of entries

One contract detail matters: the frontmatter date key is `created_at` with an RFC3339 string value. A key named `date` is silently ignored, and the note falls back to sync time, which breaks both sorting and daily bucketing.

Inside a note, wikilinks carry you sideways: from a daily entry to the call, from the call to a concept, from a concept mention back into another call.

## Set it up yourself

You need Krisp (or any recorder that produces `Speaker | MM:SS` transcripts), an OpenRouter key, Python, and a trip2g instance.

1. **Record.** Krisp installs as a virtual microphone and speaker, so it records calls from any app and produces speaker-separated transcripts with timecodes.
2. **Decode call time.** Take the first 8 hex chars of the call id: `int(prefix, 16) << 16` gives milliseconds since epoch. Convert to your timezone. Use it for `created_at`, filenames, and daily bucketing.
3. **Validate the transcript.** If timecodes appear less than once per ~20 lines, quarantine the call: on defective input the model invents timecodes, on normal input it never does.
4. **Segment.** One gpt-5.4-mini call per transcript: `[MM:SS] topic` lines, major changes only, at least 2 minutes between topics, plus TITLE / SLUG / SPEAKERS lines for the metadata fallback.
5. **Extract.** One call per 8-16 minute chunk: JSON list of typed concepts with aliases, a 1-3 sentence summary quoting only what was said here, plus actions with owner and due.
6. **Reconcile.** Keep `aliases.json` in the vault. Exact-match first, then one LLM call per transcript for the rest. Append mentions to existing notes; never rewrite a note body.
7. **Emit and publish.** Write the four folders (`calls/`, `daily/`, `concepts/`, `log/`), the three magazine `index.md` files, `_header.md` with the nav list. Sync the vault to your trip2g instance with the Obsidian plugin or the CLI.
8. **Review.** Open the site, tick checkboxes on the daily note, fix titles on `needs_review` notes. Your edits are the confirmation loop; the pipeline appends, you correct.

Budget note: cap your spend. The whole 8-call run above stayed at $0.42, and re-runs cost nothing if you cache raw LLM responses per stage.

The same five stages generalize past calls. Only stage 1, fetching transcripts and decoding time, is Krisp-specific; segment, extract, reconcile, emit work for any long text: books, YouTube, support threads. In the trip2g agent runtime (fleet) each stage is a note with a prompt applied per segment, so the pipeline itself can live in the same vault it builds.
