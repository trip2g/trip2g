---
title: Backups
free: true
lang: en
lang_redirect: "[[ru/user/Бекапы]]"
---

Obsidian stores your notes locally. Local files can be lost — a failed drive, an accidental delete, a corrupted sync. Set up a backup now, before it happens.

### Sync is not a backup

Obsidian Sync, iCloud, OneDrive, and Dropbox are sync tools, not backup tools. If you delete a file, it disappears on every synced device within seconds.

A backup is a separate copy that does not change when the original changes. If you delete a file from your vault, the backup still has it.

### Plugins for backups

**Obsidian Git**

Saves your vault to a Git repository. Every change becomes a commit. You can roll back to any previous version of any file.

**Local Backup**

Copies your vault to a folder you choose. You can configure an archive schedule and keep multiple versions.

### Other approaches

**Cloud storage:** Dropbox, Google Drive, and Backblaze can hold a copy of your vault folder. Schedule regular uploads or use their desktop clients to sync a backup folder.

**External drives:** Copy your vault folder to a USB drive or external SSD. Simple and offline.

**System tools:**

- Windows: built-in Backup or OneDrive backup for the Desktop/Documents folders
- Mac: Time Machine
- Linux: `rsync`

### Further reading

Official Obsidian documentation: [Back up your Obsidian files](https://help.obsidian.md/backup)
