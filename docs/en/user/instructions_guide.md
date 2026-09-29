---
title: "Instructions for a knowledge base: teaching an agent to search"
free: true
lang_redirect: "[[ru/user/instructions_guide]]"
---

When an agent connects to your knowledge base over MCP, it knows nothing about how the base is organized. trip2g's search tools are the same for every base: `search`, `expand`, `note_html`, `similar`. But bases differ: a reference manual, a course, a wiki, a meeting log. Instructions are a map of your base plus the rules for searching it: where to start, how to phrase queries, how to answer, and what to do when there is no answer. This page shows how to write such instructions and how to check that they work. Measure first, then write.

## Why instructions matter

We tested this on a private course knowledge base: four Claude models, the same questions, with instructions and without. The details are in [[en/thoughts/do-agents-need-instructions|Do agents need instructions for a knowledge base]]. The main points:

- **Weak models need instructions.** Without them, Claude Haiku produced every failure in the run: once it decided it had no access to the base and answered from general knowledge. With instructions there were no failures, and it opened the right note in 12 questions out of 13 instead of 9.
- **Strong models find the instructions themselves.** Without a prompt on connect, Sonnet, Opus and Fable called the `instructions()` tool on their own in most runs. It is enough for them that instructions exist.
- **Format rules work.** The rule "answer with a quote from the note" raised the share of Haiku's answers with a verbatim quote from 0.12 to 0.81.

Hence a simple conclusion: write both the short instructions that arrive on connect, for weak models, and the detailed ones the agent requests itself, for everyone.

## Where instructions live

Instructions are ordinary notes with an `mcp_method` field in the frontmatter. The technical details are in the [[en/user/mcp|MCP server documentation]].

| Note | When the agent gets it | What to put in it |
|---|---|---|
| `mcp_method: initialize` | on connect; the client puts it into the system prompt | short, up to ~1500 characters: what the base is, where to start, how to search, how to answer |
| `mcp_method: instructions` | when the agent calls `instructions()` | detailed: routes by task type, queries that work, what the base doesn't cover |
| `mcp_method: <name>` | when a client connects to `/_system/mcp?method=<name>` | an alternative `initialize`: for a role or for a different model |

Add `free: true` if the base is open to anonymous clients. Otherwise only those with access get the note.

## Instructions as a skill for searching the base

It helps to think of instructions as an agent skill: "how to search this knowledge base". The structure matches skills in Claude and other agents, from general to specific, with each level loaded only when needed:

| Skill level | In the knowledge base | When it is in context |
|---|---|---|
| Description: what the skill is and when to use it | the `initialize` note | always, from the moment of connecting |
| Body: exactly how to act | the `instructions` note, the `instructions()` call | when the agent takes on a question about the base |
| Reference material | the index, decision maps, vocabulary, `_instructions.md` with routes | when a route leads there |

This gives the writing rules. `initialize` should answer two questions: what the base is and when to go to it. As in a skill's description, this is no place for detail, or it will take up the context of every conversation, including ones that have nothing to do with the base. The detail lives in `instructions` and the reference notes, which the agent opens on its own. A good test: if the agent reads only `initialize`, will it know whether to call `instructions()`?

## Step 1. Benchmark your base first

Don't write instructions from your head. First see where an agent trips on your base without them, and write the instructions for those spots.

**Collect 15–30 questions** your readers actually ask. Mix the kinds:

- direct: the answer is in one note;
- situational: "I have X, what do I do";
- detail: a number, threshold or field hidden deep in a long note;
- 3–5 questions the base **doesn't** answer. Without them you won't find out whether the agent makes things up.

For each question, write down which note answers it and what a correct answer must contain.

**Run two models without instructions**: one weak and cheap, one strong. Any MCP client will do. For each question, note:

| Question | Found the right note? | Opened it? | Tool calls | Answer grounded in the note? | Where it got lost |
|---|---|---|---|---|---|

Look not only at the outcome but at the path: which queries the agent sent to `search`, what came back, where it took a wrong turn. Almost every line of your instructions will be the answer to one of these mistakes.

## Step 2. Fix the base where search trips

Some mistakes are fixed by the structure of the base, not by instructions. That is more reliable: it helps any agent, even one that never reads the instructions.

| What the benchmark shows | What to change in the base |
|---|---|
| The agent doesn't know where to start and keeps rephrasing | Add an index note: a table of contents with short descriptions of each section |
| On "what if…" questions the agent finds pieces but not a path | Add decision maps: a note of "situation → steps → where to look" |
| The agent searches for a word the text doesn't use (a synonym, an English term, an old name) | Add `aliases` to the frontmatter: a note whose title or alias equals the whole query rises in the results |
| The right note exists but other notes outrank it | Make the title name the subject: "Sync", not "How we do things" |
| One huge page turns up for almost any query | Split it into smaller notes: a long page has more chances that some chunk of it happens to look like the query |
| The agent answers from drafts, service notes or outdated notes | Take them out of search: `search: false` in the frontmatter |
| The agent invents answers to what the base doesn't cover | Add a "What's not here" note and point to it from the instructions |

After the fixes, run the benchmark again. Some problems may already be gone.

## Step 3. Know what kind of base you have

Good routes depend on how the base is organized.

