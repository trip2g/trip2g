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

- **8 transcript notes** holding the raw, verbatim Krisp transcript, one per call, as the auditable source
- **8 call notes** with an inferred title, a strong first paragraph, a link back to the raw transcript, and a разбор where every topic pairs a one-line summary with the verbatim transcript segment quoted underneath, cited by timerange
- **220 concept notes**: people, tools, projects, terms, each with aliases and mentions that quote the exact segment where the term came up
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

There are two ways to run this. The standalone way is a single Python script (`build_vault.py`) you run over a batch of transcripts: good for a one-off run and for understanding the stages. The trip2g-native way is an **event-driven fleet cascade**: three chained agent role-notes, each triggered by a change-webhook on the note the previous stage wrote. Drop a raw transcript into the vault and the knowledge base assembles itself. Both do the same work; the cascade just wires the stages to note-change events instead of a loop.

All LLM work is gpt-5.4-mini. It won a head-to-head benchmark against the cheaper nano on every stage: more stable topic boundaries (75% run-to-run match vs 67%), no junk concepts, no invented timecodes.

**Stage 1, Ingest** (`executor: code`, no LLM, source-specific). A Python code role pulls the raw calls from the Krisp API and writes each one verbatim as a note in `transcripts/`. This is the only stage that knows about Krisp; everything after it is source-agnostic. The transcript note is the auditable source of truth, and the разбор links back to it, so you can always check a claim against the raw words. Time comes from the call id, not the clock: the Krisp id is a UUIDv7 with a millisecond timestamp in the upper bits, so `019f178c` decodes to 2026-06-30 08:01 UTC, 15:01 in the user's timezone. That is the only source for `created_at`, for sorting, and for daily bucketing. Local clocks and file mtimes lie; the id does not.

**Stage 2, Topic segmentation** (разметка тем). A change-webhook on `transcripts/**` triggers this role when a new transcript appears. It reads the transcript and marks topic boundaries with the coarse-granularity prompt: mark only major topic changes, take timecodes verbatim, small talk at the start is one topic. The same pass infers a title and tries to name the speakers, because Krisp's calendar title is unreliable. Inferred metadata gets `title_source: inferred` and `needs_review: true`, and you confirm by editing the note.

**Stage 3, Extraction and distribution** (растаскивание заметок). The segments being written trigger the last role. It pulls typed concepts (people, orgs, projects, tools, terms), decisions, open questions, and actions from each segment; named entities only, so "the new employee" is not an entity. Cross-call dedup happens here, and it is the hard part: speech-to-text mangles names, and the run heard "Cloud Code" and "Клод Код" for Claude Code, and "век ромбс" for Backrooms. An alias table catches exact repeats for free; the rest go to one reconcile call per transcript that sees the candidates with context plus the existing glossary and answers MERGE or NEW, and every merge teaches the table new spellings. Then this role writes the results out with plain code: it updates concept notes, appends to topic logs, writes the daily-note action checkboxes, and writes the call разбор note that links back to the transcript from stage 1.

**Provenance by quoting, not paraphrasing.** Every claim in the vault cites its source, and the citation is the source: because the boundaries are already computed, each topic slices its exact transcript lines and embeds them as a blockquote with a timerange link. The model's one-line summary sits above as the value-add; the verbatim quote sits below as evidence that cannot hallucinate. Concept mentions work the same way, they quote the segment where the term came up. You can see this on the CRISP concept note from the run: the summary reads cleanly, while the quote underneath preserves the raw speech-to-text ("...прогоняю и закидываю, например, там в Charge 5 или в Cloud..."), so the mishears stay visible in the evidence and never leak into the synthesis.

Net: three roles, each woken by the note the previous one wrote. This is dogfooding, trip2g's own change-webhooks plus a fleet code executor building a knowledge base out of the same notes it lives in.

### Navigation

The vault is a regular trip2g site, so the navigation is three magazine pages plus the graph:

- `/` lists calls newest first; the card is the first paragraph of the call note, which is why the pipeline writes it as a strong 2-3 sentence takeaway
- `/daily` lists days; the card shows the day's checkboxes
- `/concepts` lists terms sorted by `mentions`, so the most discussed concepts float to the top
- `/log` lists topic logs by number of entries

One contract detail matters: the frontmatter date key is `created_at` with an RFC3339 string value. A key named `date` is silently ignored, and the note falls back to sync time, which breaks both sorting and daily bucketing.

Inside a note, wikilinks carry you sideways: from a daily entry to the call, from the call to a concept, from a concept mention back into another call.

## Set it up yourself

You need Krisp (or any recorder that produces `Speaker | MM:SS` transcripts), an OpenRouter key, Python, and a trip2g instance. Start with the standalone script; move to the cascade once it works.

1. **Ingest (record + store).** Krisp installs as a virtual microphone and speaker, so it records calls from any app and produces speaker-separated transcripts with timecodes. Fetch each call via the Krisp API and write it verbatim as a `transcripts/YYYY-MM-DD_slug.md` note. Decode the time from the id, not the clock: take the first 8 hex chars, `int(prefix, 16) << 16` gives milliseconds since epoch, convert to your timezone, and use it for `created_at`, filenames, and daily bucketing.
2. **Validate the transcript.** If timecodes appear less than once per ~20 lines, quarantine the call: on defective input the model invents timecodes, on normal input it never does.
3. **Segment.** One gpt-5.4-mini call per transcript: `[MM:SS] topic` lines, major changes only, at least 2 minutes between topics, plus TITLE / SLUG / SPEAKERS lines for the metadata fallback.
4. **Extract.** One call per 8-16 minute chunk: JSON list of typed concepts with aliases, a 1-3 sentence summary quoting only what was said here, plus actions with owner and due.
5. **Reconcile.** Keep `aliases.json` in the vault. Exact-match first, then one LLM call per transcript for the rest. Append mentions to existing notes; never rewrite a note body.
6. **Emit and publish.** Write the folders (`transcripts/`, `calls/`, `daily/`, `concepts/`, `log/`), the magazine `index.md` files, `_header.md` with the nav list. Each call note carries `transcript: "[[transcripts/...]]"` in frontmatter and an inline link to the raw source. Sync the vault to your trip2g instance with the Obsidian plugin or the CLI.
7. **Review.** Open the site, tick checkboxes on the daily note, fix titles on `needs_review` notes. Your edits are the confirmation loop; the pipeline appends, you correct.
8. **Go event-driven (optional).** Turn the three stages into fleet role-notes and wire change-webhooks: `transcripts/**` triggers segmentation, the segments trigger extraction. Now dropping a raw transcript into the vault builds the rest by itself.

Budget note: cap your spend. The whole 8-call run above stayed at $0.42, and re-runs cost nothing if you cache raw LLM responses per stage.

The stages generalize past calls. Only ingest is Krisp-specific; segment, extract, reconcile, emit work for any long text: books, YouTube, support threads. In the trip2g agent runtime (fleet) each stage is a role-note triggered by the note the previous stage wrote, so the pipeline itself lives in the same vault it builds.
