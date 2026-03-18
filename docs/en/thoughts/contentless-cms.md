---
title: "Contentless CMS"
order: 7
free: true
lang_redirect: "[[ru/thoughts/contentless-cms]]"
---

A CMS with no content inside. It sounds like a paradox, but that's the whole point.

### Headless is about the frontend

Headless CMS removed the frontend. Content lives in the system and is served via API. The site is built separately.

But the content is still inside. WordPress, Strapi, Contentful — they all have an editor. You write in their interface, your content lives in their database.

### Contentless is about the content

Contentless CMS goes further. Content is not stored in the system at all. It comes from outside — from your files.

You write in any editor: Obsidian, VS Code, Cursor, vim. The files live on your computer. The platform publishes them, but doesn't own them.

### The difference by example

**Headless CMS:**
1. Open the admin panel
2. Write in their editor
3. Content is saved in their database
4. API delivers it to your frontend

**Contentless CMS:**
1. Write in your own editor
2. Files stay on your computer
3. Sync with the platform
4. Platform publishes

### Why this matters

**Editor is your choice.** Obsidian for linking notes together. VS Code for code and text side by side. Vim if that's what you're used to. The platform doesn't dictate the tool.

**Content is yours forever.** Service shut down? Files are still there. Got banned? You move with your full history. Found something better? Take everything with you. That's [[en/thoughts/digital-sovereignty|digital sovereignty]].

**Offline by default.** No internet — keep writing. Sync whenever it's convenient.

### An analogy

Git for code works exactly the same way. Code lives locally, GitHub just hosts it. Nobody writes code in the GitHub web interface.

Contentless CMS is the Git approach for content. Local files, external publishing.

### How this works in trip2g

**[[en/thoughts/admin-panel|Two editors]].** The admin panel is for configuration (tokens, domains, publishing rules). Your editor is for content. Set it up once, then just write.

**Declarativity.** Write a note, sync it — it's published. You don't think about how, you describe what.

**Any editor.** Obsidian for linking notes. Cursor for working with AI. VS Code, vim — whatever you like. The platform takes the files, it doesn't impose the tool.
