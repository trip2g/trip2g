---
title: "CLI sync tool"
free: true
lang: en
lang_redirect: "[[ru/user/cli]]"
---

A command-line tool for syncing markdown files to your trip2g server. Works without Obsidian — from a terminal or a CI/CD pipeline.

The CLI and the Obsidian plugin share the same sync core. Whatever the plugin can publish, the CLI can too.

### When you need the CLI

- **CI/CD pipelines.** Publish documentation automatically on every push to your repository.
- **Multiple repositories.** Each team syncs its own folder with its own metadata to a shared portal.
- **Migrations.** Bulk-update frontmatter fields across many files at once.
- **Scripts.** Integrate sync into other tools or automation workflows.

For a worked example of documentation-from-a-repository, see the use cases section.

### Installation

```bash
# Download the prebuilt file
curl -L -o trip2g-sync.mjs \
  https://github.com/trip2g/obsidian-sync/releases/download/0.3.5/trip2g-sync.mjs

# Or build from source
cd obsidian-sync
npm install
npm run build:cli
# CLI is built to dist/trip2g-sync.mjs
```

### Usage

```bash
# Basic sync
node trip2g-sync.mjs --folder ./vault --api-key YOUR_API_KEY

# Specify a server
node trip2g-sync.mjs --folder ./vault --api-key YOUR_API_KEY --api-url https://yoursite.com/graphql

# Two-way sync (pull changes from the server as well)
node trip2g-sync.mjs --folder ./vault --api-key YOUR_API_KEY --two-way

# Dry run (show what would happen, make no changes)
node trip2g-sync.mjs --folder ./vault --api-key YOUR_API_KEY --dry-run
```

### Environment variables

You can supply credentials via environment variables instead of command-line arguments:

```bash
export API_KEY=your_api_key
export ENDPOINT=https://yoursite.com/graphql

node trip2g-sync.mjs --folder ./vault
```

### Parameters

| Flag | Short | Description |
|------|-------|-------------|
| `--folder <path>` | `-f` | Folder to sync (required) |
| `--api-url <url>` | `-u` | GraphQL endpoint (default: `$ENDPOINT` or `http://localhost:8081/graphql`) |
| `--api-key <key>` | `-k` | API key (default: `$API_KEY`) |
| `--two-way` | `-2` | Two-way sync |
| `--conflict-resolution <mode>` | `-c` | Conflict mode: `local`, `remote`, `skip`, `fail` (default: `local`) |
| `--meta <key=value>` | `-m` | Inject a frontmatter field into every synced file (repeatable) |
| `--verbose` | `-v` | Verbose output |
| `--dry-run` | `-n` | Show plan without executing |
| `--help` | `-h` | Show help |

### --meta and subgraph

`--meta` injects frontmatter fields into every file the CLI sends, without touching the local files. The primary use case is merging several repositories into one documentation portal with per-team access control.

**Example: three repositories, one documentation site**

```bash
# Backend team's repository
node trip2g-sync.mjs --folder ./docs --meta subgraph=backend-team

# Frontend team's repository
node trip2g-sync.mjs --folder ./docs --meta subgraph=frontend-team

# DevOps repository
node trip2g-sync.mjs --folder ./docs --meta subgraph=devops --meta team=infra
```

All three repositories publish to the same endpoint. The `subgraph` field tags each document — you then configure access so the backend team sees only `subgraph=backend-team`, while DevOps sees everything.

**How injection works:**

1. **File with no frontmatter** — a new block is created on the server side:
   ```markdown
   ---
   subgraph: backend-team
   ---
   # Content
   ```

2. **File with existing frontmatter** — the field is added or overwritten:
   ```markdown
   ---
   title: API Reference
   subgraph: backend-team    ← injected
   ---
   ```

3. **Local files are not modified** — the injection happens only in what is sent to the server.

### Conflict resolution

| Mode | Behavior |
|------|----------|
| `local` | Local version wins, overwrites the server (default) |
| `remote` | Server version wins, overwrites the local file |
| `skip` | Skip conflicting files, continue with the rest |
| `fail` | Stop with an error on the first conflict (recommended for CI) |

### GitHub Actions example

```yaml
on:
  push:
    branches: [main]
    paths: ['docs/**']

jobs:
  sync:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: |
          curl -L -o trip2g-sync.mjs \
            https://github.com/trip2g/obsidian-sync/releases/download/0.3.5/trip2g-sync.mjs
          node trip2g-sync.mjs \
            --folder ./docs \
            --api-url https://docs.yourcompany.com/graphql \
            --api-key ${{ secrets.DOCS_API_KEY }} \
            --meta subgraph=project-name \
            --conflict-resolution fail
```

Set `--conflict-resolution fail` so the workflow fails loudly if the server and the repository have diverged rather than silently overwriting either side.

### Sync state file

The CLI creates a `.sync-state.json` file inside the synced folder. It stores file hashes and timestamps to track what has changed between runs.

Add it to `.gitignore` if you do not want to share sync state between machines:

```
.sync-state.json
```

### Source code

[GitHub: trip2g/obsidian-sync](https://github.com/trip2g/obsidian-sync/tree/master/src/sync/cli)