| Kind of base | Where to start | How to search | How to answer |
|---|---|---|---|
| **Reference, documentation** | the index page or the section for the topic | with a short canonical feature name, "sync" rather than the whole question; then `expand` down to the right section | link to the page and section, steps the way the docs give them |
| **Course, manual** | the course map, decision maps for situations | by the name of the stage or concept; for situations, the map first, then the notes it points to | the action from the course with a quote and attribution: lesson, section |
| **Wiki, digital garden** | hub notes by topic | find one note, then follow its connections: `similar` and the links in the text | a synthesis of several notes with a link to each |
| **Logs, minutes, chats** | a table of contents by date or project | put the date, project or names in the query; recent entries beat old ones | with the date: "per the minutes of 12 March…"; flag old entries |
| **Bilingual base** | an index in each language | in the language the notes are written in; the English term if the notes use it | in the question's language, with a link to the note |

## Step 4. Write `initialize`

The short note an agent gets on connect. Every model reads it, weak ones included, so be concrete and brief: commands, not reasoning. Up to 1500 characters is a good size.

What it should contain:

1. **What the base is and what it doesn't cover**, in a line or two.
2. **Where to start**: the index note, maps, hubs, with paths.
3. **How to search**: the recipe for your base, which queries work, `limit`, when to `expand`, when to `note_html`.
4. **How to answer**: link to the note, quote, format.
5. **What to do when nothing is found.** Phrase it so the agent searches first: "if two or three searches find no material, say so". A bare "no material, say there is none" can make a weak model give up before searching.
6. **Where the details are**: "before a complex question, call `instructions()`".

An example for a course:

```yaml
---
mcp_method: initialize
free: true
---
Knowledge base of a photography course: from camera settings to editing. It says nothing about buying gear or selling photos.

Start with `note_html(path="_instructions.md")`: it has routes by task type. For "what if…" situations, open a map from `maps/` in full.

Search briefly, with the name of a concept or stage: `search("shutter speed", limit=8)`, not the whole question. Once you find a note, read the section you need with `note_html(path=..., toc_path=[...])`.

Answer with the action the course gives, a link to the note and a quote from it. If 2–3 searches find no material, say so; don't answer from general knowledge.
```

## Step 5. Write the detailed `instructions` note

Strong models call it on their own, weak ones when `initialize` tells them to. This is the place for detail:

- **Routes by task type.** "A question like 'why are my photos blurry' → `maps/blurry-photos.md` → the note for that branch." Take the types from your benchmark questions.
- **Queries that work.** The phrasings that found the right note in the benchmark, and the ones that didn't. "Don't search 'how to shoot at night'; search 'long exposure' or 'ISO noise'."
- **Vocabulary.** The words the base uses for things, where they differ from everyday words.
- **Gaps.** What the base covers partly, what it doesn't cover at all, and where to send the reader then.
- **Answer format** in more detail: structure, length, how to quote, how to link.

## Step 6. Different instructions for different models and roles

The `?method=` parameter chooses which note a client gets instead of `initialize`. That lets you keep several variants:

- **for weak, cheap models**, a step-by-step version: "1. open the index; 2. find…; 3. answer with a quote". They need a fixed sequence;
- **for strong models**, a short one: what the base is and where the details are. They'll find the rest;
- **for separate roles**, for example "consultant for beginners" and "reference for curators".

```yaml
---
mcp_method: initialize_steps
free: true
---
Answer strictly step by step: …
```

A client with a weak model connects to `/_system/mcp?method=initialize_steps`, everyone else to `/_system/mcp`. Check each variant with its own benchmark run.

## Step 7. Re-check and keep it current

Run the same benchmark with instructions on the same two models and compare with the first run. Look at what got better and, separately, at what got worse. Typical regressions:

- **the agent stopped searching** and answers from the instructions or gives up too early: rewrite the "if not found" rule and add "search first";
- **the agent only follows routes** and misses what they don't cover: add "if there is no route, use `search`";
- **the instructions grew** and a weak model gets lost: shorten `initialize` and move the detail to `instructions`.

Instructions go stale along with the base: new sections appear, notes get renamed, routes change. After big changes to the base, run the benchmark again. Keep the benchmark questions next to the base so a check takes minutes.

## A prompt for an agent: let it write the instructions

Steps 1–5 can be handed to an agent connected to your base over MCP. Give it this page and the prompt:

```text
You are helping prepare a knowledge base for agents. The base is connected to you over MCP.
Read the guide "Instructions for a knowledge base" (attached) and follow it:

1. Explore the structure of the base with search, expand and note_html. Don't read existing
   notes with mcp_method, if there are any.
2. Propose 20 reader questions of four kinds (direct, situational, detail, questions the base
   doesn't answer). For each, give the answering note and what the answer must contain.
   Verify each answer by opening the note.
3. Answer each question yourself using only the base's tools, and record the path:
   which queries worked, where you took a wrong turn, how many calls it took.
4. List the base's problems (step 2 of the guide) and propose fixes: indexes, maps,
   aliases, titles.
5. Draft an initialize note (up to 1500 characters) and an instructions note
   with routes, working queries, vocabulary and gaps.

Don't change anything in the base yourself: return the questions, the problem report and both drafts.
```

Then check the drafts with the benchmark (step 7) on a weak and a strong model. Instructions written by an agent need measuring too.

## Checklist

- [ ] 15–30 benchmark questions, including 3–5 the base doesn't answer.
- [ ] The benchmark was run without instructions on a weak and a strong model, and the mistakes are written down.
- [ ] The base is fixed where structure fixes the mistake: index, maps, aliases, titles.
- [ ] `initialize`: up to ~1500 characters, entry points, search recipe, answer format, the "if nothing is found after searching" rule.
- [ ] `instructions`: routes, working queries, vocabulary, gaps.
- [ ] A separate `?method=` for weak models or roles, if needed.
- [ ] The benchmark was run with instructions, with no regressions.
- [ ] The benchmark questions sit next to the base, ready for the next check.
