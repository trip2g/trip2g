---
title: "Obsidian and Note Connections"
order: 8
free: true
lang_redirect: "[[ru/thoughts/obsidian-and-wikilinks]]"
---

Why we chose Obsidian as the foundation for trip2g. What Markdown and wikilinks are, and why they change how we think about knowledge.

### Markdown — Text That Outlives Any Service

Markdown is a way to format text using simple symbols. `#` for headings, `**` for bold, `-` for lists. Files use the `.md` extension.

Why this matters:

Your notes are plain text files. They'll open in any editor in 10, 20, or 50 years. Notion might shut down. Evernote might shut down. A text file will always work.

You don't depend on any company, server, or subscription. Files on your computer are your property.

More on syntax: [Markdown: the basics](/docs/markdown)

### Obsidian — An Editor That Understands Connections

Obsidian is a free app for working with Markdown files. But its defining feature isn't editing — it's the connections between notes.

An ordinary editor sees files in isolation. Obsidian sees a network: how notes link to each other, which topics are related, where gaps in your knowledge exist.

It works with a folder on your computer. Nothing is stored in the cloud unless you set up sync yourself.

### Wikilinks — Links With Double Brackets

In Obsidian, a link to another note looks like this:

```
See also [[Another Note]]
```

Two square brackets — and the connection is made.

**Autocomplete.** Type `[[` and a list of all your notes appears. Start typing a name and the list filters. No need to remember exact titles or hunt through folders.

**Create on the fly.** Linking to a note that doesn't exist yet? No problem. Obsidian will create it when you follow the link. You can sketch out the structure first and fill in the content later.

**Backlinks.** Open a note and see who links to it. Context you forgot about. Connections you didn't plan.

### The Speed of Building Knowledge

With wikilinks you can create hundreds of connected notes in an hour.

You're writing about investing — you mention diversification. You don't stop, you keep the thought going. Later you'll come back and expand on the term. Or you won't — the link will still show that the topic is connected.

Without wikilinks you'd have to: create a file, think of a name, save it, return to the original note, insert the link, verify the path is correct. One minute per connection. A hundred connections — two hours of mechanical work.

With wikilinks, a hundred connections takes ten minutes.

### From Local Notes to Public Knowledge

You've built a knowledge base in Obsidian. Hundreds of notes, thousands of connections. Now you want to share it.

trip2g publishes your notes in one click. Wikilinks become working links — on your site and in Telegram.

Some notes are private (just for you). Some are public (visible to the whole internet). Some are paid (visible to subscribers).

Connections are preserved at every level.

### The Community

Obsidian is not a niche tool. Millions of users, hundreds of plugins, active forums.

Plugins extend the capabilities: calendars, kanban boards, AI integration, sync with any service. If Obsidian can't do something out of the box, there's a plugin for it.

The community answers questions, shares templates, and writes tutorials. It's easy to get started.

### Not Just Obsidian

trip2g works with any Markdown files. Obsidian is convenient, but not required.

Some people use **Cursor** — an editor with built-in AI. It's handy when you want artificial intelligence to help develop your knowledge base: analyzing what you've written, suggesting connections, expanding notes.

We call this [[en/thoughts/contentless-cms|Contentless CMS]] — the platform doesn't impose an editor, it works with any Markdown files.

### How Text Becomes a Website

The trip2g homepage is also Markdown files. Headings, descriptions, FAQ — all stored as notes.

Want to change the text on the homepage? Open the file, edit it, publish. No need to touch code or call a developer.

The template handles appearance: where each block goes, what colors, what spacing. The content handles meaning: what's written in the heading, what questions are in the FAQ.

A clean separation: the designer builds the template once, the author changes the text as many times as they want.

More on how we built the homepage: How we built the homepage from markdown blocks

### Summary

Markdown is a format that will always work. Obsidian is a tool that sees the connections between knowledge. Wikilinks are the way to create those connections instantly.

Together they let you build a knowledge base at any pace without losing structure. And trip2g turns that base into a public product.
