---
title: The agent reports to the team on its own
free: true
lang_redirect: "[[ru/thoughts/agent-team-reporting]]"
---

When you work with an agent all day, the team sees nothing. A commit might appear by evening — if it appears at all. What the agent was doing for six hours is a mystery. This skill closes that gap: after every turn, the agent automatically writes a short report and sends it to trip2g.

## How it works

Claude Code has a `Stop` hook — it fires every time the agent finishes a turn. The skill attaches a single Python script to that hook:

```
Stop hook → process.py reads the transcript tail
          → extracts bash commands, edited files, created tasks
          → writes a snapshot to team_activity/{team}/YYYY-MM-DDTHH-MM.md
          → updates report.md
          → if the project is not private — pushes to trip2g
```

The script is async and never blocks the agent. Network errors are silently ignored.

## What the team sees

Each snapshot is a short timestamped note:

```markdown
## 2026-05-23 14:30 UTC

**Project:** [[trip2g/projects/trip2g]]

**Tasks:**
- Add form-submission filter

**Recent actions:**
- Bash: `go test ./internal/graph/...`
- Edit: `internal/graph/form_submits_filter.go`
- Edit: `internal/graph/form_submits_filter_test.go`
```

No interpretation — just facts from the transcript. Secrets are stripped by regex before anything is written.

## Project routing

The file `team_activity/_projects.md` controls what goes where:

```yaml
projects:
  - team_name: trip2g
    cwd_patterns:
      - ~/projects2/trip2g
    visibility: team      # pushed to trip2g
  - team_name: personal
    cwd_patterns:
      - "*"
    visibility: private   # local only
```

One pattern → one team. The wildcard `*` at the end catches everything else and keeps it private.

## Goals and whoami

Snapshots are a stream. On top of that sit two files the team reads first:

- `whoami.md` — who you are, timezone, work style, current focus
- `week_goals.md` — three goals for the week with checkboxes

Once a week the agent runs a short retrospective: which goals were met, what blocked progress, what the goals are for next week. Old goals are archived to `goals_history/`. Everything gets pushed to trip2g.

## Why this matters

**Coordination without meetings.** When Alexey's agent is working on the auth module, Nikolai's agent sees this via the trip2g MCP and doesn't pick up dependent tasks.

**Retrospective from facts.** A month later you can look at not "what was planned" but "what the agent was actually doing day by day."

**Trust in agents.** The team sees specific files and commands — not general words about progress, but actual lines of code. This reduces the anxiety of "the agent is working — doing something."

## Discovering teammates via federation

trip2g supports federated search. A planned feature: a discovery command that searches trip2g nodes by the protocol keyword `agent-team`. Any instance that publishes a `whoami.md` is discoverable — the file itself is the signal, no extra metadata needed.

Run the discovery command, find matching nodes, add them as external sources. Your agent starts seeing their `whoami.md` and `week_goals.md` alongside your local team's — no central registry, no manual coordination.

This turns the activity skill from a local reporting tool into a network: agents coordinating across organizations through federation.

## Setup

```bash
# Verify the skill works:
echo '{"transcript_path":"","cwd":"/your/project","session_id":"test"}' \
  | python3 ~/.claude/skills/team-activity-process/process.py
```

If a timestamped file appears in `team_activity/{team}/` — it's working. Full docs and source: [[skills/team-activity-process]].

## Related

- [[en/thoughts/ai-changes-rules|AI changes the rules]]
- [[en/thoughts/knowledge-bot|Knowledge bot]]
