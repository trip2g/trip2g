---
title: "Browser Sync"
free: true
lang_redirect: "[[ru/user/Browser Sync]]"
---

Browser Sync publishes notes from a local folder to your trip2g site directly from the browser — no plugin, no CLI, no API key required. Open the admin page, pick a folder, and upload.

### How to use

1. Open Admin → System → Browser Sync.
2. Click **Select folder**, pick your notes folder, and allow access when the browser asks.
3. The page shows "Will upload N file(s)" and lists the files — this is a preview; nothing is sent yet.
4. Click **Upload** to publish those files to the site.

That's it. The files appear on your site immediately.

### Keeping the site in sync

After you edit files on disk, click **Refresh** to re-check the folder and see the updated file count. Browser Sync does not watch the filesystem automatically.

For hands-off operation, check **Auto-sync (every 30s)**. The page re-checks the folder and uploads any changes every 30 seconds as long as the browser tab stays open.

The selected folder is remembered in your browser and restored on the next reload — you don't need to re-pick it. This is per-browser and per-device.

Click **Clear folder** to forget the selection.

### Authentication

Browser Sync uses your existing admin session. There is nothing to configure.

### Limitations

**Upload only.** Browser Sync is one-way: local folder → site. It does not pull changes from the site back into your folder. For bidirectional sync — or if you want sync to run in the background without a browser tab open — use the [[en/user/two-way-sync|trip2g sync Obsidian plugin]].

**Desktop Chrome and Edge only.** Browser Sync uses the browser File System Access API, which is not available in Firefox, Safari, or any mobile browser.

**Not a background service.** Auto-sync runs only while the admin tab is open. Browsers throttle background tabs, so closing or minimizing the window will interrupt it. For continuous, unattended sync, use the Obsidian plugin or the sync CLI.

### When to use Browser Sync

Browser Sync is the fastest way to try trip2g without installing anything. Select a folder and your notes are live in seconds.

For day-to-day publishing, the [[en/user/two-way-sync|trip2g sync Obsidian plugin]] is the full-featured alternative: it runs in the background, handles two-way sync, and does not require a browser tab to stay open.

### Related

- [[en/user/two-way-sync|Two-way sync]] — bidirectional sync and background publishing via the Obsidian plugin
- [[en/user/getting-started|Getting started]] — install the Obsidian plugin and publish your first note
