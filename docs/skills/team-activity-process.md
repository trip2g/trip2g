---
name: team-activity-process
description: Claude Code Stop hook script — records agent activity snapshots to trip2g after every turn. Use when setting up automatic team activity reporting or debugging why snapshots aren't appearing.
---

# team-activity-process

A Python script that fires on every Claude Code `Stop` hook, reads the session transcript, and writes a timestamped activity snapshot to the local vault — and optionally pushes it to a trip2g instance.

Install path: `~/.claude/skills/team-activity-process/process.py`  
Source: [docs/skills/team-activity-process.py](team-activity-process.py)

## When to use

- Setting up automatic activity reporting for a team (see [[agent-team-reporting]] in thoughts).
- Debugging why snapshots aren't appearing — run manually with a test payload.
- Extending the script to add LLM summarization or custom extraction logic.

## How it works

```
Stop hook fires
  → reads stdin JSON: {session_id, transcript_path, cwd}
  → walks up from cwd to find <vault>/team_activity/_projects.md
  → matches cwd against projects list → team_name + visibility
  → reads .state.json → last_snapshot_at + last_git_commit
  → if last snapshot < 10 min ago → exit (throttle)
  → git log since last_git_commit → new commits
  → transcript tail → tool calls/tasks since last_snapshot_at
  → if nothing new → exit
  → writes team_activity/{team}/YYYY-MM-DDTHH-MM.md with commits + actions
  → updates team_activity/{team}/report.md
  → updates .state.json (new timestamp + HEAD commit hash)
  → if visibility != private → pushNotes to trip2g (async, fails silently)
```

Throttle interval is configurable via `TEAM_ACTIVITY_THROTTLE_MINUTES` env var (default: 10).

## Hook config (`~/.claude/settings.json`)

```json
{
  "hooks": {
    "Stop": [{
      "matcher": "*",
      "hooks": [{
        "type": "command",
        "command": "python3 ~/.claude/skills/team-activity-process/process.py",
        "async": true,
        "timeout": 15
      }]
    }]
  }
}
```

`async: true` — never blocks Claude. Network errors fail silently.

## Snapshot format

```markdown
---
session_id: abc123
cwd: /home/alex/projects/trip2g
timestamp: 2026-05-23T14:30:00+00:00
---

## 2026-05-23 14:30 UTC

**Project:** [[trip2g/projects/trip2g]]

**Tasks:**
- Fix form submission filter

**Recent actions:**
- Bash: `go test ./internal/graph/...`
- Edit: `internal/graph/form_submits_filter.go`
```

## Project routing (`team_activity/_projects.md`)

```yaml
---
user: alexes
display_name: Alexey
projects:
  - team_name: trip2g
    cwd_patterns:
      - ~/projects2/trip2g
    visibility: team      # pushed to trip2g
  - team_name: personal
    cwd_patterns:
      - "*"
    visibility: private   # local only
---
```

First matching pattern wins. Wildcard `*` should be last.

## Auth resolution order

For `api_key` and `api_url`, first non-empty wins:

1. `TRIP2G_API_KEY` / `TRIP2G_API_URL` env vars
2. `syncDirs[0]` in nearest `.obsidian/plugins/trip2g/data.json`

## Manual invocation

```bash
echo '{"transcript_path":"/path/to/transcript.jsonl","cwd":"/your/project","session_id":"test"}' \
  | python3 ~/.claude/skills/team-activity-process/process.py
```

If a timestamped file appears in `team_activity/{team}/` — it's working.

## Secret redaction

Before writing, the script strips common secret patterns: API keys, tokens, JWTs, AWS keys, GitHub tokens, long hex and base64 strings. Redacted values are replaced with `[REDACTED]` markers.

## Related

- Companion skills: `team-activity-setup` (first-time setup), `team-activity-sync` (weekly goals review)
- Thoughts: [[ru/thoughts/agent-team-reporting]], [[en/thoughts/agent-team-reporting]]
