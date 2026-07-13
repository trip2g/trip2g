---
title: "Git Access"
free: true
lang_redirect: "[[ru/user/git]]"
---

Your trip2g site is also a git repository. Clone it, edit notes with any editor or agent, push back — the server applies your commits to the live site.

```bash
git clone https://your-site.com/_system/git my-site
# Username: user
# Password: <git-token>
```

No plugin, no special CLI. Anything that speaks git — your laptop, a CI job, a coding agent — can read and write the site.

### When to use it

- **Agents and CI.** Claude Code, Codex, or a pipeline clones the vault, edits markdown, commits, pushes. Standard git tooling, standard review flow.
- **Batch edits.** Rename a tag across 200 notes with `sed`, check the diff, push once.
- **A full local copy.** `git clone` gives you every note and asset in one command; `git pull` keeps it fresh.

For day-to-day writing in Obsidian, the [[en/user/two-way-sync|sync plugin]] is still the better fit — git is for tooling, automation, and bulk work.

### Setup: create a Git token

1. Open `https://your-site.com/admin` → **Integrations** → **Git tokens**.
2. Create a token with a description. The value is shown **once** — save it.

Authentication is HTTP Basic over the clone URL: the username is always `user`, the password is the token. For non-interactive use, embed it:

```bash
git clone https://user:YOUR_TOKEN@your-site.com/_system/git my-site
```

### What's in the repository

- Every visible note (`.md` and `.html`) plus note assets (images, files), at the same paths you see in your vault.
- A single branch: `master`. That's the only branch the server accepts — pushes to any other branch are rejected.
- Server-side edits (plugin sync, in-browser editor, agents via GraphQL) appear as `server sync` commits.

### Pushing changes back

```bash
cd my-site
vim blog/post.md
git add -A
git commit -m "fix typo"
git push
```

On push the server takes the diff of your commits and applies it to the site:

- Changed or added `.md` / `.html` files update the corresponding notes.
- Deleted `.md` / `.html` files hide the notes on the site.
- If applying fails, the push is rolled back — the repository never diverges from the site.

Only fast-forward pushes are accepted. If someone edited the site since your last pull, the push is rejected — rebase and retry:

```bash
git pull --rebase
git push
```

### The database is canonical — git is a mirror

The site's database is the source of truth. The git repository is a mirror rebuilt from it on every clone, pull, and push, so you always fetch the current state. Consequences:

- **History is coarse.** Edits made through the plugin or editor are batched into `server sync` snapshot commits. Don't expect a full per-edit audit trail — git history here is not the site's edit history.
- **Assets are read-only over git.** Pushing an image or other binary file does nothing: the server applies only `.md` and `.html` changes. Upload assets through the Obsidian plugin or the [[en/user/cli|sync CLI]]; they then appear in the mirror.
- **No force pushes, no branches.** The server enforces fast-forward-only on `master`.

### Example: an agent editing the site

```bash
git clone https://user:$GIT_TOKEN@your-site.com/_system/git site
cd site
# ... agent edits markdown files ...
git add -A
git commit -m "agent: update FAQ"
git push   # changes are live once the push succeeds
```

### Example: a mirror that stays fresh

```bash
# in cron, next to your other backups
cd /backups/site && git pull
```

Each pull materializes the latest site state into the mirror — a plain-files copy of every note and asset, restorable with any git client.
