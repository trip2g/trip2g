---
title: "Canvas, Base & Excalidraw"
free: true
lang: en
lang_redirect: "[[ru/user/canvas]]"
---

Obsidian's `.canvas`, `.base`, and `.excalidraw` files all sync to your site without errors. On the web, none of them render yet — each page shows a short "not supported yet" placeholder instead of the visual board. Full web rendering is planned.

## What works today

- **Sync never breaks.** Put `.canvas`, `.base`, or `.excalidraw` files in your vault and sync — they upload cleanly and stay reachable. They don't produce broken pages or errors.
- **Canvas drives Telegram navigation.** A `.canvas` file can power a Telegram navigation bot: a reader moves through your notes by following the canvas's nodes and connections, right inside Telegram. See [[telegram]].

## On the web: a placeholder, for now

Open a `.canvas`, `.base`, or `.excalidraw` page on the live site and you'll see a line like "Canvas files are not supported yet." in place of the board. The file is published and linkable — it just isn't drawn yet.

So if your vault contains these files, publishing won't break. Readers see a clear message rather than the diagram.

## What's planned

Web rendering for all three formats — drawing the actual board, sketch, or table — is on the [[roadmap]]. If you need it sooner, leave a request on [[version_requests]].

## Related

- [[roadmap]] — what works now and what's planned next
- [[telegram]] — canvas-driven navigation bots
- [[version_requests]] — influence what ships next
