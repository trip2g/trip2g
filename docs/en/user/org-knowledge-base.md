---
title: "An organisation's knowledge base: personal bases and keepers"
description: "Lay out company knowledge in trip2g so that nobody writes to a shared place: everyone writes to their own base, a keeper agent builds the level above, and every base has one writer."
free: true
lang_redirect: "[[ru/user/org-knowledge-base]]"
---

This page describes one way to lay out the knowledge of a team or a company in trip2g: nobody writes to a shared place. Everyone writes to their own base, a keeper agent builds the level above, and every base at every level has one writer. After reading it you can decide how many bases you need and what each agent may read. It is for whoever rolls trip2g out to more than one person. If you run a single base alone, [[en/user/team-knowledge-base-mcp|a team knowledge base for agents]] is enough.

In this article:

- [Start from one question: who writes](#start-from-one-question-who-writes)
- [The keeper](#the-keeper)
- [What holds whatever the delivery](#what-holds-whatever-the-delivery)
- [What the keeper sees in a personal base](#what-the-keeper-sees-in-a-personal-base)
- [How personal agents read the organisation base](#how-personal-agents-read-the-organisation-base)
- [Delivery is yours to choose](#delivery-is-yours-to-choose)
- [What this costs](#what-this-costs)

## Start from one question: who writes

A domain here is a boundary of authorship. It is not a topic and not a department. Count the writers and you have counted the domains:

| Who writes | Layout |
|---|---|
| One writer | One base. Subgraphs split the readers |
| Several writers | One base per writer, joined by federation |

The two tools answer different questions. Federation answers "who writes": separate bases, each with its own owner, joined by edges. Subgraphs answer "who reads": the writer is the same, the readers differ. The mechanics of both are in [[en/user/context-separation|Context separation]], [[en/user/federation|MCP Federation]] and [[en/user/subgraphs|Subgraphs]].

How many writers an organisation has is decided by the people who roll it out and by the heads of its teams. One keeper for the whole company and a keeper per team are both valid layouts.

trip2g does not enforce "one writer". It is a rule you keep by handing out one write credential per base. An API key writes only within its write patterns, so a keeper's key can be limited to the folders it builds. See [[en/user/update_notes|updateNotes]] for how write access is checked.

## The keeper

A keeper is an agent that is the only writer of an assembled base. People write to their own bases. The keeper reads those and outside sources, then writes the level above: summaries, indexes, a glossary, a handbook.

The public worked example is [[en/user/calls-knowledge-base|Calls into a knowledge base]]. Call recordings arrive as transcript notes. [[en/user/fleet|Fleet]] roles then write call notes, concept notes, daily notes and topic logs.

A function such as HR does not need its own agent. It is a section of a keeper's base, opened to the readers who should see it. A person's own agent reads it with that person's access.

## What holds whatever the delivery

Three things stay true however knowledge reaches the keeper:

1. **Only the keeper writes to the assembled base.** People do not edit it.
2. **An assembled note shows its source and the source's owner.** In the calls pipeline every call note links back to the raw transcript, and every quote carries a time range.
3. **A correction is made in the source.** The keeper then rebuilds. A fix made by hand in the assembled note can be overwritten on the next run, and it makes the assembled base a second place people write to.

## What the keeper sees in a personal base

The owner of a personal base opens a part of it, not all of it. Three credentials exist today, and they differ in how much they open:

| Credential | What it reads |
|---|---|
| Federation key | Notes in the subgraphs the owner granted to that key |
| Personal access token | What its user may read: the subgraphs granted to that user |
| API key | Every note in the base |

A new inbound federation key starts with access to nothing. The owner picks the subgraphs for it in the admin panel, and each direction is granted separately.

Two details decide what "a part" means in practice:

- **A note with no subgraph counts as shared.** Once a federation key or a user holds at least one granted subgraph, it also reads every note that carries no `subgraph:` label. In a personal base label what is private, not what is shared.
- **An API key is not a way to open a part.** Over MCP it reads every note, the same as a site admin. Give a keeper a federation key or a user account with a grant instead.

## How personal agents read the organisation base

A personal agent connects to the organisation base over [[en/user/mcp|MCP]] and reads with its owner's access. Two set-ups exist:

- **A personal access token.** The owner has an account on the organisation base. The token inherits that account's subgraph grants and adds nothing to them.
- **An edge from the owner's own hub.** The agent talks to its owner's base. That base holds a federation key to the organisation base and reaches it through `federated_search` and `federated_note_html`.

The second set-up keeps one MCP connection per agent however many bases stand behind it. [[en/user/federation|MCP Federation]] explains why that matters.

## Delivery is yours to choose

How knowledge gets from a personal base or an outside source to the keeper is not prescribed. Some ways that work today, and the list is not complete:

- **A fleet role on change.** A note changes in the keeper's hub and a role runs. See [[en/user/fleet|Fleet]].
- **A fleet role on a schedule.** A code role pulls an outside API and writes notes. The calls pipeline ingests transcripts this way.
- **By hand.** A person moves a note into the keeper's base.
- **An outside agent.** An agent such as Claude Code reads one base over MCP and writes to another through [[en/user/update_notes|updateNotes]] or the [[en/user/cli|sync CLI]].

One limit to plan around: a fleet role reads and writes the notes of the hub its fleet is connected to. An LLM role has no tool that searches a federated peer. Notes from another base reach a fleet keeper by one of the other ways, or a fleet runs next to that base.

## What this costs

- **The assembled level lags its sources.** It is as fresh as the keeper's last run.
- **A personal base holds personal things.** The boundary is set on purpose, note by note or by folder. An unlabelled note is readable through any granted key.
- **The hub records who wrote, not where the knowledge came from.** Each note version stores the user, the API key and the client that saved it, and a version written by a fleet run is tied to the delivery that produced it. A layout can print the author of the current version with `LastEditedBy()`, which is admin-only data. Nothing records which source note an assembled note was built from. The keeper's role has to write that link itself.
- **A correction takes two steps.** The reader tells the owner of the source, and the owner fixes it. Nobody edits the note in front of them.

## Related

- [[en/user/context-separation|Context separation]]: edges and subgraphs on a ten-base example
- [[en/user/federation|MCP Federation]]: keys, KB-notes and scopes
- [[en/user/subgraphs|Subgraphs]]: who reads a note
- [[en/user/fleet|Fleet]]: agents described as notes
- [[en/user/calls-knowledge-base|Calls into a knowledge base]]: a keeper built from fleet roles
