---
home_position: 75
title: Two-Way Sync
free: true
lang_redirect: "[[ru/user/Двухсторонняя синхронизация]]"
---

By default the plugin syncs one way: you write in Obsidian, changes go to the server. The server never modifies your files.

Two-way sync sends changes in both directions. If a file is updated on the server, the plugin downloads it to your vault automatically.

### When you need it

**Team editing** — multiple people edit the same knowledge base. Everyone sees each other's changes in real time.

**Server-side automation** — Telegram post imports, content generation, data processing. Results appear directly in your vault.

### Important

Two-way sync can modify your local files. The plugin tracks changes and resolves simple conflicts automatically.

If you and a colleague edit the same file simultaneously, the plugin shows a prompt:
- Keep your version
- Accept the server version
- Save both versions

Complex conflicts can be subtle. Set up backups before enabling.

### How to enable

1. Open trip2g sync plugin settings
2. Enable **Two-way sync**
3. Run sync — the plugin checks the server state

On the first sync, the plugin asks which version to treat as the source of truth: local or server.

### Live updates (real-time)

With two-way sync on, you can also turn on **live updates**: server changes appear in your vault the instant they happen — no sync click, no waiting for the periodic check. The plugin keeps a live connection and pulls changes as they land.

**Enable it:** with two-way sync on, set **Live pull patterns** in the plugin settings — include/exclude globs that choose which folders update live:

- `**` — everything
- `blog/**`, `docs/**` — only these folders
- exclude `drafts/**` — never auto-pull drafts
- leave empty — live updates off

**It stays safe:**

- Only non-conflicting changes apply automatically. If you edited a file locally, a server change never overwrites it — you get a conflict prompt.
- Server-side deletions don't silently remove your local file — the plugin asks first.

A background reconciliation (every few minutes) and manual sync still catch anything missed — for example, changes that happened while you were offline.

### Status indicator

A colored dot on the sync icon shows the current state:

| Color | Meaning |
|-------|---------|
| 🔵 Blue | Server has changes |
| 🟢 Green | Local changes pending |
| 🟠 Orange | Both sides have changes |
| 🔴 Red | Conflicts detected |

```mermaid
stateDiagram-v2
    [*] --> Synced
    Synced --> ServerChanges: server edited a file
    Synced --> LocalChanges: you edited a file
    ServerChanges --> BothChanged: you also edit
    LocalChanges --> BothChanged: server also edits
    BothChanged --> Conflict: same file, both sides
    ServerChanges --> Synced: sync downloads (blue)
    LocalChanges --> Synced: sync uploads (green)
    BothChanged --> Synced: auto-merge (orange)
    Conflict --> Synced: resolve - keep / accept / both (red)
```
