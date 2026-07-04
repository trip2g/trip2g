# Universal tools and the positioning problem

Date: 2026-07-04. A thought piece, dev-to-dev. Grounded in `README.md`, `docs/en/thoughts/markdown-operating-system.md`, `docs/en/thoughts/for-whom.md`, and the launch material in `docs/dev/2026-07-02_launch_kit.md` / `2026-07-03_hero_demo_v*.md`.

## TL;DR

Universal tools resist a one-line pitch because every user enters through a different door. Obsidian is the proof: "fancy markdown editor with wiki-links and plugins" is technically accurate and sells it completely short, because Obsidian is a substrate, and substrates get their meaning from what people build on them. trip2g has exactly the same problem right now, and it explains why the demo, the landing, and the HN title have gone through so many revisions. The way out is not to find THE pitch. It is to pick one door per audience and let the substrate reveal the rest after they walk in.

## Obsidian, reduced and un-reduced

Describe Obsidian mechanically and you get: a markdown editor, `[[wiki-links]]` with autocomplete and backlinks, and a plugin API. Three features. A weekend project, on paper.

Now list the doors people actually enter through:

- plain note-taking (finally, notes that are just files)
- personal knowledge management
- second brain / Zettelkasten
- a personal or team wiki
- long-form writing (books, papers, with plugins for citations)
- research (literature notes, Zotero integration)
- task and project management (Tasks, Kanban, Dataview)
- a digital garden (publish the vault, tend it in public)
- engineering notebooks and runbooks
- code snippet libraries
- daily journaling

Each of those is a real community with its own vocabulary, its own influencers, its own YouTube subculture. A Zettelkasten person and a Tasks-plugin GTD person both live in Obsidian and would give you completely different answers to "what is Obsidian for." Marketing that lands for one bounces off the other: "your second brain" means nothing to someone who wants a runbook wiki, and "local-first task manager" actively repels the writer crowd.

That multiplicity is the strength. It is also why the elevator pitch never fits. There is no sentence that covers eleven doors without going abstract, and abstract sentences ("a tool for thought") don't sell anything to anyone.

Notably, the highest-scoring Obsidian pitch ever written skipped the abstraction entirely. The 2020 Show HN title was "a knowledge base that works on local Markdown files" (1087 points, see the launch-kit research). That's one door: data ownership. Not the best door, not the complete door. The door that fits HN.

## Why universality causes this

The mechanism is simple: a universal tool is a substrate, not a feature.

A feature has a built-in story. A screenshot tool is for taking screenshots; the pitch writes itself. A substrate is a small set of primitives that compose: for Obsidian, plain files + links + extensibility. The primitives themselves are boring to describe, and honestly describing them ("it's markdown files with links") sounds like nothing. The value only appears in the compositions, and the compositions belong to the users, not the vendor.

So the vendor faces a structural trap. Pitch the primitives and you undersell (a fancy markdown editor). Pitch one composition and you shrink the tool to one door and lose everyone else. Pitch all the compositions and you produce a feature-grid landing page nobody reads. Unix has the same property: "everything is a file" describes the substrate exactly, means nothing to a newcomer, and every sysadmin, embedded dev, and web host "gets" Unix through a different door.

Substrates get evangelized bottom-up, door by door, by users showing each other what they built. They are almost never sold top-down with one line.

## trip2g has the same disease

Same substrate shape: markdown files + a server. Same fan-out. The repo itself documents at least eight doors, none of them invented:

- **Publish an Obsidian vault as a website.** The original one, and still line one of the README.
- **Self-hosted MCP memory for AI agents.** Line one of the README, second half. `search` / `note_html` / `similar` over your own notes, no vector-store copy.
- **Obsidian → website with subscriptions + Telegram.** The CLAUDE.md description of the project; the creator-economy door (`docs/en/thoughts/three-zones.md`, `for-whom.md`).
- **Markdown Operating System.** The README's own reframing: notes as filesystem, MCP as syscalls, federation as network, agents as processes (`markdown-operating-system.md`).
- **Telegram publisher.** For some users the website is incidental and the channel is the product.
- **Paywalled notes / sell your second brain.** Subgraphs, subscription ACLs, the whole `for-whom.md` cast: YouTuber, course creator, consultant.
- **Self-hosted wiki / digital garden / CMS.** The `vs-quartz`, `vs-notion`, `vs-gitbook` comparison pages exist because people arrive asking exactly those questions.
- **Federation: a mesh of knowledge bases.** One query fans out across peer hubs (`2026-07-02_murmur_vs_trip2g.md` treats this as a substrate in its own right).

And `for-whom.md` fans each of those out again by persona: community wiki, corporate KB with access control, expert consultant with an AI answering routine questions.

This is not scope creep. All of it is one substrate (path-addressed markdown notes, one server, one permission model) rendered through different interfaces: web page, MCP call, Telegram post, git remote. The doors are compositions, exactly like Obsidian's.

And it explains the current struggle precisely. The hero demo has five scenario drafts (`2026-07-03_hero_demo_v1..v4` plus optB). The Show HN title has three A/B options in the launch kit. The launch kit's own research warns against putting "Markdown OS" in the title because "on HN unexplained metaphors read as marketing." None of this is indecision about what trip2g is. It is the structural impossibility of covering eight doors in one line, rediscovered per artifact.

## So what

Stop treating the one-liner as a search problem with a right answer. There isn't one. Instead:

**One door per audience, chosen deliberately.**

- Show HN: the data-ownership + agent door. "Self-hosted server that turns an Obsidian vault into a website and MCP server" (already option A in the launch kit). Nouns, one sentence, verifiable.
- Self-hosters (r/selfhosted, the fly.io/docker docs): one Go binary, SQLite, your files on your disk. The Plausible shape.
- Obsidian community: publish your vault, keep your workflow, wikilinks resolve. Never lead with agents here.
- Agent/MCP crowd: MCP memory you can read, diff, and `git clone`. Lead with the token-economy benchmark, never with "website."
- Creators: three zones, subscriptions, Telegram. The `for-whom.md` pitch.

**Keep "Markdown OS" as the internal map, not the external pitch.** It is the true description of the substrate, which is exactly why it doesn't sell: substrate descriptions never do. It earns its keep as the document that shows a curious visitor, after they've entered through some door, that the other doors exist and are load-bearing.

**The honest one-liner is the substrate one, used sparingly.** Something like "your markdown, hosted and connected" is the only sentence that doesn't lie by omission. It belongs on the site as a masthead, not in a launch title, because it only means something after you've seen a door.

**Judge each artifact by its door, not by coverage.** The recurring failure mode in the demo/landing drafts is trying to show website + MCP + Telegram + subscriptions in one 15-second GIF. A GIF is one door. Pick it, and put a visible hallway ("also: MCP, Telegram, subscriptions") behind it.

The reassuring part: Obsidian never solved this either. It won by letting each community write its own pitch. The vendor's job was to keep the substrate honest (local files, no lock-in) and stay out of the way. That is an available strategy.

## Public version?

Worth adapting, yes, but not as-is. A public Philosophy piece (`docs/{en,ru}/thoughts/`) would keep the Obsidian analysis and the substrate mechanism, drop the launch-kit internals and door-per-channel tactics (those read as "here is how we market to you" when the reader is the target), and end on the substrate one-liner instead. Natural neighbors: `obsidian-and-wikilinks.md` and `markdown-operating-system.md`. Deferred; do it after launch positioning settles, since the public piece should name the door we actually chose.
