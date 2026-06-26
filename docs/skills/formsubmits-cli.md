---
name: formsubmits-cli
description: Use when triaging or processing admin form submissions from the shell or an agent — list, inspect, mark processed via scripts/formsubmits.py, no UI needed.
---

# formsubmits CLI

A thin CLI wrapper over the `admin.formSubmits` query and `markFormSubmitProcessed` mutation. Lives at `scripts/formsubmits.py`. Same auth model as `scripts/trip2g-preview.mjs`: walks up from cwd looking for `.obsidian/plugins/trip2g/data.json`, or reads `TRIP2G_API_KEY` / `TRIP2G_API_URL`.

## When to use

- An agent or operator needs to scan unprocessed form submissions across the site without opening the admin UI.
- You want to verify the new pagination/filter pipeline end-to-end (server up, API key in place).
- You need a scriptable way to mark submissions processed after handling them externally (email reply, manual follow-up).

## When not to use

- For batch back-office work (use SQL or a proper admin tool).
- Through public API surfaces — this hits `/graphql` with an admin-scoped API key, treat the key as a secret.

## Commands

| Command | What it does |
|---|---|
| `list` | Page through submissions. Supports all filter fields. |
| `show <id>` | Print one submission with all field values. Looks within the first 200 results — narrow with filters if needed. |
| `count` | Print `totalCount` for the given filter. Sets `limit=0` so the server skips loading nodes. |
| `mark <id>` | Run `markFormSubmitProcessed`. Optional `--comment`. |

Global flags: `--api-key`, `--api-url` (otherwise resolved from env / data.json).

## Filter flags (`list` and `count`)

| Flag | Maps to |
|---|---|
| `--note-path-id N` | `filter.notePathId` |
| `--form-id NAME` | `filter.formId` |
| `--status pending|visible|hidden` | `filter.status` |
| `--unprocessed` | `filter.processed = false` |
| `--processed` | `filter.processed = true` |
| `--created-after RFC3339` | `filter.createdAt.gte` |
| `--created-before RFC3339` | `filter.createdAt.lte` |
| `--limit N` | `filter.limit` (default 50, server caps at 200) |
| `--offset N` | `filter.offset` |

`--unprocessed` and `--processed` are mutually exclusive.

## Examples

```bash
# Newest 20 unprocessed across all forms
scripts/formsubmits.py list --unprocessed --limit 20

# Just the count of unprocessed
scripts/formsubmits.py count --unprocessed

# Submissions for one note, verbose (with all field values)
scripts/formsubmits.py list --note-path-id 4281 -v

# Time window for a specific form
scripts/formsubmits.py list \
  --form-id contact \
  --created-after 2026-05-01T00:00:00Z \
  --created-before 2026-05-31T23:59:59Z

# Inspect one submission
scripts/formsubmits.py show 17

# Mark processed with a note
scripts/formsubmits.py mark 17 --comment "answered via email"

# Raw JSON for piping into jq
scripts/formsubmits.py list --unprocessed --json | jq '.nodes[].id'
```

## Agent workflow: "process new submissions"

Typical loop an agent can drive without human input:

1. `count --unprocessed` — bail early if zero.
2. `list --unprocessed --limit 50 -v` — get a page with field values.
3. For each submission: decide (spam / answer / ignore), then `mark <id> --comment "..."`.
4. Re-run `count --unprocessed` — assert it dropped by the number marked.

## Auth resolution order

For each of `api_key` and `api_url`, the first non-empty wins:

1. `--api-key` / `--api-url` flag
2. `TRIP2G_API_KEY` / `TRIP2G_API_URL` env var
3. `syncDirs[0]` in the nearest `.obsidian/plugins/trip2g/data.json`
4. (URL only) fallback `http://localhost:8081`

The API key must belong to an admin user — `admin.formSubmits` is admin-gated.

## Exit codes

| Code | Meaning |
|---|---|
| 0 | Success |
| 1 | GraphQL errors, missing API key, submission not found in `show`, or `ErrorPayload` returned from `mark` |

## Related

- Plan: `.omc/plans/2026-05-22-form-submits-pagination.md` — the schema/resolver work this CLI exercises.
- Code: `internal/graph/schema.resolvers.go` (resolver), `internal/graph/form_submits_filter.go` (filter builder), `queries.read.sql` (`ListFormSubmits`, `CountFormSubmits`).
- Sibling CLI: `scripts/trip2g-preview.mjs` — same auth pattern, zero-dependency node ESM style.
