---
title: Content Infrastructure for Human-AI Workflows
slug: content-infrastructure
order: 11
layout: trip2g/content-first
free: true
lang_redirect: "[[ru/thoughts/content-infrastructure]]"
---

trip2g is content infrastructure. People and AI agents work with the same content on equal footing. Content doesn't sit idle — it flows between participants and transforms at every step.

Not a CMS where you "create and publish." Not a wiki where only humans edit. Not a static site generator where you "build and deploy." This is a **programmable platform** where content flows between people, bots, AI agents, Telegram, and [[en/thoughts/obsidian-and-wikilinks|Obsidian]] — and changes at every transition.

---

## Why "Content Infrastructure"

**Infrastructure** — because this is not an end product, but a foundation. An electrical grid is infrastructure for homes. A road network is infrastructure for transport. trip2g is infrastructure for content. On top of it you build specific things: blogs, knowledge bases, AI pipelines, inbox systems.

**Content** — because the unit of work is a markdown note. Everything else (versions, metadata, webhooks) revolves around notes. Markdown is the universal interface: a person writes in [[en/thoughts/obsidian-and-wikilinks|Obsidian]], an AI agent generates via API, a bot pushes from Telegram.

**Human-AI workflows** — because [[en/thoughts/ai-changes-rules|AI is not a tool, it's a participant]]. An AI agent can do everything a human can: read notes, change notes, react to changes. The difference is that AI only sees what it's been permitted to see, and it can't get stuck in a loop. More on that below.

---

## Content as a Living Flow

The core metaphor: content doesn't sit still — it flows. Every note is a point in the stream through which transformations pass.

Imagine a chain:

```
A person writes a note in Obsidian
  -> sync -> trip2g saves a version
    -> webhook "note changed" -> AI linter checks grammar
      -> AI pushes corrections -> new version
        -> sync -> Obsidian shows the changes
          -> person reviews and accepts
            -> rules automatically add metadata
              -> template renders HTML
                -> reader sees the page
```

Every step is a transformation. Every participant (person, bot, AI, template engine) is a link in the chain. trip2g is the pipe the content flows through.

---

## Key Properties

### Markdown — The Universal Interface

One note. One format. Everyone works with it:
- A person — through [[en/thoughts/obsidian-and-wikilinks|Obsidian]], a browser editor, Telegram
- An AI agent — through the platform API or webhooks
- The system — through metadata rules, templates, [[en/thoughts/knowledge-graph|links between notes]]

Markdown wasn't chosen because it's the best format. It was chosen because it's **the only format that's equally readable by both humans and machines**. JSON is great for machines; visual editors are great for people. Markdown is the compromise that works for both.

### Versioning Everything

Every change is a new version. The model is like git: save it — it's committed. This gives three things:

- **History.** What changed, when, by whom — human or AI. You always know who the author is.
- **Rollback.** AI generated garbage — roll back to the previous version with one command.
- **Safety with concurrent edits.** If two agents edit the same note at the same time, nothing breaks — the system figures out who was first and doesn't overwrite anyone's changes.

### Webhooks — The Notification System

Webhooks are how trip2g tells the outside world "something happened":

- **Change webhook.** "A note changed" — react. A linter checks grammar, an indexer updates search, an AI reviewer suggests edits.
- **Scheduled webhook.** "Time's up" — act. Weekly digest, inbox cleanup, content generation.

An agent can return changes directly in the webhook response (synchronously) or work through the API (asynchronously). The system applies the changes — and they may trigger the next webhooks. The result is a transformation pipeline.

### Depth — Safe AI Chains

The main problem with AI agents: infinite loops. Agent A changes a note. This triggers agent B. Agent B also changes the note. This triggers agent A again. And so on forever.

trip2g solves this with a depth counter:

- A person edits a note (depth 0) — all webhooks fire
- An AI agent edits (depth 1) — only webhooks that allow depth 2+ fire
- A second AI agent (depth 2) — webhooks restricted to depth 2 no longer fire

Agent chains work. Loops are blocked. This is configured per webhook — one can run at any depth, another only on manual edits.

### Restricted Access for Agents

Each AI agent gets exactly as much access as it needs. No more.

- **What it can read:** for example, only notes in the `blog/` folder
- **What it can change:** for example, only notes in that same folder
- **Token lifetime:** 60 minutes by default, then access expires

A blog linter can't see private notes and can't change the configuration. This is the principle of least privilege — like [[en/thoughts/three-zones|three access zones]], but for agents.

### Obsidian — Humans Stay in Control of AI

[[en/thoughts/obsidian-and-wikilinks|Obsidian]] syncs with trip2g. When an AI agent changes a note, here's what happens:

1. AI makes changes via webhook or API
2. Sync delivers the changes to your Obsidian vault
3. You see exactly what changed, in a familiar editor
4. You accept, edit, or roll back

Obsidian becomes a **moderation screen for AI-generated content**. You don't lose control — you always see what AI did and decide whether to accept it. AI proposes, the human decides.

### Metadata Rules — [[en/thoughts/declarativity|Declarative]] Transformations

Simple path-based rules. Automatically add or change note metadata. Applied by priority, from general to specific:

```
Rule: blog/*     -> { free: true }
Rule: premium/** -> { free: false }
Rule: *          -> append " — Site" to title
```

Describe the rules once — the system applies them to all notes automatically. No code to write, no manual work. [[en/thoughts/declarativity|Declarativity]] in its purest form: describe what you want — get the result.

### Atomic Content Operations

The operation "find a string in a note, replace it with another" is atomic. No conflicts during concurrent editing — everything happens in one transaction.

This is the foundation of the "marker-anchor" pattern. You place a marker in the note — any string. Agents insert content at that position. The marker stays, so the next insertion goes after the previous one:

```
find:    "$INBOX$"
replace: "## New message\nText\n\n$INBOX$"
```

Content accumulates. The agent doesn't know the full contents of the note — it only knows the marker and what to insert. A simple, reliable way for AI to add content in the right place.

### Flexible Template Engine — Content as Building Blocks

The trip2g template engine doesn't see plain text — it sees document structure. It parses markdown into logical blocks: introduction, headings, sections under each heading. This is called AST access — access to the document's abstract syntax tree.

What this means in practice. You write ordinary markdown:

```markdown
Frequently asked questions about the service.

### How do I get started?

Register and create your first project.

### How much does it cost?

The basic plan is free.

### Is there an API?

Yes, documentation is on the site.
```

And the template turns third-level headings into cards, and the first paragraph (before the headings) into an intro:

```jet
{{ intro := note.PartialRenderer().Introduce() }}
<p class="lead">{{ intro.ContentHTML | unsafe }}</p>

<div class="cards">
  {{ range i, s := note.PartialRenderer().Sections(3) }}
    <div class="card">
      <h3>{{ s.TitleHTML | unsafe }}</h3>
      {{ s.ContentHTML | unsafe }}
    </div>
  {{ end }}
</div>
```

The author writes plain text. The template decides how to display it: as cards, accordions, slides, FAQ. The same markdown can look completely different on different pages — it depends on the template, not the content.

The template can also reach out to other notes — pull navigation from a sidebar, grab the latest blog post, show backlinks. The content stays clean, and all display logic lives in the template. This is [[en/thoughts/declarativity|declarativity]]: the author describes what, the template decides how.

More in the [[en/user/templates|template documentation]].

### MCP — Knowledge as a Service

MCP (Model Context Protocol) turns a knowledge base into a consultant you plug into your AI client. Claude Desktop, Cursor, Claude Code — any client with MCP support gets direct access to someone else's expertise.

An author publishes notes through trip2g and writes instructions for AI: how to use the base, what tone to respond in, what questions to answer. A reader connects the MCP server and asks their AI — which answers from that specific expert's knowledge, not from its general training.

Available methods:

```
search(query)        — full-text search across the base
note_html(slug)      — retrieve a specific note
similar(slug)        — similar notes
instructions()       — author's instructions for AI
```

Access is configurable: open or paid by subscription. The access model is the same as the [[en/thoughts/three-zones|three zones]]: what's publicly open, what's behind a subscription, what's only for the author.

Privacy works both ways. The MCP server returns only text from the author's base — it cannot see the reader's chat context. The author doesn't know exactly what people are asking.

This closes the loop: the author writes → AI reads → the reader asks → AI answers from the author's knowledge. Knowledge flows from expert to reader through AI, with no intermediate steps.

### Multi-Domain — One Platform, Many Sites

A single trip2g instance serves multiple independent domains. Just add to a note's frontmatter:

```yaml
route: portfolio.com/
```

— and the note appears on that domain instead of the main one.

Domains are isolated: content from portfolio.com doesn't mix with mysite.com. Each domain has its own sitemap, its own templates, its own settings. For search engines and visitors, these are entirely separate sites.

A practical scenario: a freelancer runs portfolio.com, a blog at blog.com, and a landing page for a client at client-landing.com — all from a single trip2g installation, one set of notes, one interface. Adding a new domain means adding one field to the frontmatter. More in [[en/user/advanced|custom domains]].

This follows the same "infrastructure" logic: the way one server rack hosts dozens of websites, one trip2g installation hosts dozens of domains.

### One File — The Whole System

One binary. The entire database — one file. No dependencies. Automatic cloud backup for reliability.

```
Typical system:           trip2g:
Application server        One program file
Database                    with everything inside
Cache server                  including the database
Message queue
```

Deploy to a server — one minute. The data is yours. Backup — copy one file. Move to another server — move that file. This is [[en/thoughts/digital-sovereignty|digital sovereignty]] at the infrastructure level.

---

## Who This Is For

### Content Creators Working With AI

A blogger wants AI to check grammar, generate summaries, suggest tags — but still stay in control. They write in Obsidian, AI works through webhooks, the result appears in the same vault. Accepted — it's published. Rejected — rolled back.

### Teams With a Knowledge Base

An internal wiki where AI agents automatically update tables of contents, generate cross-references, and check for outdated content. People moderate through Obsidian or the web interface. The [[en/thoughts/knowledge-graph|knowledge graph]] grows on its own, but under human supervision.

### AI-First Publishing

A Telegram bot collects notes into an inbox. An agent runs on a schedule and generates a weekly digest. Another agent formats it into HTML. A third publishes it to a Telegram channel. The person configures the pipeline and moderates the result.

### AI Agent Developers

trip2g as a backend for AI agents: storage, versioning, webhooks, restricted access. An agent connects via webhook or API, works with content, and doesn't have to think about storage or concurrency conflicts.

More on use cases — [[en/thoughts/for-whom|who this is for]].

---

## What This Is Like

Looking for analogies from familiar territory:

- **Obsidian** — working with markdown and connections between notes
- **GitHub Actions** — an event happens, an agent reacts, makes a change, triggers the next event
- **Vercel** — one deploy, everything works, minimal configuration

But there's no perfect analogy. The closest description: **a programmable content platform where AI agents are participants on equal footing with humans**.

This site is a [[en/thoughts/dogfooding|live example]]. It's built on trip2g and uses everything described above.
