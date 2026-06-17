---
title: Advanced
free: true
lang_redirect: "[[ru/user/advanced]]"
---

Advanced features for custom domains, SEO, backups, and command-line access.

### Custom domains

Any note can be served at a custom domain or subdomain. Add a `route` key to the note's frontmatter:

```yaml
---
route: mysite.com/
---
```

After adding the route, point your domain at the trip2g server with a CNAME DNS record. HTTPS certificates are issued automatically on first request.

#### Route variants

```yaml
# Note becomes the root of the domain
route: mysite.com/

# Note at a specific path on the domain
route: mysite.com/about

# Alias on the main domain (no custom domain needed)
route: /blog

# Multiple routes at once
routes:
  - mysite.com/
  - mysite.com/home
```

`www.mysite.com` and `mysite.com` are treated as identical. `route` and `slug` work independently — you can use both on the same note.

#### Multiple independent sites

One trip2g account can serve multiple separate sites. Domains are isolated: notes on `mysite.com` don't appear at their path on the main domain, and vice versa.

A common pattern is to apply routes across a whole folder using [[en/user/frontmatter-patches|frontmatter patches]] instead of editing each note:

```
docs/**.md → { route: "docs.mysite.com" }
```

<!-- TODO: verify with product — confirm frontmatter_patches slug/link -->

#### Cross-domain wikilinks

Wikilinks resolve automatically based on where the reader is. If the target note is on a different custom domain, the link becomes a full URL (`https://other.com/path`). If it's on the same domain, it stays relative.

#### DNS troubleshooting

- CNAME changes can take up to 48 hours to propagate. Check with `dig yourdomain.com` or [dnschecker.org](https://dnschecker.org).
- Use `route: mysite.com/` (with trailing slash) to make a note the root of the domain. Without the slash, the note is accessible at its normal permalink on that domain.
- Each custom domain gets its own `/sitemap.xml` containing only its own notes.
- Session cookies are scoped to the domain. For private notes on a custom domain, the reader must log in separately on that domain. For open content, add `free: true`.

### SEO

trip2g generates HTML server-side and serves pages from cache — the same behaviour as a static site generator. Search engines see the same fully-rendered HTML that human visitors do.

#### What trip2g handles automatically

- **Sitemap** — `/sitemap.xml` includes all public pages and updates on changes
- **Robots.txt** — private pages are excluded from indexing
- **Clean URLs** — `/docs/seo` instead of `/page.php?id=123`
- **Meta tags** — `<title>` comes from the note title; `<meta name="description">` comes from the `description` frontmatter field. If you don't set it, no description is emitted — so write one
- **Open Graph** — social preview tags work out of the box; set the preview image with `og_image` (a `[[link]]` or path) in frontmatter, otherwise the first image in the note body is used
- **Fast page loads** — no client-side JavaScript frameworks

#### Adding analytics

Connect Google Analytics, Fathom, or any other analytics service by pasting the script tag into site settings. It runs on every page.

#### Paywalled content and search indexing

Search engines see the same content as an unauthenticated visitor: the title, description, and a content preview. Full text stays behind the paywall. This matches how The New York Times, Medium, and Substack handle paywalled SEO — the page is indexed and discoverable, but full access requires a subscription.

#### Wikilinks as internal links

`[[note]]` wikilinks become standard `<a href="...">` HTML elements. Search engines follow them normally. Anchor links (`[[note#section]]`) resolve to heading IDs.

#### Accelerating indexing

Submit your site to Google Search Console or Yandex Webmaster and send the sitemap for indexing. New pages typically appear in search results within 2–7 days.

### OAuth login

Readers can authenticate with Google or GitHub to access private notes. Enable OAuth in site settings and configure which provider to use.

<!-- TODO: verify with product — confirm exact OAuth setup steps in admin panel -->

### Backups

trip2g stores your notes where Obsidian stores them — locally on your device. Set up independent backups to protect against accidental deletion or device loss.

**Sync services are not backups.** Obsidian Sync, iCloud, Dropbox, and OneDrive replicate deletions. A true backup is a separate copy that doesn't change when you change the original.

#### Recommended approaches

**Obsidian Git plugin** — commits each change to a Git repository. Full version history, easy rollback to any point.

**Local Backup plugin** — copies the vault to a chosen folder on a schedule. Configure archiving and retention period.

**Cloud storage** — copy your vault folder to Dropbox, Google Drive, or Backblaze.

**System tools** — Time Machine on macOS, rsync on Linux, OneDrive backup on Windows.

For detailed Obsidian backup guidance, see the [official Obsidian documentation](https://help.obsidian.md/backup).

### CLI

The trip2g CLI syncs markdown files from any directory to the server — without Obsidian, from a terminal or CI/CD pipeline.

#### Installation

```bash
curl -L -o trip2g-sync.mjs \
  https://github.com/trip2g/obsidian-sync/releases/download/0.3.5/trip2g-sync.mjs
```

#### Basic usage

```bash
# One-way sync (local → server)
node trip2g-sync.mjs --folder ./vault --api-key YOUR_API_KEY

# With explicit server URL
node trip2g-sync.mjs --folder ./vault --api-key YOUR_API_KEY \
  --api-url https://yoursite.com/graphql

# Two-way sync (pull changes from server too)
node trip2g-sync.mjs --folder ./vault --api-key YOUR_API_KEY --two-way

# Dry run (show what would happen, make no changes)
node trip2g-sync.mjs --folder ./vault --api-key YOUR_API_KEY --dry-run
```

Use environment variables instead of flags: `API_KEY` and `ENDPOINT`.

#### Key parameters

| Parameter | Short | Purpose |
|-----------|-------|---------|
| `--folder <path>` | `-f` | Directory to sync (required) |
| `--api-url <url>` | `-u` | GraphQL endpoint |
| `--api-key <key>` | `-k` | API key |
| `--two-way` | `-2` | Bidirectional sync |
| `--conflict-resolution <mode>` | `-c` | `local`, `remote`, `skip`, or `fail` |
| `--meta <key=value>` | `-m` | Inject frontmatter field into all synced files |
| `--dry-run` | `-n` | Preview changes without applying them |
| `--verbose` | `-v` | Verbose output |

#### Injecting frontmatter with --meta

`--meta` adds a frontmatter field to every synced file at upload time without modifying local files. The main use case is combining multiple repositories into one documentation portal:

```bash
# Backend repo
node trip2g-sync.mjs --folder ./docs --meta subgraph=backend-team

# Frontend repo
node trip2g-sync.mjs --folder ./docs --meta subgraph=frontend-team
```

All three publish to the same endpoint. The `subgraph` field lets you control access per team.

#### Conflict resolution modes

| Mode | Behaviour |
|------|-----------|
| `local` | Local version wins (default) |
| `remote` | Server version wins |
| `skip` | Skip conflicting files |
| `fail` | Stop on first conflict — useful for CI |

#### GitHub Actions example

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

The CLI creates a `.sync-state.json` file in the synced folder to track file hashes. Add it to `.gitignore` if you don't want to share sync state across machines.

### Related

- [[en/user/templates]] — control page layout with templates
- [[en/user/webhooks]] — automate actions when notes change
