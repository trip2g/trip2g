---
title: Live editing
free: true
lang: en
lang_redirect: "[[ru/user/live-editing]]"
---

The moment you sync a note, the published page updates. No cache to clear, no full reload — only the parts of the page that changed repaint.

### How it works

When the sync plugin sends a note to the server, the server emits a `noteChanges` event over a live connection (SSE). Every browser that has that page open receives the event instantly. The event carries `changedHtmlSelectors` — the specific CSS selectors whose rendered HTML changed. The browser patches only those parts of the DOM. The rest of the page stays untouched.

This means a heading edit repaints the heading. Adding a paragraph repaints the content area. The reader sees the change without a flash, without losing scroll position, without a network round-trip for assets that did not change.

### Edit in Obsidian, watch the browser update

Open your published site in a browser window. Open the same note in Obsidian. Edit the note, then hit sync in the plugin.

> 🖼️ **Screenshot — `assets/admin/live-editing-side-by-side.png`**
> Obsidian and a browser window open side by side. On the left: the note being edited in Obsidian with a visible change (e.g. a heading being retyped). On the right: the published page in the browser showing the updated content already reflected. Highlight that no reload indicator (spinner, flicker) is visible in the browser tab.

The update arrives in under a second on a normal connection. The latency is the sync time itself — the browser receives the event the moment the server commits the note.

### What counts as "changed"

The server compares the rendered HTML of the updated note against what it had before. Anything that changes the visible output counts: text edits, formatting, frontmatter fields that affect the template, new or removed wikilinks. Changes that do not affect the rendered output (whitespace, frontmatter fields not used by the template) produce no DOM update.

### No configuration needed

Live updates are active for every published page automatically. There is no toggle to enable in the admin panel and no frontmatter flag to set. As long as the browser has the page open and the note is published with `free: true` (or the reader is authenticated), updates arrive live.

### The in-browser editor

Admins who use the in-browser editor — the editor icon in the admin panel — also see live updates. If an AI agent or a colleague edits a note through the API while you have the admin editor open, the editor content refreshes automatically.

### Related

- [[en/user/two-way-sync|Two-way sync]] — receive server-side changes back into Obsidian automatically
- [[en/user/publishing|Publishing notes]] — how notes reach the server
- [[en/user/webhooks|Webhooks & automation]] — `noteChanges` as a webhook trigger for custom integrations
