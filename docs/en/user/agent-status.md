---
title: "Auto-broadcast your work status to the team"
free: true
lang_redirect: "[[ru/user/agent-status]]"
---

Your team sees what each person is working on without asking. A Claude Code hook captures the session summary when you finish working, redacts it, runs it through an LLM to produce a structured status note, and syncs that note into the shared vault. Teammates query the hub and get everyone's current status in one call.

This removes the need for daily standups, manual status updates, and polling a shared doc.

## How it works

```
Claude Code session ends
       │
       ▼
SessionEnd hook fires
       │
       ▼
Redact: strip keys, secrets, client paths
       │
       ▼
LLM summarizes only new transcript lines (cursor-tracked)
       │
       ▼
Writes status/{your-name}.md to vault
       │
       ▼
trip2g-sync --watch picks it up → server
       │
       ▼
Hub federates it → teammates query federated_search
```

Each step is a small script. The hook wires them together.

## Why trigger on session end, not a cron

The original approach (a cron job every 2-4 hours) works but has a mismatch: the cron fires on a clock tick, not when you actually finish something. You get stale status between ticks, and redundant summaries if the session is still running.

A `SessionEnd` hook fires the moment Claude Code closes. The status reflects exactly what you just did, freshness measured in seconds not hours. It also costs nothing to run during idle time.

Keep a cron as a fallback for sessions that crash or are killed without a clean stop. Set it to 3-4 hours and have it skip the run if a status note was already written in the last hour.

```bash
# .claude/hooks/SessionEnd.sh  (simplified)
#!/usr/bin/env bash
set -euo pipefail

VAULT_DIR="${TEAM_VAULT_DIR:-$HOME/team-vault}"
MY_NAME="${TEAM_MEMBER_NAME:-$(whoami)}"
SESSION_LOG="$1"   # path passed by Claude Code to the hook

# 1. read cursor, extract only new lines
CURSOR_FILE="$VAULT_DIR/.status-cursor-$MY_NAME"
CURSOR=$(cat "$CURSOR_FILE" 2>/dev/null || echo "0")
NEW_LINES=$(tail -n +"$CURSOR" "$SESSION_LOG")
NEW_CURSOR=$(wc -l < "$SESSION_LOG")

# 2. redact before the LLM sees anything
CLEAN=$(echo "$NEW_LINES" | sed -E \
  's/(API_KEY|SECRET|PASSWORD|Bearer |sk-|ghp_)[^\s"]*/[REDACTED]/g')

# 3. summarize
STATUS_JSON=$(echo "$CLEAN" | llm -s \
  'Output JSON only: {"status":"one sentence","area":"feat|bug|refactor|blocked|exploring","confidence":"high|low"}')

# 4. write note
cat > "$VAULT_DIR/status/$MY_NAME.md" <<EOF
---
title: "$MY_NAME — status"
free: false
team-share: true
updated: "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
---
$(echo "$STATUS_JSON" | jq -r '"**Status:** \(.status)\n**Area:** \(.area)\n**Confidence:** \(.confidence)"')
EOF

# 5. advance cursor
echo "$NEW_CURSOR" > "$CURSOR_FILE"
```

Register the hook in `.claude/settings.json`:

```json
{
  "hooks": {
    "Stop": [
      { "type": "command", "command": "bash .claude/hooks/SessionEnd.sh $CLAUDE_SESSION_LOG" }
    ]
  }
}
```

`Stop` fires when the session ends cleanly. Add a `PostToolUse` hook on long sessions if you want mid-session updates, but session-end coverage is usually enough.

## Redact before the LLM

The LLM is not a reliable secret detector. It will redact what you ask it to redact in the prompt, but it cannot know what you consider sensitive if you did not tell it. Run the regex strip first, before the transcript reaches the model.

Minimum patterns to strip:

```
API_KEY|SECRET|PASSWORD|Bearer |sk-|ghp_|t2g_
```

Add client folder names and internal project codenames as a second pass if your work touches those.

## Structured output

Ask the LLM for three fields, not prose:

| Field | Values | Use |
|-------|--------|-----|
| `status` | One sentence | What you finished or where you are |
| `area` | `feat`, `bug`, `refactor`, `blocked`, `exploring` | Lets teammates filter at a glance |
| `confidence` | `high`, `low` | Was the session productive or exploratory? |

When `confidence` is `low` (the session was mostly reading, debugging blindly, or exploring unfamiliar territory), write `exploring` in `area` rather than guessing a completion. A fabricated "fixed the auth bug" when you spent the session reading logs is worse than "exploring auth issues."

## Opt in per session

Add `team-share: true` to the status note frontmatter. Without it, the note stays private even if it syncs to the vault. This gives each person control: a session on a client project does not broadcast automatically.

You can also skip the hook entirely for a session by setting an env var:

```bash
TEAM_SHARE_OFF=1 claude
```

Check for it in the hook script and exit early if set.

## Team view via the hub

Teammates do not read per-person files directly. They query the federation hub:

```
federated_search(query="what is everyone working on", kb_id="team-status")
```

The hub returns summaries from all synced status notes. One call, everyone's status.

Set this up with a KB-note in the shared vault:

```yaml
---
title: "Team status"
free: false
mcp_federation_kb_url: https://your-hub.example.com/_system/mcp
mcp_federation_kb_id: team-status
---
Use when: checking what teammates are working on or who is blocked.
```

For the full federation setup, see [[en/user/federation]].

## Setup checklist

The `team-activity` OMC skill handles most of the wiring. Run:

```
/oh-my-claudecode:team-activity setup
```

That creates the hook file, registers it in `.claude/settings.json`, writes the initial cursor file, and adds a `hub.md` KB-note pointing at the team hub. You still need to:

1. Set `TEAM_VAULT_DIR` and `TEAM_MEMBER_NAME` in your shell profile.
2. Confirm the shared vault has `trip2g-sync --watch` running against it. See [[en/user/agent-memory]] section "4a. Continuous sync."
3. Exchange HMAC secrets with teammates if using a private hub peer. See [[en/user/federation]] ("Adding a private peer").

## Staleness and limits

A status note is as fresh as the last session end or cron tick. If someone is mid-session, their note reflects where they were when their previous session closed. This is inherent to the design: you cannot summarize a session that has not ended.

The cron fallback (3-4 hours) catches sessions left open overnight. It will re-summarize from the same cursor position. If no new lines exist since the last run, the output will be "no new activity," which is accurate.

What this does not do: real-time activity tracking, screen sharing, or any form of surveillance. The hook summarizes completed work, not ongoing keystrokes.

## Related

- [[en/user/agent-memory]]: the vault and sync setup this note builds on
- [[en/user/federation]]: how the hub federates team status across instances
- [[en/user/webhooks]]: change webhooks for automation triggered by note updates
