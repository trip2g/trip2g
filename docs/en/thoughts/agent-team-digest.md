---
title: The agent keeps a team journal
free: true
lang_redirect: "[[ru/thoughts/agent-team-digest]]"
---

Every morning the agent reads what team members worked on yesterday and writes a daily note. Every Monday, a weekly summary. Nobody enters anything manually — the data comes from each member's trip2g instance via MCP.

## The job

When several people work with agents in parallel, a blind spot forms: everyone sees their own work but not their neighbor's. To coordinate, you either interrupt someone to ask, or dig through commits.

The digest agent removes both options. It reads the snapshots itself and assembles the picture.

## Daily note

Each morning the agent visits members listed in `team_activity/_projects.md`, makes an MCP request to their trip2g instance, reads yesterday's snapshots, and writes a shared note:

```markdown
## 2026-05-23

**Alexey** — [[trip2g/projects/trip2g-infra]]
- Added steel-browser Nomad job
- Tests: `nomad job validate steel-browser.nomad.hcl`

**Nikolai** — [[nicksenin/projects/knowlu]]
- Refactored transcript parser
- Updated prompts for case-extraction
```

The file goes to `team_activity/digests/YYYY-MM-DD.md`.

## Weekly note

On Monday the agent aggregates seven daily notes and maps them against last week's goals:

```markdown
## Week 21

**Alexey**
- Done: steel-browser infra, cross-node backup
- Carries over: grafana MCP endpoint

**Nikolai**
- Done: knowlu parser v2, 14 new cases
- Carries over: Hermes integration

**Pattern:** both worked with MCP — natural sync point
```

## Team page

Digests are for internal use. But the same agent can read each member's `whoami.md` once a day and update a public team page on the site. Each person edits only their own file — the agent assembles the current aggregate overnight.

## Coordination without meetings

The daily note is also a signal. Before picking up a task, Nikolai's agent reads yesterday's digest: if Alexey is already working on that module, the task moves to the queue. Agent-to-agent coordination through published data — no shared scheduler, no Jira, no sync call.

## Running it

The digest agent runs on a cron schedule via `CronCreate` in Claude Code:

```
daily at 07:00 → read member MCPs → write digest
weekly Mon 08:00 → aggregate daily notes → write weekly summary
```

Members are discovered via federated search using the `agent-team` protocol — see [[en/thoughts/agent-team-reporting|The agent reports to the team]].

## Related

- [[en/thoughts/agent-team-reporting|The agent reports to the team]]
- [[en/thoughts/knowledge-as-a-service|Knowledge-as-a-Service]]
- [[en/thoughts/ai-changes-rules|AI changes the rules]]
