---
title: "Universal tools have no elevator pitch"
free: true
lang_redirect: "[[ru/thoughts/universal-tools]]"
---

*Why Obsidian is so hard to explain in one sentence, why trip2g has the same problem, and why that's a property of the tool, not a failure of marketing.*

Universal tools resist a one-line pitch because every user enters through a different door. Obsidian is the proof: "a markdown editor with wiki-links and plugins" is technically accurate and sells it completely short. The reason is structural. Obsidian is a substrate, not a feature, and substrates get their meaning from what people build on them. trip2g inherits the same substrate shape and the same problem. The honest response is not to search for the one true pitch. It is to accept that there are many valid ones, each true for a different reader.

### Obsidian, reduced

Describe Obsidian mechanically and you get three features: a markdown editor, `[[wiki-links]]` with autocomplete and backlinks, and a plugin API. On paper, a weekend project.

Now list the doors people actually enter through:

- plain note-taking (finally, notes that are just files)
- personal knowledge management
- second brain, Zettelkasten
- a personal or team wiki
- long-form writing, with plugins for citations
- research and literature notes
- task and project management (Tasks, Kanban, Dataview)
- a digital garden, published and tended in public
- engineering notebooks and runbooks
- code snippet libraries
- daily journaling

Each of those is a real community with its own vocabulary and its own YouTube subculture. A Zettelkasten person and a GTD person both live in Obsidian and would give you completely different answers to "what is Obsidian for." Marketing that lands for one bounces off the other: "your second brain" means nothing to someone who wants a runbook wiki, and "local-first task manager" repels the writer crowd.

That multiplicity is the strength. It is also why the elevator pitch never fits. No sentence covers eleven doors without going abstract, and abstract sentences ("a tool for thought") sell nothing to anyone.

The best Obsidian pitch ever written skipped the abstraction entirely. The 2020 Show HN title was "a knowledge base that works on local Markdown files" — over a thousand points. That's one door: data ownership. Not the complete door. The door that fit that particular room.

### Why universality causes this

A feature has a built-in story. A screenshot tool is for taking screenshots; the pitch writes itself. A substrate is a small set of primitives that compose. For Obsidian: plain files, links, extensibility. The primitives are boring to describe, and describing them honestly ("it's markdown files with links") sounds like nothing. The value only appears in the compositions, and the compositions belong to the users, not the vendor.

So the vendor faces a trap with three bad exits. Pitch the primitives and you undersell: a fancy markdown editor. Pitch one composition and you shrink the tool to one door and lose everyone else. Pitch all the compositions and you get a feature-grid landing page nobody reads.

Unix has the same property. "Everything is a file" describes the substrate exactly, means nothing to a newcomer, and every sysadmin, embedded developer, and web host "got" Unix through a different door. Substrates spread bottom-up, door by door, by users showing each other what they built. They are almost never sold top-down with one line.

### trip2g has the same disease

Same substrate shape: markdown files plus a server. Same fan-out. Count the doors, all of them shipped:

- Publish an Obsidian vault as a website. The original door, still the first line of the README.
- A self-hosted MCP memory for AI agents: `search` and `note_html` over your own notes, no vector-store copy of your knowledge.
- A creator platform: paid subscriptions, three zones of access, Telegram publishing.
- A digital garden or self-hosted CMS.
- A team wiki or corporate knowledge base with access control.
- A federation node: one agent query fans out across peer hubs.
- A Markdown Operating System — the substrate described literally, notes as filesystem, MCP as syscalls.

All of it is one substrate: path-addressed markdown notes, one server, one permission model, rendered through different interfaces — web page, MCP call, Telegram post, git remote. The doors are compositions, exactly like Obsidian's.

Which door is "the real trip2g"? Wrong question. Each is the real one for the person standing in front of it. A self-hoster hears "one Go binary on SQLite, your files on your disk" and gets it. An Obsidian user hears "publish your vault, keep your workflow" and gets it. An agent builder hears "MCP memory you can read, diff, and git clone" and gets it. Say any of those sentences to the wrong person and it lands as noise.

### The honest conclusion

You don't explain a universal tool. You pick a door for the reader in front of you and let the substrate reveal the rest after they walk in.

The only sentence that doesn't lie by omission is the substrate one: *your markdown, hosted and connected.* It's true, and it only means something after you've seen at least one door. That's fine. Obsidian never solved this either. It won by keeping the substrate honest — local files, no lock-in — and letting each community write its own pitch.

If you're deciding whether trip2g fits you, don't look for the one-line answer. Look through the door closest to what you already do.
