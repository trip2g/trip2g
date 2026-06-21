---
title: "In-browser file editor"
free: true
lang: en
lang_redirect: "[[ru/user/editor]]"
---

Fix a typo, reword a paragraph, or roll back to yesterday's version — without opening Obsidian.

The editor is available to admins only. Readers never see it.

> 🖼️ **Screenshot (light theme) — `assets/admin/editor-open.png`**
> The editor open on a note. Left: folder-tree file browser. Center: the note's markdown. Top-right: **Save** and **Versions**.

### How to open the editor

The editor icon appears in two places:

- **Top-right of the admin panel** — click it from any admin screen.
- **Next to the search bar on every page** — visible only when you are logged in as admin.

Click the icon. The note for the page you were just viewing opens automatically.

### Browse and switch files

The left side of the editor shows a folder tree of everything in your vault. Click any file to load it into the editor. The tree reflects the same folder structure you see in Obsidian.

### Edit and save

Type directly in the editor. Your changes stay in the browser — nothing is written to the server until you press **Save**. If you close the editor or navigate away before saving, the edits are lost.

After you save, the page reflects the new content immediately. If [[live-editing]] is active on your site, the update appears to readers without a page reload.

**Note:** if you edit a file in the browser and then sync the same note from Obsidian, the Obsidian version overwrites the browser edit. The rule is: the last sync wins.

### Versions panel

Every time you save, trip2g stores a version of the file.

> 🖼️ **Screenshot (light theme) — `assets/admin/editor-versions.png`**
> The versions panel on the right — a list of past saves with dates, one selected, with a **Restore** button.

To go back to an earlier state:

1. Open the editor on the file you want to restore.
2. Click **Versions** (top-right, next to **Save**).
3. Click a version in the list — its content loads into the editor.
4. Click **Save** to make it the current version, or close the panel to discard.

### Ctrl+Click to follow a wikilink

In the editor, hover over any `[[wikilink]]` — the cursor changes to a pointer. Hold **Ctrl** (or **Cmd** on macOS) and click to open the linked note directly in the editor.

This lets you jump through connected notes the same way you would in Obsidian.

### Where the editor icon sits

> 🖼️ **Screenshot — `assets/admin/editor-icon-location.png`**
> The top of a published page with the admin bar visible. The editor icon (pencil) is highlighted next to the search bar. A second instance of the same icon is shown in the admin panel header in a separate inset.

### Related

- [[live-editing]] — browser-pushed updates so readers see changes the moment you save
- [[en/user/publishing]] — frontmatter properties that control visibility and slugs
- [[en/user/two-way-sync]] — how edits made here relate to Obsidian sync
