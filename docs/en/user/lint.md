---
title: "Linting your vault"
free: true
lang: en
lang_redirect: "[[ru/user/lint]]"
---

`trip2g lint <dir>` scans a folder of markdown notes and prints broken wikilinks and other problems. No server, no database: just point it at a directory and run.

### What it checks

The linter runs the same link-resolution engine the server uses when publishing your vault. It flags:

- **Broken wikilinks**: a `[[note-name]]` that points to a note that does not exist in the folder.
- **Cross-language leaks**: a bare `[[note-name]]` in a `ru/` note that resolves to an `en/` page (or vice versa). This usually means a translation is missing and the link quietly crossed over to the other language.
- **Broken images**: `![[image.png]]` references where the image file is absent.
- **Layout render errors**: problems in `_layouts/` Jet templates that would surface as broken pages on publish.

### Reading the output

Each finding is one line:

```
path/to/note.md:0: info broken link: some-slug
path/to/note.md:0: warning cross-language wikilink leak: [[Foo]] resolves to /en/foo (note lang=ru, link lang=en)
```

The format is `file:line: level message`. Line is always `0` because wikilink resolution does not track character positions. The level is `info`, `warning`, or `critical`.

**Exit codes:**

| Code | Meaning |
|------|---------|
| 0 | Clean, or only `info`-level findings (advisory) |
| 1 | At least one `warning` or `critical` finding |
| 2 | Usage error or the directory could not be loaded |

`info` findings are printed but do not fail the run. `warning` and above do.

### Basic usage

```bash
trip2g lint ./my-vault
```

A clean vault produces no output and exits 0. A vault with problems prints each finding and exits 1.

### Real example

A knowledge base built from concept notes had 24 links pointing at notes that lived in a different vault instance. The linter surfaced them all as broken links:

```
concepts/agency.md:0: info broken link: self-organization
concepts/cognition.md:0: info broken link: extended-mind
concepts/memory.md:0: info broken link: situated-cognition
...
```

The fix was to replace those internal links with a "see also" section pointing at the external source. After the fix, `trip2g lint` exited clean.

### Baseline: accept known findings, fail only on new ones

If your vault already has some broken links you cannot fix right now, a baseline lets CI pass on the existing problems and fail only when new ones appear.

**Step 1.** Generate a baseline from the current state:

```bash
trip2g lint --generate-baseline .lint-baseline ./my-vault
```

This writes the current findings to `.lint-baseline` and exits 0. Commit the baseline file.

**Step 2.** In CI, pass the baseline:

```bash
trip2g lint --baseline .lint-baseline ./my-vault
```

Findings that were already in the baseline are suppressed. Only new findings are printed, and only new `warning`-level findings cause exit 1.

When you fix a baselined issue, re-generate the baseline so the file shrinks. Treat a growing baseline as a warning sign.

### Using lint in CI

GitHub Actions example:

```yaml
- name: Lint vault
  run: trip2g lint --baseline .lint-baseline ./docs
```

For a stricter setup with no baseline (every finding is a failure):

```yaml
- name: Lint vault
  run: trip2g lint ./docs
```

### Pre-commit hook

To catch problems before committing:

```bash
# .git/hooks/pre-commit
trip2g lint --baseline .lint-baseline ./docs
```

Make the hook executable: `chmod +x .git/hooks/pre-commit`.

### Notes on image warnings

The linter does not resolve image assets: it has no access to your object storage or local asset folders. Any `![[image.png]]` reference will produce an `info` finding if the image file is not a note in the same directory. These are advisory (`info` level) and do not fail the run. Use the baseline to suppress them if they are false positives in your setup.
